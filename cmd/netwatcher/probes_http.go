package main

import (
	"context"
	"net/http"
	"time"

	"netwatcher/internal/dnslookup"
	"netwatcher/internal/ping"
	"netwatcher/internal/portscan"
	"netwatcher/internal/report"
	"netwatcher/internal/store"
	"netwatcher/internal/tcpping"
	"netwatcher/pkg/telemetry"
)

type jsonResult struct {
	Seq   int      `json:"seq,omitempty"`
	Port  int      `json:"port,omitempty"`
	State string   `json:"state,omitempty"`
	RTT   string   `json:"rtt,omitempty"`
	RTTMs *float64 `json:"rtt_ms,omitempty"`
	Error string   `json:"error,omitempty"`
	Addr  string   `json:"addr,omitempty"`
}

type jsonSummary struct {
	Sent     int     `json:"sent"`
	Recv     int     `json:"recv"`
	Loss     float64 `json:"loss"`
	Min      string  `json:"min,omitempty"`
	Avg      string  `json:"avg,omitempty"`
	Max      string  `json:"max,omitempty"`
	StdDev   string  `json:"stddev,omitempty"`
	Scanned  int     `json:"scanned,omitempty"`
	Open     int     `json:"open,omitempty"`
	Closed   int     `json:"closed,omitempty"`
	Timeout  int     `json:"timeout,omitempty"`
	Duration string  `json:"duration,omitempty"`
}

type jsonReport struct {
	Protocol string       `json:"protocol"`
	Target   string       `json:"target"`
	Addr     string       `json:"addr"`
	Port     int          `json:"port,omitempty"`
	Results  []jsonResult `json:"results"`
	Summary  jsonSummary  `json:"summary"`
}

type pingRequest struct {
	Host       string `json:"host"`
	Count      int    `json:"count"`
	IntervalMs int    `json:"interval_ms"`
	TimeoutMs  int    `json:"timeout_ms"`
	Size       int    `json:"size"`
	IPv6       bool   `json:"ipv6"`
}

type tcpPingRequest struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Count      int    `json:"count"`
	IntervalMs int    `json:"interval_ms"`
	TimeoutMs  int    `json:"timeout_ms"`
}

type portScanRequest struct {
	Host        string `json:"host"`
	Ports       string `json:"ports"`
	TimeoutMs   int    `json:"timeout_ms"`
	Concurrency int    `json:"concurrency"`
}

type dnsLookupRequest struct {
	Target     string `json:"target"`
	TimeoutMs  int    `json:"timeout_ms"`
	RecordType string `json:"record_type,omitempty"`
	Resolver   string `json:"resolver,omitempty"`
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	user, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	if !isAllowedRequestOrigin(r) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
		return
	}

	var req pingRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	if normalizedHost, err := normalizeHost(req.Host); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	} else {
		req.Host = normalizedHost
	}

	count := req.Count
	interval := time.Second
	if req.IntervalMs > 0 {
		interval = time.Duration(req.IntervalMs) * time.Millisecond
	}
	timeout := 2 * time.Second
	if req.TimeoutMs > 0 {
		timeout = time.Duration(req.TimeoutMs) * time.Millisecond
	}
	size := req.Size

	cfg, err := normalizePingConfig(count, interval, timeout, size, req.IPv6)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	release, ok := tryAcquireProbeSlot()
	if !ok {
		writeJSON(w, http.StatusTooManyRequests, apiError{Error: "server busy, try again"})
		return
	}
	defer release()

	rep, err := ping.RunStreamContext(r.Context(), req.Host, cfg, func(res report.Result) error {
		storeResult(r.Context(), user.TenantID, user.ID, "ping", req.Host, res.Addr, 0, res, time.Now())
		return nil
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	enqueueAlertEvaluation(user, "ping", req.Host, 0, rep.Summary, buildDashboardURL(r, "alerts"))

	writeJSON(w, http.StatusOK, toJSONReport(rep))
}

func handleTCPPing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	user, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	if !isAllowedRequestOrigin(r) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
		return
	}

	var req tcpPingRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

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

	count := req.Count
	interval := time.Second
	if req.IntervalMs > 0 {
		interval = time.Duration(req.IntervalMs) * time.Millisecond
	}
	timeout := 2 * time.Second
	if req.TimeoutMs > 0 {
		timeout = time.Duration(req.TimeoutMs) * time.Millisecond
	}

	host, port, cfg, err := normalizeTCPPingRequest(host, port, count, interval, timeout)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	release, ok := tryAcquireProbeSlot()
	if !ok {
		writeJSON(w, http.StatusTooManyRequests, apiError{Error: "server busy, try again"})
		return
	}
	defer release()

	rep, err := tcpping.RunStreamContext(r.Context(), host, port, cfg, func(res report.Result) error {
		storeResult(r.Context(), user.TenantID, user.ID, "tcpping", host, res.Addr, port, res, time.Now())
		return nil
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	enqueueAlertEvaluation(user, "tcpping", host, port, rep.Summary, buildDashboardURL(r, "alerts"))

	writeJSON(w, http.StatusOK, toJSONReport(rep))
}

func handlePortScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	if _, ok := requireAuthenticatedUser(w, r); !ok {
		return
	}
	if !isAllowedRequestOrigin(r) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
		return
	}

	var req portScanRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	timeout := defaultProbeTimeout
	if req.TimeoutMs > 0 {
		timeout = time.Duration(req.TimeoutMs) * time.Millisecond
	}

	host, _, cfg, err := normalizePortScanRequest(req.Host, req.Ports, timeout, req.Concurrency)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	release, ok := tryAcquireProbeSlot()
	if !ok {
		writeJSON(w, http.StatusTooManyRequests, apiError{Error: "server busy, try again"})
		return
	}
	defer release()

	rep, err := portscan.RunContext(r.Context(), host, cfg)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, toJSONPortScanReport(rep))
}

