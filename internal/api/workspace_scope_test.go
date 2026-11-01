package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"qnetward/internal/store"
)

// A teammate who switches to a shared workspace sees the probe history and
// alert rules the owner recorded there, and loses them on switching back.
func TestTeammatesShareHistoryAndAlertRules(t *testing.T) {
	st := installTestStore(t)
	owner, ownerCookie := issueTestSession(t, st)
	member, memberCookie := issueTestSession(t, st)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	invitation, err := st.CreateWorkspaceInvitationByUserID(ctx, owner.ID, member.Email, "member", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("invite member: %v", err)
	}
	if _, err := st.AcceptWorkspaceInvitationByUserID(ctx, member.ID, invitation.ID); err != nil {
		t.Fatalf("accept invitation: %v", err)
	}
	teamID := testWorkspaceID(t, st, owner.ID)

	st.InsertAsync(store.ResultRecord{
		WorkspaceID: teamID,
		UserID:      owner.ID,
		Ts:          time.Now(),
		Protocol:    "ping",
		Target:      "shared.example",
		Seq:         1,
		RttMs:       float64Ptr(11),
	})
	waitFor(t, 10*time.Second, func() bool {
		points, err := st.QueryHistory(teamID, "ping", "shared.example", 0, 10)
		return err == nil && len(points) == 1
	})

	createReq := httptest.NewRequest(http.MethodPost, "/api/alerts/rules", strings.NewReader(`{
		"name":"Shared latency","protocol":"ping","target":"shared.example","latency_threshold_ms":150
	}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.AddCookie(ownerCookie)
	createRR := httptest.NewRecorder()
	testSrv.handleAlertRules(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("create rule: expected 201, got %d: %s", createRR.Code, createRR.Body.String())
	}

	historyCount := func() int {
		req := httptest.NewRequest(http.MethodGet, "/api/history?type=ping&host=shared.example&limit=10", nil)
		req.AddCookie(memberCookie)
		rr := httptest.NewRecorder()
		testSrv.handleHistory(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("history: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var points []store.HistoryPoint
		if err := json.Unmarshal(rr.Body.Bytes(), &points); err != nil {
			t.Fatalf("decode history: %v", err)
		}
		return len(points)
	}
	ruleCount := func() int {
		req := httptest.NewRequest(http.MethodGet, "/api/alerts/rules", nil)
		req.AddCookie(memberCookie)
		rr := httptest.NewRecorder()
		testSrv.handleAlertRules(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("rules: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var rules []store.AlertRule
		if err := json.Unmarshal(rr.Body.Bytes(), &rules); err != nil {
			t.Fatalf("decode rules: %v", err)
		}
		return len(rules)
	}
	switchTo := func(workspaceID int64) {
		if err := st.SetSessionActiveWorkspace(ctx, hashSessionToken(memberCookie.Value), member.ID, workspaceID); err != nil {
			t.Fatalf("switch workspace: %v", err)
		}
	}

	if got := historyCount(); got != 0 {
		t.Fatalf("personal workspace: expected no shared history, got %d", got)
	}
	if got := ruleCount(); got != 0 {
		t.Fatalf("personal workspace: expected no shared rules, got %d", got)
	}

	switchTo(teamID)
	if got := historyCount(); got != 1 {
		t.Fatalf("team workspace: expected the owner's sample, got %d", got)
	}
	if got := ruleCount(); got != 1 {
		t.Fatalf("team workspace: expected the owner's rule, got %d", got)
	}

	switchTo(testWorkspaceID(t, st, member.ID))
	if got := historyCount(); got != 0 {
		t.Fatalf("back in personal workspace: expected no shared history, got %d", got)
	}
}
