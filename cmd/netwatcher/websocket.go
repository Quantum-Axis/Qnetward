package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"netwatcher/internal/ping"
	"netwatcher/internal/portscan"
	"netwatcher/internal/report"
	"netwatcher/internal/store"
	"netwatcher/internal/tcpping"
)

type wsRequest struct {
	Type        string `json:"type"`
	Host        string `json:"host"`
	Port        int    `json:"port,omitempty"`
	Ports       string `json:"ports,omitempty"`
	Count       int    `json:"count"`
	IntervalMs  int    `json:"interval_ms"`
	TimeoutMs   int    `json:"timeout_ms"`
	Size        int    `json:"size"`
	IPv6        bool   `json:"ipv6"`
	Concurrency int    `json:"concurrency,omitempty"`
}

type wsEvent struct {
	Type     string       `json:"type"`
	Protocol string       `json:"protocol,omitempty"`
	Target   string       `json:"target,omitempty"`
	Addr     string       `json:"addr,omitempty"`
	Port     int          `json:"port,omitempty"`
	Ts       int64        `json:"ts,omitempty"`
	Result   *jsonResult  `json:"result,omitempty"`
	Summary  *jsonSummary `json:"summary,omitempty"`
	Error    string       `json:"error,omitempty"`
}

type wsWriter struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

type wsSession struct {
	writer *wsWriter

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

func (w *wsWriter) WriteEvent(evt wsEvent) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.conn.SetWriteDeadline(time.Now().Add(wsWriteTimeout)); err != nil {
		return err
	}
	return w.conn.WriteJSON(evt)
}

func (s *wsSession) start(user store.User, req wsRequest) bool {
	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		return false
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	s.cancel = cancel
	s.done = done
	s.mu.Unlock()

	go func() {
		defer close(done)
		err := runWSRequest(ctx, s.writer, user, req)
		if err != nil && !errors.Is(err, context.Canceled) {
			_ = s.writer.WriteEvent(wsEvent{Type: "error", Error: err.Error()})
		}
		s.clear(done)
	}()
	return true
}

func (s *wsSession) stop() bool {
	s.mu.Lock()
	cancel := s.cancel
	s.mu.Unlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

func (s *wsSession) clear(done chan struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.done != done {
		return
	}
	s.cancel = nil
	s.done = nil
}

func (s *wsSession) close() {
	s.stop()
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return isAllowedWSOrigin(r, origin)
	},
}

func handleWS(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WebSocket upgrade error: %v\n", err)
		return
	}
	defer conn.Close()

	session := &wsSession{writer: &wsWriter{conn: conn}}
	defer session.close()

	conn.SetReadLimit(maxWSMessageBytes)
	conn.SetReadDeadline(time.Now().Add(wsReadTimeout))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(wsReadTimeout))
	})

	for {
		var req wsRequest
		if err := conn.ReadJSON(&req); err != nil {
			// Check if it's a normal close or an error
			if !websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				return // Normal close
			}
			fmt.Fprintf(os.Stderr, "WebSocket read error: %v\n", err)
			return // Abnormal close or error
		}
		req.Type = strings.ToLower(strings.TrimSpace(req.Type))
		if req.Type == "" {
			_ = session.writer.WriteEvent(wsEvent{Type: "error", Error: "type is required"})
			continue
		}

		conn.SetReadDeadline(time.Now().Add(wsReadTimeout))

		switch req.Type {
		case "stop":
			if !session.stop() {
				_ = session.writer.WriteEvent(wsEvent{Type: "error", Error: "no active run"})
			}
		case "ping":
			if !session.start(user, req) {
				_ = session.writer.WriteEvent(wsEvent{Type: "error", Error: "a run is already in progress"})
			}
		case "tcpping":
			if !session.start(user, req) {
				_ = session.writer.WriteEvent(wsEvent{Type: "error", Error: "a run is already in progress"})
			}
		case "portscan":
			if !session.start(user, req) {
				_ = session.writer.WriteEvent(wsEvent{Type: "error", Error: "a run is already in progress"})
			}
		default:
			_ = session.writer.WriteEvent(wsEvent{Type: "error", Error: "unknown type"})
		}
	}
}

func runWSRequest(ctx context.Context, writer *wsWriter, user store.User, req wsRequest) error {
	switch req.Type {
	case "ping":
		return streamPingWS(ctx, writer, user, req)
	case "tcpping":
		return streamTCPPingWS(ctx, writer, user, req)
	case "portscan":
		return streamPortScanWS(ctx, writer, req)
	default:
		return fmt.Errorf("unknown type")
	}
}

