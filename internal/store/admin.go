package store

import (
	"context"
	"time"
)

type AdminOverview struct {
	UserCount              int `json:"user_count"`
	VerifiedUserCount      int `json:"verified_user_count"`
	WorkspaceCount         int `json:"workspace_count"`
	InvitationCount        int `json:"invitation_count"`
	PendingInvitationCount int `json:"pending_invitation_count"`
	SessionCount           int `json:"session_count"`
}

type AdminUserSummary struct {
	ID                  int64      `json:"id"`
	Name                string     `json:"name"`
	Email               string     `json:"email"`
	EmailVerifiedAt     *time.Time `json:"email_verified_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	OwnedWorkspaceCount int        `json:"owned_workspace_count"`
	WorkspaceCount      int        `json:"workspace_count"`
	ActiveSessionCount  int        `json:"active_session_count"`
}

type AdminWorkspaceSummary struct {
	ID                     int64     `json:"id"`
	Name                   string    `json:"name"`
	Plan                   string    `json:"plan"`
	BillingStatus          string    `json:"billing_status"`
	CreatedAt              time.Time `json:"created_at"`
	OwnerUserID            int64     `json:"owner_user_id"`
	OwnerName              string    `json:"owner_name"`
	OwnerEmail             string    `json:"owner_email"`
	MemberCount            int       `json:"member_count"`
	PendingInvitationCount int       `json:"pending_invitation_count"`
	ActiveSessionCount     int       `json:"active_session_count"`
}

func (s *Store) GetAdminOverview(ctx context.Context) (AdminOverview, error) {
	var overview AdminOverview
	err := s.db.QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(*) FROM users),
			(SELECT COUNT(*) FROM users WHERE email_verified_at IS NOT NULL),
			(SELECT COUNT(*) FROM workspaces),
			(SELECT COUNT(*) FROM workspace_invitations),
			(SELECT COUNT(*) FROM workspace_invitations WHERE status = 'pending'),
			(SELECT COUNT(*) FROM sessions WHERE expires_at > NOW())
	`).Scan(
		&overview.UserCount,
		&overview.VerifiedUserCount,
		&overview.WorkspaceCount,
		&overview.InvitationCount,
		&overview.PendingInvitationCount,
		&overview.SessionCount,
	)
	return overview, err
}

func (s *Store) ListAdminUsers(ctx context.Context, limit int) ([]AdminUserSummary, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			u.id,
			u.name,
			u.email,
			u.email_verified_at,
			u.created_at,
			(
				SELECT COUNT(*)
				FROM workspaces owned
				WHERE owned.owner_user_id = u.id
			) AS owned_workspace_count,
			(
				SELECT COUNT(*)
				FROM workspace_members memberships
				WHERE memberships.user_id = u.id
			) AS workspace_count,
			(
				SELECT COUNT(*)
				FROM sessions active_sessions
				WHERE active_sessions.user_id = u.id
					AND active_sessions.expires_at > NOW()
			) AS active_session_count
		FROM users u
		ORDER BY u.created_at DESC, u.id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]AdminUserSummary, 0, limit)
	for rows.Next() {
		var user AdminUserSummary
		if err := rows.Scan(
			&user.ID,
			&user.Name,
			&user.Email,
			&user.EmailVerifiedAt,
			&user.CreatedAt,
			&user.OwnedWorkspaceCount,
			&user.WorkspaceCount,
			&user.ActiveSessionCount,
		); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (s *Store) ListAdminWorkspaces(ctx context.Context, limit int) ([]AdminWorkspaceSummary, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			w.id,
			w.name,
			w.plan,
			w.billing_status,
			w.created_at,
			owner.id,
			owner.name,
			owner.email,
			(
				SELECT COUNT(*)
				FROM workspace_members wm
				WHERE wm.workspace_id = w.id
			) AS member_count,
			(
				SELECT COUNT(*)
				FROM workspace_invitations wi
				WHERE wi.workspace_id = w.id
					AND wi.status = 'pending'
			) AS pending_invitation_count,
			(
				SELECT COUNT(*)
				FROM sessions s
				WHERE s.active_workspace_id = w.id
					AND s.expires_at > NOW()
			) AS active_session_count
		FROM workspaces w
		INNER JOIN users owner ON owner.id = w.owner_user_id
		ORDER BY w.created_at DESC, w.id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	workspaces := make([]AdminWorkspaceSummary, 0, limit)
	for rows.Next() {
		var workspace AdminWorkspaceSummary
		if err := rows.Scan(
			&workspace.ID,
			&workspace.Name,
			&workspace.Plan,
			&workspace.BillingStatus,
			&workspace.CreatedAt,
			&workspace.OwnerUserID,
			&workspace.OwnerName,
			&workspace.OwnerEmail,
			&workspace.MemberCount,
			&workspace.PendingInvitationCount,
			&workspace.ActiveSessionCount,
		); err != nil {
			return nil, err
		}
		workspaces = append(workspaces, workspace)
	}
	return workspaces, rows.Err()
}
