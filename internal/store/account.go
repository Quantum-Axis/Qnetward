package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	// ErrOwnsSharedWorkspace means the account owns a workspace that other people
	// still belong to, so deleting it would delete their workspace too.
	ErrOwnsSharedWorkspace = errors.New("account owns a workspace with other members")
	ErrSessionNotOwned     = errors.New("session not found for this user")
)

// SessionMeta describes where a session was created.
type SessionMeta struct {
	UserAgent string
	IPAddress string
}

// SessionInfo is one active session as shown to its owner.
type SessionInfo struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	UserAgent string    `json:"user_agent"`
	IPAddress string    `json:"ip_address"`
	Current   bool      `json:"current"`
}

const maxUserAgentLength = 300

// CreateSessionWithMeta stores a new session along with the browser and address that opened it.
func (s *Store) CreateSessionWithMeta(ctx context.Context, userID int64, tokenHash string, expiresAt time.Time, meta SessionMeta) error {
	ua := strings.TrimSpace(meta.UserAgent)
	if len(ua) > maxUserAgentLength {
		ua = ua[:maxUserAgentLength]
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (user_id, token_hash, expires_at, user_agent, ip_address)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, tokenHash, expiresAt.UTC(), ua, strings.TrimSpace(meta.IPAddress))
	return err
}

// ListUserSessions returns the user's unexpired sessions, newest first, marking the one with currentHash.
func (s *Store) ListUserSessions(ctx context.Context, userID int64, currentHash string) ([]SessionInfo, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, created_at, expires_at, user_agent, ip_address, token_hash = $2
		FROM sessions
		WHERE user_id = $1 AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 100
	`, userID, currentHash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SessionInfo{}
	for rows.Next() {
		var info SessionInfo
		if err := rows.Scan(&info.ID, &info.CreatedAt, &info.ExpiresAt, &info.UserAgent, &info.IPAddress, &info.Current); err != nil {
			return nil, err
		}
		out = append(out, info)
	}
	return out, rows.Err()
}

// DeleteUserSession signs out one of the user's sessions by ID.
func (s *Store) DeleteUserSession(ctx context.Context, userID, sessionID int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = $1 AND user_id = $2`, sessionID, userID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrSessionNotOwned
	}
	return nil
}

// DeleteOtherUserSessions signs the user out everywhere except the session with keepHash.
func (s *Store) DeleteOtherUserSessions(ctx context.Context, userID int64, keepHash string) (int64, error) {
	result, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = $1 AND token_hash <> $2`, userID, keepHash)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// GetUserPasswordHash returns the stored bcrypt hash for the user.
func (s *Store) GetUserPasswordHash(ctx context.Context, userID int64) (string, error) {
	var hash string
	err := s.db.QueryRowContext(ctx, `SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&hash)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrUserNotFound
	}
	return hash, err
}

// ChangeUserPassword replaces the password hash and ends every other session,
// so a leaked password stops working everywhere but here.
func (s *Store) ChangeUserPassword(ctx context.Context, userID int64, passwordHash, keepSessionHash string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	result, err := tx.ExecContext(ctx, `UPDATE users SET password_hash = $1 WHERE id = $2`, passwordHash, userID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return ErrUserNotFound
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM sessions WHERE user_id = $1 AND token_hash <> $2`, userID, keepSessionHash); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM password_reset_tokens WHERE user_id = $1`, userID); err != nil {
		return err
	}
	return tx.Commit()
}

// SharedWorkspacesOwnedBy lists workspaces the user owns that have other members.
func (s *Store) SharedWorkspacesOwnedBy(ctx context.Context, userID int64) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT w.name
		FROM workspaces w
		WHERE w.owner_user_id = $1
			AND EXISTS (SELECT 1 FROM workspace_members m WHERE m.workspace_id = w.id AND m.user_id <> $1)
		ORDER BY w.name
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// DeleteUserAccount removes the user and everything that cascades from the
// user row: sessions, tokens, profile, and the workspaces they own alone, with
// those workspaces' alert rules and probe results. Results and rules they added
// to other teams' workspaces stay with those teams. It refuses while they own a
// workspace others use.
func (s *Store) DeleteUserAccount(ctx context.Context, userID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var shared bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM workspaces w
			WHERE w.owner_user_id = $1
				AND EXISTS (SELECT 1 FROM workspace_members m WHERE m.workspace_id = w.id AND m.user_id <> $1)
		)
	`, userID).Scan(&shared); err != nil {
		return err
	}
	if shared {
		return ErrOwnsSharedWorkspace
	}
	// Results have no foreign keys (the table may be a hypertable), so the owned
	// workspaces' rows are removed here before the cascade drops the workspaces.
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
		DELETE FROM %s
		WHERE workspace_id IN (SELECT id FROM workspaces WHERE owner_user_id = $1)
			OR (workspace_id IS NULL AND user_id = $1)
	`, s.tableRef()), userID); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, userID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return ErrUserNotFound
	}
	return tx.Commit()
}

// GetUserAvatarKey returns the object key of the user's avatar, or "" if none.
func (s *Store) GetUserAvatarKey(ctx context.Context, userID int64) (string, error) {
	var key string
	err := s.db.QueryRowContext(ctx, `SELECT avatar_key FROM user_profiles WHERE user_id = $1`, userID).Scan(&key)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return key, err
}

// SetUserAvatarKey records the avatar's object key ("" clears it) and returns the previous one.
func (s *Store) SetUserAvatarKey(ctx context.Context, userID int64, key string) (string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }()

	var previous string
	err = tx.QueryRowContext(ctx, `SELECT avatar_key FROM user_profiles WHERE user_id = $1 FOR UPDATE`, userID).Scan(&previous)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO user_profiles (user_id, avatar_key, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id) DO UPDATE SET avatar_key = EXCLUDED.avatar_key, updated_at = NOW()
	`, userID, key); err != nil {
		if isForeignKeyViolation(err) {
			return "", ErrUserNotFound
		}
		return "", err
	}
	return previous, tx.Commit()
}