func streamPingWS(ctx context.Context, writer *wsWriter, user store.User, req wsRequest) error {
	release, ok := tryAcquireProbeSlot()
	if !ok {
		return writer.WriteEvent(wsEvent{Type: "error", Error: "server busy, try again"})
	}
	defer release()

	interval := time.Second
	if req.IntervalMs > 0 {
		interval = time.Duration(req.IntervalMs) * time.Millisecond
	}
	timeout := 2 * time.Second
	if req.TimeoutMs > 0 {
		timeout = time.Duration(req.TimeoutMs) * time.Millisecond
	}
	size := req.Size
	var err error
	req.Host, err = normalizeHost(req.Host)
	if err != nil {
		return writer.WriteEvent(wsEvent{Type: "error", Error: err.Error()})
	}
	cfg, err := normalizePingConfig(req.Count, interval, timeout, size, req.IPv6)
	if err != nil {
		return writer.WriteEvent(wsEvent{Type: "error", Error: err.Error()})
	}

	rep, err := ping.RunStreamContext(ctx, req.Host, cfg, func(res report.Result) error {
		ts := time.Now()
		storeResult(user.ID, "ping", req.Host, res.Addr, 0, res, ts)
		jr := toJSONResult(res)
		return writer.WriteEvent(wsEvent{
			Type:     "result",
			Protocol: "icmp",
			Target:   req.Host,
			Ts:       ts.UnixMilli(),
			Result:   &jr,
		})
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			summary := toJSONSummary(rep.Summary)
			return writer.WriteEvent(wsEvent{
				Type:     "stopped",
				Protocol: "icmp",
				Target:   req.Host,
				Addr:     rep.Addr,
				Summary:  &summary,
			})
		}
		return writer.WriteEvent(wsEvent{Type: "error", Error: err.Error()})
	}

	summary := toJSONSummary(rep.Summary)
	enqueueAlertEvaluation(user, "ping", req.Host, 0, rep.Summary, buildConfiguredDashboardURL("alerts"))
	return writer.WriteEvent(wsEvent{
		Type:     "summary",
		Protocol: rep.Protocol,
		Target:   rep.Target,
		Addr:     rep.Addr,
		Summary:  &summary,
	})
}

func streamTCPPingWS(ctx context.Context, writer *wsWriter, user store.User, req wsRequest) error {
	release, ok := tryAcquireProbeSlot()
	if !ok {
		return writer.WriteEvent(wsEvent{Type: "error", Error: "server busy, try again"})
	}
	defer release()

	host := req.Host
	port := req.Port
	if parsedHost, parsedPort, ok := splitHostPort(host); ok {
		host = parsedHost
		if port == 0 {
			port = parsedPort
		}
	}
	if port == 0 {
		port = 80
	}

	interval := time.Second
	if req.IntervalMs > 0 {
		interval = time.Duration(req.IntervalMs) * time.Millisecond
	}
	timeout := 2 * time.Second
	if req.TimeoutMs > 0 {
		timeout = time.Duration(req.TimeoutMs) * time.Millisecond
	}

	host, port, cfg, err := normalizeTCPPingRequest(host, port, req.Count, interval, timeout)
	if err != nil {
		return writer.WriteEvent(wsEvent{Type: "error", Error: err.Error()})
	}

	rep, err := tcpping.RunStreamContext(ctx, host, port, cfg, func(res report.Result) error {
		ts := time.Now()
		storeResult(user.ID, "tcpping", host, res.Addr, port, res, ts)
		jr := toJSONResult(res)
		return writer.WriteEvent(wsEvent{
			Type:     "result",
			Protocol: "tcp",
			Target:   host,
			Port:     port,
			Ts:       ts.UnixMilli(),
			Result:   &jr,
		})
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			summary := toJSONSummary(rep.Summary)
			return writer.WriteEvent(wsEvent{
				Type:     "stopped",
				Protocol: "tcp",
				Target:   host,
				Addr:     rep.Addr,
				Port:     port,
				Summary:  &summary,
			})
		}
		return writer.WriteEvent(wsEvent{Type: "error", Error: err.Error()})
	}

	summary := toJSONSummary(rep.Summary)
	enqueueAlertEvaluation(user, "tcpping", host, port, rep.Summary, buildConfiguredDashboardURL("alerts"))
	return writer.WriteEvent(wsEvent{
		Type:     "summary",
		Protocol: rep.Protocol,
		Target:   rep.Target,
		Addr:     rep.Addr,
		Port:     rep.Port,
		Summary:  &summary,
	})
}

func streamPortScanWS(ctx context.Context, writer *wsWriter, req wsRequest) error {
	release, ok := tryAcquireProbeSlot()
	if !ok {
		return writer.WriteEvent(wsEvent{Type: "error", Error: "server busy, try again"})
	}
	defer release()

	timeout := defaultProbeTimeout
	if req.TimeoutMs > 0 {
		timeout = time.Duration(req.TimeoutMs) * time.Millisecond
	}

	host, _, cfg, err := normalizePortScanRequest(req.Host, req.Ports, timeout, req.Concurrency)
	if err != nil {
		return writer.WriteEvent(wsEvent{Type: "error", Error: err.Error()})
	}

	rep, err := portscan.RunStreamContext(ctx, host, cfg, func(res portscan.Result) error {
		ts := time.Now()
		jr := toJSONPortScanResult(res)
		return writer.WriteEvent(wsEvent{
			Type:     "result",
			Protocol: "tcp-portscan",
			Target:   host,
			Ts:       ts.UnixMilli(),
			Result:   &jr,
		})
	})
	if err != nil {
		if errors.Is(err, context.Canceled) {
			summary := toJSONPortScanSummary(rep.Summary)
			return writer.WriteEvent(wsEvent{
				Type:     "stopped",
				Protocol: rep.Protocol,
				Target:   rep.Target,
				Addr:     rep.Addr,
				Summary:  &summary,
			})
		}
		return writer.WriteEvent(wsEvent{Type: "error", Error: err.Error()})
	}

	summary := toJSONPortScanSummary(rep.Summary)
	return writer.WriteEvent(wsEvent{
		Type:     "summary",
		Protocol: rep.Protocol,
		Target:   rep.Target,
		Addr:     rep.Addr,
		Summary:  &summary,
	})
}

func tryAcquireProbeSlot() (func(), bool) {
	select {
	case probeLimiter <- struct{}{}:
		return func() { <-probeLimiter }, true
	default:
		rejected := rejectedProbeRequests.Add(1)
		if rejected == 1 || rejected%100 == 0 {
			fmt.Fprintf(os.Stderr, "probe limiter rejected %d request(s)\n", rejected)
		}
		return nil, false
	}
}
