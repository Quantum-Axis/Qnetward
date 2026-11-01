package store

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestCreateUserCreatesDefaultWorkspace(t *testing.T) {
	st := openTestStore(t, Config{
		BatchSize:     1,
		FlushInterval: 5 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	email := fmt.Sprintf("workspace-user-%d@example.com", time.Now().UnixNano())
	user, err := st.CreateUser(ctx, "Workspace Operator", email, "hash-1", false)
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}

	workspace, err := st.GetWorkspaceContextByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetWorkspaceContextByUserID returned error: %v", err)
	}
	if workspace.Name != "Workspace Operator Workspace" {
		t.Fatalf("unexpected workspace name: %q", workspace.Name)
	}
	if workspace.Role != "owner" {
		t.Fatalf("expected owner role, got %q", workspace.Role)
	}
	if workspace.MemberCount != 1 {
		t.Fatalf("expected member count 1, got %d", workspace.MemberCount)
	}
	if workspace.Plan != "starter" {
		t.Fatalf("expected starter plan, got %q", workspace.Plan)
	}
	if workspace.BillingStatus != "trial" {
		t.Fatalf("expected trial billing status, got %q", workspace.BillingStatus)
	}
}

func TestEnsureDefaultWorkspaceForUserBackfillsExistingUser(t *testing.T) {
	st := openTestStore(t, Config{
		BatchSize:     1,
		FlushInterval: 5 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	email := fmt.Sprintf("workspace-backfill-%d@example.com", time.Now().UnixNano())
	user, err := st.CreateUser(ctx, "Backfill Operator", email, "hash-1", false)
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}

	if _, err := st.db.ExecContext(ctx, `DELETE FROM workspace_members WHERE user_id = $1`, user.ID); err != nil {
		t.Fatalf("delete workspace_members: %v", err)
	}
	if _, err := st.db.ExecContext(ctx, `DELETE FROM workspaces WHERE owner_user_id = $1`, user.ID); err != nil {
		t.Fatalf("delete workspaces: %v", err)
	}

	workspace, err := st.EnsureDefaultWorkspaceForUser(ctx, user)
	if err != nil {
		t.Fatalf("EnsureDefaultWorkspaceForUser returned error: %v", err)
	}
	if workspace.ID == 0 {
		t.Fatal("expected created workspace id")
	}
	if workspace.MemberCount != 1 {
		t.Fatalf("expected member count 1, got %d", workspace.MemberCount)
	}
}

func TestWorkspaceMembersAndInvitations(t *testing.T) {
	st := openTestStore(t, Config{
		BatchSize:     1,
		FlushInterval: 5 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	email := fmt.Sprintf("workspace-team-%d@example.com", time.Now().UnixNano())
	user, err := st.CreateUser(ctx, "Team Owner", email, "hash-1", false)
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}

	members, err := st.ListWorkspaceMembersByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListWorkspaceMembersByUserID returned error: %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("expected one default member, got %d", len(members))
	}
	if !members[0].IsCurrent || members[0].Role != "owner" {
		t.Fatalf("unexpected default member: %#v", members[0])
	}

	invitation, err := st.CreateWorkspaceInvitationByUserID(ctx, user.ID, fmt.Sprintf("invite-%d@example.com", time.Now().UnixNano()), "admin", time.Now().Add(72*time.Hour))
	if err != nil {
		t.Fatalf("CreateWorkspaceInvitationByUserID returned error: %v", err)
	}
	if invitation.Role != "admin" || invitation.Status != "pending" {
		t.Fatalf("unexpected invitation: %#v", invitation)
	}

	invitations, err := st.ListWorkspaceInvitationsByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListWorkspaceInvitationsByUserID returned error: %v", err)
	}
	if len(invitations) != 1 {
		t.Fatalf("expected 1 invitation, got %d", len(invitations))
	}
	if invitations[0].Email != invitation.Email {
		t.Fatalf("expected invitation email %q, got %q", invitation.Email, invitations[0].Email)
	}
}

