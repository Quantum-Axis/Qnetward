package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"qnetward/internal/store"
)

func TestAlertRulesCRUDFlow(t *testing.T) {
	st := installTestStore(t)
	cookie := issueTestSessionCookie(t, st)

	createReq := httptest.NewRequest(http.MethodPost, "/api/alerts/rules", strings.NewReader(`{
		"name":"Latency guard",
		"protocol":"ping",
		"target":"example.com",
		"latency_threshold_ms":200,
		"consecutive_breaches":2,
		"cooldown_minutes":15
	}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.AddCookie(cookie)
	createRR := httptest.NewRecorder()
	testSrv.handleAlertRules(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", createRR.Code, createRR.Body.String())
	}

	var created store.AlertRule
	if err := json.Unmarshal(createRR.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created rule: %v", err)
	}
	if created.ID == 0 || created.Protocol != "ping" || created.Target != "example.com" {
		t.Fatalf("unexpected created rule: %#v", created)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/alerts/rules", nil)
	listReq.AddCookie(cookie)
	listRR := httptest.NewRecorder()
	testSrv.handleAlertRules(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", listRR.Code, listRR.Body.String())
	}

	var rules []store.AlertRule
	if err := json.Unmarshal(listRR.Body.Bytes(), &rules); err != nil {
		t.Fatalf("decode rules: %v", err)
	}
	if len(rules) != 1 || rules[0].ID != created.ID {
		t.Fatalf("unexpected rules: %#v", rules)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/alerts/rules?id=%d", created.ID), nil)
	deleteReq.AddCookie(cookie)
	deleteRR := httptest.NewRecorder()
	testSrv.handleAlertRules(deleteRR, deleteReq)
	if deleteRR.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", deleteRR.Code, deleteRR.Body.String())
	}
}

func TestBuildDashboardURLUsesPanel(t *testing.T) {
	oldBase := testSrv.publicBaseURL
	testSrv.publicBaseURL = "https://qnetward.example.com"
	t.Cleanup(func() {
		testSrv.publicBaseURL = oldBase
	})

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8080/", nil)
	got := testSrv.buildDashboardURL(req, "alerts")
	want := "https://qnetward.example.com/alerts"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
