package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"qnetward/internal/engine/ping"
	"qnetward/internal/engine/report"
	"qnetward/internal/engine/tcpping"
	"qnetward/internal/platform/mailer"
	"qnetward/internal/probe"
	"qnetward/internal/store"
)

// Scheduling bounds for uptime monitors.
const (
	monitorPollEvery      = 5 * time.Second
	monitorClaimBatch     = 20
	monitorConcurrency    = 8
	monitorProbeCount     = 3
	monitorProbeInterval  = 500 * time.Millisecond
	monitorProbeTimeout   = 2 * time.Second
	monitorCheckRetention = 30 * 24 * time.Hour
	monitorPruneEvery     = time.Hour
)

type monitorRequest struct {
	ID              int64  `json:"id,omitempty"`
	Name            string `json:"name,omitempty"`
	Protocol        string `json:"protocol"`
	Target          string `json:"target"`
	Port            int    `json:"port,omitempty"`
	IntervalSeconds int    `json:"interval_seconds"`
	NotifyEmail     string `json:"notify_email"`
	Enabled         *bool  `json:"enabled,omitempty"`
}

func (srv *Server) handleMonitors(w http.ResponseWriter, r *http.Request) {
	user, ok := srv.requireActor(w, r)
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		monitors, err := srv.store.ListMonitors(ctx, user.WorkspaceID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, monitors)
	case http.MethodPost:
		if !srv.isAllowedRequestOrigin(r) {
			writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
			return
		}
		var req monitorRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
			return
		}
		monitor, err := normalizeMonitorRequest(user, req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		saved, err := srv.store.SaveMonitor(ctx, monitor)
		if err != nil {
			switch {
			case errors.Is(err, store.ErrMonitorNotFound):
				writeJSON(w, http.StatusNotFound, apiError{Error: err.Error()})
			case errors.Is(err, store.ErrMonitorDuplicate), errors.Is(err, store.ErrMonitorLimit):
				writeJSON(w, http.StatusConflict, apiError{Error: err.Error()})
			default:
				writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
			}
			return
		}
		status := http.StatusOK
		if req.ID == 0 {
			status = http.StatusCreated
		}
		writeJSON(w, status, saved)
	case http.MethodDelete:
		if !srv.isAllowedRequestOrigin(r) {
			writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
			return
		}
		id, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("id")), 10, 64)
		if err != nil || id <= 0 {
			writeJSON(w, http.StatusBadRequest, apiError{Error: "valid monitor id is required"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		if err := srv.store.DeleteMonitor(ctx, user.WorkspaceID, id); err != nil {
			if errors.Is(err, store.ErrMonitorNotFound) {
				writeJSON(w, http.StatusNotFound, apiError{Error: err.Error()})
				return
			}
			writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": id})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
	}
}

func normalizeMonitorRequest(user actor, req monitorRequest) (store.Monitor, error) {
	target := strings.TrimSpace(req.Target)
	port := req.Port
	if host, parsedPort, ok := probe.SplitHostPort(target); ok {
		target = host
		if port == 0 {
			port = parsedPort
		}
	}
	host, err := probe.NormalizeHost(target)
	if err != nil {
		return store.Monitor{}, err
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	m := store.Monitor{
		ID:              req.ID,
		WorkspaceID:     user.WorkspaceID,
		UserID:          user.ID,
		Name:            req.Name,
		Protocol:        req.Protocol,
		Target:          host,
		IntervalSeconds: req.IntervalSeconds,
		NotifyEmail:     req.NotifyEmail,
		Enabled:         enabled,
	}
	if strings.EqualFold(strings.TrimSpace(req.Protocol), "tcpping") {
		m.Port = &port
	}
	return m, nil
}

// RunMonitors checks due monitors until ctx is cancelled. Every instance can
// run it: claiming a monitor moves its next run forward, so no two instances
// check the same monitor at once.
func (srv *Server) RunMonitors(ctx context.Context) {
	if srv.store == nil {
		return
	}
	poll := time.NewTicker(monitorPollEvery)
	defer poll.Stop()
	prune := time.NewTicker(monitorPruneEvery)
	defer prune.Stop()

	slots := make(chan struct{}, monitorConcurrency)
	var wg sync.WaitGroup
	defer wg.Wait()

	for {
		select {
		case <-ctx.Done():
			return
		case <-prune.C:
			pruneCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			if err := srv.store.PruneMonitorChecks(pruneCtx, time.Now().Add(-monitorCheckRetention)); err != nil {
				log.Printf("monitor check pruning failed: %v", err)
			}
			cancel()
		case <-poll.C:
			free := monitorConcurrency - len(slots)
			if free <= 0 {
				continue
			}
			claimCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			due, err := srv.store.ClaimDueMonitors(claimCtx, min(free, monitorClaimBatch))
			cancel()
			if err != nil {
				log.Printf("claiming due monitors failed: %v", err)
				continue
			}
			for _, m := range due {
				slots <- struct{}{}
				wg.Add(1)
				go func(m store.Monitor) {
					defer wg.Done()
					defer func() { <-slots }()
					srv.checkMonitor(ctx, m)
				}(m)
			}
		}
	}
}

// checkMonitor probes one monitor, records the outcome, feeds the samples to
// history and alert rules, and emails on a change between up and down.
func (srv *Server) checkMonitor(ctx context.Context, m store.Monitor) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	port := 0
	if m.Port != nil {
		port = *m.Port
	}
	record := func(res report.Result) error {
		srv.storeResult(m.WorkspaceID, m.UserID, m.Protocol, m.Target, res.Addr, port, res, time.Now())
		return nil
	}

	var rep report.Report
	var err error
	switch m.Protocol {
	case "ping":
		var cfg ping.Config
		cfg, err = probe.NormalizePingConfig(monitorProbeCount, monitorProbeInterval, monitorProbeTimeout, 0, false, false)
		if err == nil {
			cfg.AddrCheck = srv.targetPolicy.CheckIP
			rep, err = ping.RunStreamContext(ctx, m.Target, cfg, record)
		}
	case "tcpping":
		var cfg tcpping.Config
		_, _, cfg, err = probe.NormalizeTCPPingRequest(m.Target, port, monitorProbeCount, monitorProbeInterval, monitorProbeTimeout, false)
		if err == nil {
			cfg.AddrCheck = srv.targetPolicy.CheckIP
			rep, err = tcpping.RunStreamContext(ctx, m.Target, port, cfg, record)
		}
	default:
		err = fmt.Errorf("unsupported protocol %q", m.Protocol)
	}
	if ctx.Err() != nil && errors.Is(err, context.Canceled) {
		return // shutting down
	}

	check := store.MonitorCheck{CheckedAt: time.Now(), LossPercent: 100}
	switch {
	case err != nil:
		check.Error = err.Error()
	case rep.Summary.Recv == 0:
		check.Error = firstProbeError(rep)
	default:
		check.OK = true
		check.LossPercent = rep.Summary.Loss
		avg := float64(rep.Summary.Avg) / float64(time.Millisecond)
		check.RttMs = &avg
	}

	saveCtx, saveCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer saveCancel()
	previous, err := srv.store.RecordMonitorCheck(saveCtx, m, check)
	if err != nil {
		if !errors.Is(err, store.ErrMonitorNotFound) {
			log.Printf("recording check for monitor %d failed: %v", m.ID, err)
		}
		return
	}

	if rep.Summary.Sent > 0 {
		srv.enqueueAlertEvaluation(actor{User: store.User{ID: m.UserID}, WorkspaceID: m.WorkspaceID},
			m.Protocol, m.Target, port, rep.Summary, srv.buildConfiguredDashboardURL("alerts"))
	}

	wentDown := !check.OK && previous != "down"
	recovered := check.OK && previous == "down"
	if (wentDown || recovered) && m.NotifyEmail != "" && srv.mailer != nil {
		srv.sendMonitorEmail(saveCtx, m, check, recovered)
	}
}

func (srv *Server) sendMonitorEmail(ctx context.Context, m store.Monitor, check store.MonitorCheck, recovered bool) {
	target := m.Target
	if m.Port != nil {
		target = fmt.Sprintf("%s:%d", m.Target, *m.Port)
	}
	in := mailer.AlertEmailInput{
		To:           m.NotifyEmail,
		Severity:     "critical",
		Title:        m.Name + " is down",
		Summary:      fmt.Sprintf("No replies from %s in the last check. %s", target, check.Error),
		Target:       target,
		TriggeredAt:  check.CheckedAt.UTC(),
		DashboardURL: srv.buildConfiguredDashboardURL("monitors"),
	}
	if recovered {
		in.Severity = "success"
		in.Title = m.Name + " is back up"
		in.Summary = fmt.Sprintf("%s is answering again.", target)
		if check.RttMs != nil {
			in.Summary = fmt.Sprintf("%s is answering again, average %.1f ms.", target, *check.RttMs)
		}
	}
	in.Summary = strings.TrimSpace(in.Summary)
	if err := srv.mailer.SendAlert(ctx, in); err != nil {
		log.Printf("monitor email failed for monitor %d: %v", m.ID, err)
	}
}

func firstProbeError(rep report.Report) string {
	for _, res := range rep.Results {
		if res.Error != "" {
			return res.Error
		}
	}
	return "no replies"
}
