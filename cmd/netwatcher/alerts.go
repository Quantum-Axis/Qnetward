package main

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

	"netwatcher/internal/mailer"
	"netwatcher/internal/report"
	"netwatcher/internal/store"
	"netwatcher/pkg/telemetry"
)

// alertEvalWG tracks in-flight background alert evaluations spawned by
// enqueueAlertEvaluation. Tests that swap out the global appStore/appMailer
// between runs (see installTestStore in main_test.go) wait on this before
// reassigning those globals or closing the store, so a goroutine left over
// from a previous test never touches a store that has since been closed or
// replaced.
var alertEvalWG sync.WaitGroup

type alertRuleRequest struct {
	ID                   int64   `json:"id,omitempty"`
	Name                 string  `json:"name,omitempty"`
	Protocol             string  `json:"protocol"`
	Target               string  `json:"target"`
	Port                 int     `json:"port,omitempty"`
	RecipientEmail       string  `json:"recipient_email,omitempty"`
	LatencyThresholdMs   float64 `json:"latency_threshold_ms,omitempty"`
	LossThresholdPercent float64 `json:"loss_threshold_percent,omitempty"`
	ConsecutiveBreaches  int     `json:"consecutive_breaches,omitempty"`
	CooldownMinutes      int     `json:"cooldown_minutes,omitempty"`
	NotifyOnRecovery     *bool   `json:"notify_on_recovery,omitempty"`
	Enabled              *bool   `json:"enabled,omitempty"`
}

func handleAlertRules(w http.ResponseWriter, r *http.Request) {
	user, ok := requireAuthenticatedUser(w, r)
	if !ok {
		return
	}
	if appStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		rules, err := appStore.ListAlertRules(ctx, user.TenantID, user.ID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, rules)
	case http.MethodPost:
		if !isAllowedRequestOrigin(r) {
			writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
			return
		}
		var req alertRuleRequest
		if err := decodeJSON(w, r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
			return
		}
		rule, err := normalizeAlertRuleRequest(user, req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		saved, err := appStore.SaveAlertRule(ctx, user.TenantID, rule)
		if err != nil {
			switch {
			case errors.Is(err, store.ErrAlertRuleNotFound):
				writeJSON(w, http.StatusNotFound, apiError{Error: "alert rule not found"})
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
		if !isAllowedRequestOrigin(r) {
			writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
			return
		}
		ruleID, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("id")), 10, 64)
		if err != nil || ruleID <= 0 {
			writeJSON(w, http.StatusBadRequest, apiError{Error: "valid rule id is required"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		if err := appStore.DeleteAlertRule(ctx, user.TenantID, user.ID, ruleID); err != nil {
			switch {
			case errors.Is(err, store.ErrAlertRuleNotFound):
				writeJSON(w, http.StatusNotFound, apiError{Error: "alert rule not found"})
			default:
				writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
			}
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "id": ruleID})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
	}
}

func normalizeAlertRuleRequest(user store.User, req alertRuleRequest) (store.AlertRule, error) {
	protocol := strings.ToLower(strings.TrimSpace(req.Protocol))
	target := strings.TrimSpace(req.Target)
	port := req.Port

	switch protocol {
	case "ping":
		normalizedTarget, err := normalizeHost(target)
		if err != nil {
			return store.AlertRule{}, err
		}
		target = normalizedTarget
		port = 0
	case "tcpping":
		if parsedHost, parsedPort, ok := splitHostPort(target); ok {
			target = parsedHost
			if port == 0 {
				port = parsedPort
			}
		}
		normalizedTarget, normalizedPort, _, err := normalizeTCPPingRequest(target, port, 1, defaultProbeInterval, defaultProbeTimeout)
		if err != nil {
			return store.AlertRule{}, err
		}
		target = normalizedTarget
		port = normalizedPort
	default:
		return store.AlertRule{}, fmt.Errorf("protocol must be ping or tcpping")
	}

	var latencyThreshold *float64
	if req.LatencyThresholdMs > 0 {
		v := req.LatencyThresholdMs
		latencyThreshold = &v
	}
	var lossThreshold *float64
	if req.LossThresholdPercent > 0 {
		v := req.LossThresholdPercent
		lossThreshold = &v
	}

	recipientEmail := strings.TrimSpace(req.RecipientEmail)
	if recipientEmail == "" {
		recipientEmail = user.Email
	}

	notifyOnRecovery := true
	if req.NotifyOnRecovery != nil {
		notifyOnRecovery = *req.NotifyOnRecovery
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	rule := store.AlertRule{
		ID:                   req.ID,
		UserID:               user.ID,
		Name:                 strings.TrimSpace(req.Name),
		Protocol:             protocol,
		Target:               target,
		RecipientEmail:       recipientEmail,
		LatencyThresholdMs:   latencyThreshold,
		LossThresholdPercent: lossThreshold,
		ConsecutiveBreaches:  req.ConsecutiveBreaches,
		CooldownMinutes:      req.CooldownMinutes,
		NotifyOnRecovery:     notifyOnRecovery,
		Enabled:              enabled,
	}
	if protocol == "tcpping" {
		rule.Port = &port
	}
	return rule, nil
}

func enqueueAlertEvaluation(user store.User, protocol, target string, port int, summary report.Summary, dashboardURL string) {
	// Snapshot the globals now, synchronously, rather than reading them
	// again inside the goroutine below: appStore/appMailer can be
	// reassigned concurrently (only observed from tests, which swap them
	// per test run), and alertEvalWG only guarantees this goroutine finishes
	// before the *next* reassignment, not that appStore stays unchanged
	// for its entire duration.
	st := appStore
	if st == nil {
		return
	}
	mailerClient := appMailer

	sample := store.AlertSample{
		LossPercent: summary.Loss,
		Sent:        summary.Sent,
		Recv:        summary.Recv,
	}
	if summary.Recv > 0 {
		avgMs := float64(summary.Avg) / float64(time.Millisecond)
		sample.LatencyAvgMs = &avgMs
	}

	alertEvalWG.Add(1)
	go func() {
		defer alertEvalWG.Done()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		decisions, err := st.EvaluateAlertRules(ctx, user.TenantID, user.ID, protocol, target, port, sample)
		if err != nil {
			log.Printf("alert evaluation failed for %s/%s: %v", protocol, target, err)
			return
		}
		telemetry.RecordAlertEvaluation(ctx, protocol, len(decisions))
		if mailerClient == nil {
			return
		}
		for _, decision := range decisions {
			alertTitle := decision.Rule.Name
			if strings.TrimSpace(alertTitle) == "" {
				alertTitle = "Monitoring Alert"
			}
			if err := mailerClient.SendAlert(ctx, mailer.AlertEmailInput{
				To:           decision.Rule.RecipientEmail,
				Name:         user.Name,
				Severity:     decision.Severity,
				Title:        alertTitle,
				Summary:      decision.Summary,
				Target:       alertTargetLabel(decision.Rule),
				TriggeredAt:  time.Now().UTC(),
				DashboardURL: dashboardURL,
			}); err != nil {
				log.Printf("alert email failed for rule %d: %v", decision.Rule.ID, err)
			}
		}
	}()
}

func alertTargetLabel(rule store.AlertRule) string {
	if rule.Protocol == "tcpping" && rule.Port != nil {
		return fmt.Sprintf("%s:%d", rule.Target, *rule.Port)
	}
	return rule.Target
}