func handleDNSLookup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return
	}
	if _, ok := requireAuthenticatedUser(w, r); !ok {
		return
	}
	if !isAllowedRequestOrigin(r) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
		return
	}

	var req dnsLookupRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	target, err := normalizeHost(req.Target)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	timeout := dnslookup.DefaultTimeout
	if req.TimeoutMs > 0 {
		timeout = time.Duration(req.TimeoutMs) * time.Millisecond
	}
	timeout, err = normalizeDNSLookupTimeout(timeout)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}

	release, ok := tryAcquireProbeSlot()
	if !ok {
		writeJSON(w, http.StatusTooManyRequests, apiError{Error: "server busy, try again"})
		return
	}
	defer release()

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	rep, err := dnslookup.RunContext(ctx, target, dnslookup.Config{
		Timeout:    timeout,
		RecordType: req.RecordType,
		Resolver:   req.Resolver,
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

func toJSONReport(rep report.Report) jsonReport {
	jr := jsonReport{
		Protocol: rep.Protocol,
		Target:   rep.Target,
		Addr:     rep.Addr,
		Port:     rep.Port,
		Summary:  toJSONSummary(rep.Summary),
	}

	jr.Results = make([]jsonResult, 0, len(rep.Results))
	for _, r := range rep.Results {
		jr.Results = append(jr.Results, toJSONResult(r))
	}

	return jr
}

func toJSONPortScanReport(rep portscan.Report) jsonReport {
	jr := jsonReport{
		Protocol: rep.Protocol,
		Target:   rep.Target,
		Addr:     rep.Addr,
		Summary:  toJSONPortScanSummary(rep.Summary),
	}

	jr.Results = make([]jsonResult, 0, len(rep.Results))
	for _, r := range rep.Results {
		jr.Results = append(jr.Results, toJSONPortScanResult(r))
	}

	return jr
}

func toJSONSummary(s report.Summary) jsonSummary {
	js := jsonSummary{
		Sent: s.Sent,
		Recv: s.Recv,
		Loss: s.Loss,
	}
	if s.Recv > 0 {
		js.Min = formatDuration(s.Min)
		js.Avg = formatDuration(s.Avg)
		js.Max = formatDuration(s.Max)
		js.StdDev = formatDuration(s.StdDev)
	}
	return js
}

func toJSONPortScanSummary(s portscan.Summary) jsonSummary {
	return jsonSummary{
		Scanned:  s.Scanned,
		Open:     s.Open,
		Closed:   s.Closed,
		Timeout:  s.Timeout,
		Duration: s.Duration.Round(time.Millisecond).String(),
	}
}

func toJSONResult(r report.Result) jsonResult {
	jr := jsonResult{Seq: r.Seq}
	if r.Error != "" {
		jr.Error = r.Error
	} else {
		jr.RTT = formatDuration(r.RTT)
		ms := float64(r.RTT) / float64(time.Millisecond)
		jr.RTTMs = &ms
	}
	if r.Addr != "" {
		jr.Addr = r.Addr
	}
	return jr
}

func toJSONPortScanResult(r portscan.Result) jsonResult {
	jr := jsonResult{
		Port:  r.Port,
		State: r.State,
		Error: r.Error,
		Addr:  r.Addr,
	}
	if r.State == "open" {
		jr.RTT = formatDuration(r.RTT)
		ms := float64(r.RTT) / float64(time.Millisecond)
		jr.RTTMs = &ms
	}
	return jr
}

func storeResult(ctx context.Context, tenantID, userID int64, protocol, target, addr string, port int, res report.Result, ts time.Time) {
	telemetry.RecordProbe(ctx, protocol, res.Error == "", res.RTT)

	if appStore == nil {
		return
	}
	rec := store.ResultRecord{
		TenantID: tenantID,
		UserID:   userID,
		Ts:       ts,
		Protocol: protocol,
		Target:   target,
		Addr:     addr,
		Port:     port,
		Seq:      res.Seq,
		Error:    res.Error,
	}
	if res.Error == "" {
		ms := float64(res.RTT) / float64(time.Millisecond)
		rec.RttMs = &ms
	}
	appStore.InsertAsync(rec)
}