func TestIncomingWorkspaceInvitationAcceptsMembership(t *testing.T) {
	st := openTestStore(t, Config{
		BatchSize:     1,
		FlushInterval: 5 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ownerEmail := fmt.Sprintf("workspace-owner-%d@example.com", time.Now().UnixNano())
	owner, err := st.CreateUser(ctx, "Owner Operator", ownerEmail, "hash-owner", false)
	if err != nil {
		t.Fatalf("CreateUser owner returned error: %v", err)
	}

	inviteeEmail := fmt.Sprintf("workspace-invitee-%d@example.com", time.Now().UnixNano())
	invitation, err := st.CreateWorkspaceInvitationByUserID(ctx, owner.ID, inviteeEmail, "viewer", time.Now().Add(72*time.Hour))
	if err != nil {
		t.Fatalf("CreateWorkspaceInvitationByUserID returned error: %v", err)
	}

	invitee, err := st.CreateUser(ctx, "Invited Operator", inviteeEmail, "hash-invitee", false)
	if err != nil {
		t.Fatalf("CreateUser invitee returned error: %v", err)
	}

	incoming, err := st.ListIncomingWorkspaceInvitationsByEmail(ctx, inviteeEmail)
	if err != nil {
		t.Fatalf("ListIncomingWorkspaceInvitationsByEmail returned error: %v", err)
	}
	if len(incoming) != 1 {
		t.Fatalf("expected one incoming invitation, got %d", len(incoming))
	}
	if incoming[0].WorkspaceName == "" {
		t.Fatal("expected workspace name on incoming invitation")
	}

	accepted, err := st.AcceptWorkspaceInvitationByUserID(ctx, invitee.ID, invitation.ID)
	if err != nil {
		t.Fatalf("AcceptWorkspaceInvitationByUserID returned error: %v", err)
	}
	if accepted.Status != "accepted" {
		t.Fatalf("expected accepted status, got %q", accepted.Status)
	}

	members, err := st.ListWorkspaceMembersByUserID(ctx, owner.ID)
	if err != nil {
		t.Fatalf("ListWorkspaceMembersByUserID returned error: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("expected 2 workspace members after accept, got %d", len(members))
	}
}

func TestWorkspaceOwnerCanUpdateAndRemoveMember(t *testing.T) {
	st := openTestStore(t, Config{
		BatchSize:     1,
		FlushInterval: 5 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ownerEmail := fmt.Sprintf("workspace-owner-manage-%d@example.com", time.Now().UnixNano())
	owner, err := st.CreateUser(ctx, "Owner Manager", ownerEmail, "hash-owner", false)
	if err != nil {
		t.Fatalf("CreateUser owner returned error: %v", err)
	}
	memberEmail := fmt.Sprintf("workspace-member-manage-%d@example.com", time.Now().UnixNano())
	member, err := st.CreateUser(ctx, "Team Member", memberEmail, "hash-member", false)
	if err != nil {
		t.Fatalf("CreateUser member returned error: %v", err)
	}

	invitation, err := st.CreateWorkspaceInvitationByUserID(ctx, owner.ID, member.Email, "member", time.Now().Add(72*time.Hour))
	if err != nil {
		t.Fatalf("CreateWorkspaceInvitationByUserID returned error: %v", err)
	}
	if _, err := st.AcceptWorkspaceInvitationByUserID(ctx, member.ID, invitation.ID); err != nil {
		t.Fatalf("AcceptWorkspaceInvitationByUserID returned error: %v", err)
	}

	updated, err := st.UpdateWorkspaceMemberRoleByUserID(ctx, owner.ID, member.ID, "admin")
	if err != nil {
		t.Fatalf("UpdateWorkspaceMemberRoleByUserID returned error: %v", err)
	}
	if updated.Role != "admin" {
		t.Fatalf("expected admin role, got %q", updated.Role)
	}

	if err := st.RemoveWorkspaceMemberByUserID(ctx, owner.ID, member.ID); err != nil {
		t.Fatalf("RemoveWorkspaceMemberByUserID returned error: %v", err)
	}

	members, err := st.ListWorkspaceMembersByUserID(ctx, owner.ID)
	if err != nil {
		t.Fatalf("ListWorkspaceMembersByUserID returned error: %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("expected 1 member after removal, got %d", len(members))
	}
}

func TestWorkspaceAdminCannotManageOwner(t *testing.T) {
	st := openTestStore(t, Config{
		BatchSize:     1,
		FlushInterval: 5 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ownerEmail := fmt.Sprintf("workspace-owner-admin-%d@example.com", time.Now().UnixNano())
	owner, err := st.CreateUser(ctx, "Owner Root", ownerEmail, "hash-owner", false)
	if err != nil {
		t.Fatalf("CreateUser owner returned error: %v", err)
	}
	adminEmail := fmt.Sprintf("workspace-admin-%d@example.com", time.Now().UnixNano())
	admin, err := st.CreateUser(ctx, "Admin User", adminEmail, "hash-admin", false)
	if err != nil {
		t.Fatalf("CreateUser admin returned error: %v", err)
	}

	invitation, err := st.CreateWorkspaceInvitationByUserID(ctx, owner.ID, admin.Email, "admin", time.Now().Add(72*time.Hour))
	if err != nil {
		t.Fatalf("CreateWorkspaceInvitationByUserID returned error: %v", err)
	}
	if _, err := st.AcceptWorkspaceInvitationByUserID(ctx, admin.ID, invitation.ID); err != nil {
		t.Fatalf("AcceptWorkspaceInvitationByUserID returned error: %v", err)
	}

	// The admin also owns a personal workspace, so act explicitly in the owner's
	// workspace rather than letting a ByUserID helper pick the admin's own.
	ownerWorkspace, err := st.GetWorkspaceContextByUserID(ctx, owner.ID)
	if err != nil {
		t.Fatalf("GetWorkspaceContextByUserID returned error: %v", err)
	}
	if _, err := st.UpdateWorkspaceMemberRoleByWorkspaceIDAndUserID(ctx, ownerWorkspace.ID, admin.ID, owner.ID, "member"); err != ErrWorkspacePermissionDenied {
		t.Fatalf("expected permission denied updating owner, got %v", err)
	}
	if err := st.RemoveWorkspaceMemberByWorkspaceIDAndUserID(ctx, ownerWorkspace.ID, admin.ID, owner.ID); err != ErrWorkspacePermissionDenied {
		t.Fatalf("expected permission denied removing owner, got %v", err)
	}
}

func TestWorkspaceInvitationPermissionsAndRevocation(t *testing.T) {
	st := openTestStore(t, Config{
		BatchSize:     1,
		FlushInterval: 5 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ownerEmail := fmt.Sprintf("workspace-owner-invite-%d@example.com", time.Now().UnixNano())
	owner, err := st.CreateUser(ctx, "Owner Invite", ownerEmail, "hash-owner", false)
	if err != nil {
		t.Fatalf("CreateUser owner returned error: %v", err)
	}
	workspace, err := st.GetWorkspaceContextByUserID(ctx, owner.ID)
	if err != nil {
		t.Fatalf("GetWorkspaceContextByUserID returned error: %v", err)
	}

	adminEmail := fmt.Sprintf("workspace-admin-invite-%d@example.com", time.Now().UnixNano())
	admin, err := st.CreateUser(ctx, "Admin Invite", adminEmail, "hash-admin", false)
	if err != nil {
		t.Fatalf("CreateUser admin returned error: %v", err)
	}
	adminInvitation, err := st.CreateWorkspaceInvitationByWorkspaceIDAndUserID(ctx, workspace.ID, owner.ID, admin.Email, "admin", time.Now().Add(72*time.Hour))
	if err != nil {
		t.Fatalf("CreateWorkspaceInvitationByWorkspaceIDAndUserID admin returned error: %v", err)
	}
	if _, err := st.AcceptWorkspaceInvitationByUserID(ctx, admin.ID, adminInvitation.ID); err != nil {
		t.Fatalf("AcceptWorkspaceInvitationByUserID admin returned error: %v", err)
	}

	memberEmail := fmt.Sprintf("workspace-member-invite-%d@example.com", time.Now().UnixNano())
	member, err := st.CreateUser(ctx, "Member Invite", memberEmail, "hash-member", false)
	if err != nil {
		t.Fatalf("CreateUser member returned error: %v", err)
	}
	memberInvitation, err := st.CreateWorkspaceInvitationByWorkspaceIDAndUserID(ctx, workspace.ID, owner.ID, member.Email, "member", time.Now().Add(72*time.Hour))
	if err != nil {
		t.Fatalf("CreateWorkspaceInvitationByWorkspaceIDAndUserID member returned error: %v", err)
	}
	if _, err := st.AcceptWorkspaceInvitationByUserID(ctx, member.ID, memberInvitation.ID); err != nil {
		t.Fatalf("AcceptWorkspaceInvitationByUserID member returned error: %v", err)
	}

	if _, err := st.CreateWorkspaceInvitationByWorkspaceIDAndUserID(ctx, workspace.ID, admin.ID, fmt.Sprintf("blocked-admin-%d@example.com", time.Now().UnixNano()), "admin", time.Now().Add(72*time.Hour)); err != ErrWorkspacePermissionDenied {
		t.Fatalf("expected admin invite permission denied, got %v", err)
	}
	if _, err := st.CreateWorkspaceInvitationByWorkspaceIDAndUserID(ctx, workspace.ID, member.ID, fmt.Sprintf("blocked-member-%d@example.com", time.Now().UnixNano()), "viewer", time.Now().Add(72*time.Hour)); err != ErrWorkspacePermissionDenied {
		t.Fatalf("expected member invite permission denied, got %v", err)
	}

	revoked, err := st.CreateWorkspaceInvitationByWorkspaceIDAndUserID(ctx, workspace.ID, owner.ID, fmt.Sprintf("revoke-%d@example.com", time.Now().UnixNano()), "viewer", time.Now().Add(72*time.Hour))
	if err != nil {
		t.Fatalf("CreateWorkspaceInvitationByWorkspaceIDAndUserID revoke returned error: %v", err)
	}
	revokedInvitation, err := st.CancelWorkspaceInvitationByWorkspaceIDAndUserID(ctx, workspace.ID, owner.ID, revoked.ID)
	if err != nil {
		t.Fatalf("CancelWorkspaceInvitationByWorkspaceIDAndUserID returned error: %v", err)
	}
	if revokedInvitation.Status != "revoked" {
		t.Fatalf("expected revoked status, got %q", revokedInvitation.Status)
	}
}
