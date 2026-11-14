package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"qnetward/internal/store"
)

// Viewers see a workspace but can't change it: reads (including reads sent as
// POST, like an export) work; creating, running, and probing are refused.
func TestViewersCanReadButNotChange(t *testing.T) {
	st := installTestStore(t)
	owner, _ := issueTestSession(t, st)
	viewer, viewerCookie := issueTestSession(t, st)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	invitation, err := st.CreateWorkspaceInvitationByUserID(ctx, owner.ID, viewer.Email, "viewer", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.AcceptWorkspaceInvitationByUserID(ctx, viewer.ID, invitation.ID); err != nil {
		t.Fatal(err)
	}
	teamID := testWorkspaceID(t, st, owner.ID)
	if err := st.SetSessionActiveWorkspace(ctx, hashSessionToken(viewerCookie.Value), viewer.ID, teamID); err != nil {
		t.Fatal(err)
	}

	call := func(method, path, body string) int {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(viewerCookie)
		rr := httptest.NewRecorder()
		testSrv.Handler().ServeHTTP(rr, req)
		return rr.Code
	}

	for _, read := range []struct{ method, path, body string }{
		{http.MethodGet, "/api/monitors", ""},
		{http.MethodGet, "/api/history/recent-targets?limit=5", ""},
		{http.MethodPost, "/api/export/history", `{"format":"json"}`},
	} {
		if code := call(read.method, read.path, read.body); code == http.StatusForbidden || code == http.StatusUnauthorized {
			t.Errorf("%s %s: a viewer should be able to read, got %d", read.method, read.path, code)
		}
	}
	for _, write := range []struct{ method, path, body string }{
		{http.MethodPost, "/api/monitors", `{"name":"x","protocol":"tcpping","target":"example.com","port":443,"interval_seconds":300,"enabled":true}`},
		{http.MethodPost, "/api/tcpping", `{"host":"example.com","port":443,"count":1}`},
		{http.MethodPost, "/api/status-pages", `{"slug":"viewer-test","title":"x","published":false,"monitors":[]}`},
	} {
		if code := call(write.method, write.path, write.body); code != http.StatusForbidden {
			t.Errorf("%s %s: a viewer must be refused, got %d", write.method, write.path, code)
		}
	}
}

func TestRoleMayAct(t *testing.T) {
	for _, c := range []struct {
		role, scope, method string
		want                bool
	}{
		{"viewer", store.ScopeRead, http.MethodPost, true}, // a read sent as POST
		{"viewer", store.ScopeWrite, http.MethodPost, false},
		{"viewer", store.ScopeProbe, http.MethodPost, false},
		{"viewer", "", http.MethodGet, true},
		{"viewer", "", http.MethodDelete, false},
		{"member", store.ScopeWrite, http.MethodPost, true},
		{"admin", store.ScopeProbe, http.MethodPost, true},
	} {
		if got := roleMayAct(c.role, c.scope, c.method); got != c.want {
			t.Errorf("%s %s %s: got %v, want %v", c.role, c.scope, c.method, got, c.want)
		}
	}
}

// An invitation's link signs existing users in and signs new ones up with
// their address filled in; either way it ends on their invitations.
func TestInvitationAcceptPath(t *testing.T) {
	if got := invitationAcceptPath("a@example.com", true); got != "/login?next=%2Fsettings%2Fworkspaces" {
		t.Fatalf("existing account: %q", got)
	}
	if got := invitationAcceptPath("new+x@example.com", false); got != "/signup?email=new%2Bx%40example.com&next=%2Fsettings%2Fworkspaces" {
		t.Fatalf("new account: %q", got)
	}
}

// Ownership moves in one step and a workspace never ends up without an owner.
func TestTransferOwnershipAndLeave(t *testing.T) {
	st := installTestStore(t)
	owner, ownerCookie := issueTestSession(t, st)
	member, memberCookie := issueTestSession(t, st)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	invitation, err := st.CreateWorkspaceInvitationByUserID(ctx, owner.ID, member.Email, "member", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.AcceptWorkspaceInvitationByUserID(ctx, member.ID, invitation.ID); err != nil {
		t.Fatal(err)
	}
	teamID := testWorkspaceID(t, st, owner.ID)
	if err := st.SetSessionActiveWorkspace(ctx, hashSessionToken(memberCookie.Value), member.ID, teamID); err != nil {
		t.Fatal(err)
	}
	ws, err := st.GetWorkspaceContextByWorkspaceIDAndUserID(ctx, teamID, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	post := func(cookie *http.Cookie, path, body string) int {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		testSrv.Handler().ServeHTTP(rr, req)
		return rr.Code
	}
	role := func(userID int64) string {
		w, err := st.GetWorkspaceContextByWorkspaceIDAndUserID(ctx, teamID, userID)
		if err != nil {
			return "none"
		}
		return w.Role
	}

	if code := post(ownerCookie, "/api/workspace/leave", ""); code != http.StatusConflict {
		t.Fatalf("the only owner must not leave, got %d", code)
	}
	if code := post(ownerCookie, "/api/workspace/team/transfer", fmt.Sprintf(`{"user_id":%d,"confirm":"not the name"}`, member.ID)); code != http.StatusBadRequest {
		t.Fatalf("a wrong confirmation must be refused, got %d", code)
	}
	if code := post(memberCookie, "/api/workspace/team/transfer", fmt.Sprintf(`{"user_id":%d,"confirm":%q}`, owner.ID, ws.Name)); code != http.StatusForbidden {
		t.Fatalf("a member can't take ownership, got %d", code)
	}
	if code := post(ownerCookie, "/api/workspace/team/transfer", fmt.Sprintf(`{"user_id":%d,"confirm":%q}`, member.ID, ws.Name)); code != http.StatusOK {
		t.Fatalf("transfer: got %d", code)
	}
	if role(member.ID) != "owner" || role(owner.ID) != "admin" {
		t.Fatalf("after the transfer: new owner %q, old owner %q", role(member.ID), role(owner.ID))
	}
	if code := post(ownerCookie, "/api/workspace/leave", ""); code != http.StatusOK {
		t.Fatalf("an admin may leave, got %d", code)
	}
	if role(owner.ID) != "none" {
		t.Fatal("the old owner should be gone from the workspace")
	}
	if code := post(memberCookie, "/api/workspace/leave", ""); code != http.StatusConflict {
		t.Fatalf("the new, only owner must not leave, got %d", code)
	}
}

// A workspace that requires two-factor sign-in keeps its data from members
// without it, and an owner can only turn it on with two-factor themselves.
func TestRequireTwoFactor(t *testing.T) {
	st := installTestStore(t)
	owner, ownerCookie := issueTestSession(t, st)
	member, memberCookie := issueTestSession(t, st)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	invitation, err := st.CreateWorkspaceInvitationByUserID(ctx, owner.ID, member.Email, "member", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.AcceptWorkspaceInvitationByUserID(ctx, member.ID, invitation.ID); err != nil {
		t.Fatal(err)
	}
	teamID := testWorkspaceID(t, st, owner.ID)
	if err := st.SetSessionActiveWorkspace(ctx, hashSessionToken(memberCookie.Value), member.ID, teamID); err != nil {
		t.Fatal(err)
	}
	call := func(cookie *http.Cookie, method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(cookie)
		rr := httptest.NewRecorder()
		testSrv.Handler().ServeHTTP(rr, req)
		return rr
	}

	if rr := call(ownerCookie, http.MethodPatch, "/api/workspace", `{"require_mfa":true}`); rr.Code != http.StatusConflict {
		t.Fatalf("an owner without two-factor can't require it, got %d", rr.Code)
	}
	if err := st.StartMFASetup(ctx, owner.ID, "sealed-secret"); err != nil {
		t.Fatal(err)
	}
	if err := st.EnableMFA(ctx, owner.ID, 1, []string{}); err != nil {
		t.Fatal(err)
	}
	if rr := call(ownerCookie, http.MethodPatch, "/api/workspace", `{"require_mfa":true}`); rr.Code != http.StatusOK {
		t.Fatalf("require two-factor: %d %s", rr.Code, rr.Body.String())
	}
	if rr := call(memberCookie, http.MethodGet, "/api/monitors", ""); rr.Code != http.StatusForbidden || !strings.Contains(rr.Body.String(), "mfa_required") {
		t.Fatalf("a member without two-factor must be kept out, got %d %s", rr.Code, rr.Body.String())
	}
	if rr := call(memberCookie, http.MethodGet, "/api/account/mfa", ""); rr.Code != http.StatusOK {
		t.Fatalf("they can still reach their own security settings, got %d", rr.Code)
	}
	if rr := call(ownerCookie, http.MethodGet, "/api/monitors", ""); rr.Code != http.StatusOK {
		t.Fatalf("the owner, with two-factor, gets in, got %d", rr.Code)
	}
	// Owners see who has two-factor; members don't.
	if rr := call(ownerCookie, http.MethodGet, "/api/workspace/team", ""); !strings.Contains(rr.Body.String(), `"mfa_enabled"`) {
		t.Fatalf("owners should see two-factor status: %s", rr.Body.String())
	}
	if rr := call(ownerCookie, http.MethodPatch, "/api/workspace", `{"require_mfa":false}`); rr.Code != http.StatusOK {
		t.Fatalf("turn the requirement off: %d", rr.Code)
	}
	if rr := call(memberCookie, http.MethodGet, "/api/monitors", ""); rr.Code != http.StatusOK {
		t.Fatalf("with the requirement off the member gets in again, got %d", rr.Code)
	}
}
