package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"qnetward/internal/store"
)

func TestHandleAdminOverviewRequiresAdmin(t *testing.T) {
	st := installTestStore(t)
	_, cookie := issueTestSession(t, st)

	previous := testSrv.adminEmails
	testSrv.adminEmails = map[string]struct{}{}
	t.Cleanup(func() {
		testSrv.adminEmails = previous
	})

	req := httptest.NewRequest(http.MethodGet, "/api/admin/overview", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()

	testSrv.handleAdminOverview(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHandleAdminOverviewReturnsCountsForAdmin(t *testing.T) {
	st := installTestStore(t)
	user, cookie := issueTestSession(t, st)

	previous := testSrv.adminEmails
	testSrv.adminEmails = map[string]struct{}{strings.ToLower(user.Email): {}}
	t.Cleanup(func() {
		testSrv.adminEmails = previous
	})

	req := httptest.NewRequest(http.MethodGet, "/api/admin/overview", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()

	testSrv.handleAdminOverview(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var overview store.AdminOverview
	if err := json.Unmarshal(rr.Body.Bytes(), &overview); err != nil {
		t.Fatalf("decode admin overview: %v", err)
	}
	if overview.UserCount < 1 || overview.WorkspaceCount < 1 {
		t.Fatalf("unexpected admin overview payload: %#v", overview)
	}
}

func TestHandleAdminUsersWorkspacesAndSystemReturnPayloads(t *testing.T) {
	st := installTestStore(t)
	user, cookie := issueTestSession(t, st)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := st.CreateSession(ctx, user.ID, fmt.Sprintf("extra-session-%d", time.Now().UnixNano()), time.Now().Add(24*time.Hour)); err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}

	previousAdmins := testSrv.adminEmails
	previousDemoEnabled := testSrv.demoEnabled
	previousPublicBaseURL := testSrv.publicBaseURL
	testSrv.adminEmails = map[string]struct{}{strings.ToLower(user.Email): {}}
	testSrv.demoEnabled = true
	testSrv.publicBaseURL = "https://qnetward.gitzen.cloud"
	t.Cleanup(func() {
		testSrv.adminEmails = previousAdmins
		testSrv.demoEnabled = previousDemoEnabled
		testSrv.publicBaseURL = previousPublicBaseURL
	})

	usersReq := httptest.NewRequest(http.MethodGet, "/api/admin/users?limit=10", nil)
	usersReq.AddCookie(cookie)
	usersRR := httptest.NewRecorder()
	testSrv.handleAdminUsers(usersRR, usersReq)

	if usersRR.Code != http.StatusOK {
		t.Fatalf("expected 200 from admin users, got %d: %s", usersRR.Code, usersRR.Body.String())
	}

	var usersResp adminUsersResponse
	if err := json.Unmarshal(usersRR.Body.Bytes(), &usersResp); err != nil {
		t.Fatalf("decode admin users response: %v", err)
	}
	if len(usersResp.Users) == 0 {
		t.Fatal("expected admin users payload")
	}
	if usersResp.Users[0].PlatformRole == "" {
		t.Fatal("expected platform role in admin users payload")
	}

	workspacesReq := httptest.NewRequest(http.MethodGet, "/api/admin/workspaces?limit=10", nil)
	workspacesReq.AddCookie(cookie)
	workspacesRR := httptest.NewRecorder()
	testSrv.handleAdminWorkspaces(workspacesRR, workspacesReq)

	if workspacesRR.Code != http.StatusOK {
		t.Fatalf("expected 200 from admin workspaces, got %d: %s", workspacesRR.Code, workspacesRR.Body.String())
	}

	var workspacesResp adminWorkspacesResponse
	if err := json.Unmarshal(workspacesRR.Body.Bytes(), &workspacesResp); err != nil {
		t.Fatalf("decode admin workspaces response: %v", err)
	}
	if len(workspacesResp.Workspaces) == 0 {
		t.Fatal("expected admin workspaces payload")
	}
	if workspacesResp.Workspaces[0].OwnerEmail == "" {
		t.Fatal("expected workspace owner email in admin workspaces payload")
	}

	systemReq := httptest.NewRequest(http.MethodGet, "/api/admin/system", nil)
	systemReq.AddCookie(cookie)
	systemRR := httptest.NewRecorder()
	testSrv.handleAdminSystem(systemRR, systemReq)

	if systemRR.Code != http.StatusOK {
		t.Fatalf("expected 200 from admin system, got %d: %s", systemRR.Code, systemRR.Body.String())
	}

	var systemResp authAdminSystemResponse
	if err := json.Unmarshal(systemRR.Body.Bytes(), &systemResp); err != nil {
		t.Fatalf("decode admin system response: %v", err)
	}
	if systemResp.Features.PublicBaseURL != "https://qnetward.gitzen.cloud" {
		t.Fatalf("unexpected public base url: %#v", systemResp.Features)
	}
	if systemResp.Features.ConfiguredAdmins != 1 {
		t.Fatalf("expected 1 configured admin, got %d", systemResp.Features.ConfiguredAdmins)
	}
	if systemResp.Overview.UserCount < 1 {
		t.Fatalf("unexpected system overview payload: %#v", systemResp.Overview)
	}
}
