package store

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestAdminOverviewAndListings(t *testing.T) {
	st := openTestStore(t, Config{
		BatchSize:     1,
		FlushInterval: 5 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ownerEmail := fmt.Sprintf("admin-owner-%d@example.com", time.Now().UnixNano())
	owner, err := st.CreateUser(ctx, "Admin Owner", ownerEmail, "hash-owner", false)
	if err != nil {
		t.Fatalf("CreateUser owner returned error: %v", err)
	}

	memberEmail := fmt.Sprintf("admin-member-%d@example.com", time.Now().UnixNano())
	member, err := st.CreateUser(ctx, "Admin Member", memberEmail, "hash-member", false)
	if err != nil {
		t.Fatalf("CreateUser member returned error: %v", err)
	}

	if err := st.CreateSession(ctx, owner.ID, fmt.Sprintf("session-owner-%d", time.Now().UnixNano()), time.Now().Add(24*time.Hour)); err != nil {
		t.Fatalf("CreateSession owner returned error: %v", err)
	}
	if err := st.CreateSession(ctx, member.ID, fmt.Sprintf("session-member-%d", time.Now().UnixNano()), time.Now().Add(24*time.Hour)); err != nil {
		t.Fatalf("CreateSession member returned error: %v", err)
	}

	invitation, err := st.CreateWorkspaceInvitationByUserID(ctx, owner.ID, member.Email, "admin", time.Now().Add(72*time.Hour))
	if err != nil {
		t.Fatalf("CreateWorkspaceInvitationByUserID returned error: %v", err)
	}
	if _, err := st.AcceptWorkspaceInvitationByUserID(ctx, member.ID, invitation.ID); err != nil {
		t.Fatalf("AcceptWorkspaceInvitationByUserID returned error: %v", err)
	}

	overview, err := st.GetAdminOverview(ctx)
	if err != nil {
		t.Fatalf("GetAdminOverview returned error: %v", err)
	}
	if overview.UserCount < 2 {
		t.Fatalf("expected at least 2 users, got %d", overview.UserCount)
	}
	if overview.WorkspaceCount < 2 {
		t.Fatalf("expected at least 2 workspaces, got %d", overview.WorkspaceCount)
	}
	if overview.SessionCount < 2 {
		t.Fatalf("expected at least 2 active sessions, got %d", overview.SessionCount)
	}

	users, err := st.ListAdminUsers(ctx, 10)
	if err != nil {
		t.Fatalf("ListAdminUsers returned error: %v", err)
	}
	if len(users) < 2 {
		t.Fatalf("expected at least 2 admin users, got %d", len(users))
	}

	var ownerSummary *AdminUserSummary
	for i := range users {
		if users[i].Email == owner.Email {
			ownerSummary = &users[i]
			break
		}
	}
	if ownerSummary == nil {
		t.Fatalf("expected owner %q in admin users list", owner.Email)
	}
	if ownerSummary.OwnedWorkspaceCount != 1 {
		t.Fatalf("expected 1 owned workspace for owner, got %d", ownerSummary.OwnedWorkspaceCount)
	}
	if ownerSummary.ActiveSessionCount < 1 {
		t.Fatalf("expected active session count for owner, got %d", ownerSummary.ActiveSessionCount)
	}

	workspaces, err := st.ListAdminWorkspaces(ctx, 10)
	if err != nil {
		t.Fatalf("ListAdminWorkspaces returned error: %v", err)
	}
	if len(workspaces) < 2 {
		t.Fatalf("expected at least 2 admin workspaces, got %d", len(workspaces))
	}

	var ownerWorkspace *AdminWorkspaceSummary
	for i := range workspaces {
		if workspaces[i].OwnerEmail == owner.Email {
			ownerWorkspace = &workspaces[i]
			break
		}
	}
	if ownerWorkspace == nil {
		t.Fatalf("expected workspace owned by %q in admin workspaces list", owner.Email)
	}
	if ownerWorkspace.MemberCount != 2 {
		t.Fatalf("expected member count 2 for owner workspace, got %d", ownerWorkspace.MemberCount)
	}
}
