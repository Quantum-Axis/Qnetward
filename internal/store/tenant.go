package store

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"
)

// Tenant is the organization/workspace a user belongs to. Every user
// belongs to exactly one tenant today, auto-created at signup; inviting
// teammates into an existing tenant (a members join table) is a
// follow-up, not yet built.
type Tenant struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// withTenant runs fn inside a transaction with the Postgres session
// variable app.tenant_id set (via set_config, is_local=true) to tenantID
// for that transaction only. Row-Level Security policies that compare
// tenant_id = current_setting('app.tenant_id') then see the right value
// for exactly this transaction and nothing else — including for the
// table's own owning role, since the tables this is used against are
// created with FORCE ROW LEVEL SECURITY. It commits on success and rolls
// back on any error, including one returned by fn.
func (s *Store) withTenant(ctx context.Context, tenantID int64, fn func(*sql.Tx) error) error {
	if tenantID <= 0 {
		return errors.New("tenant id is required")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, `SELECT set_config('app.tenant_id', $1, true)`, strconv.FormatInt(tenantID, 10)); err != nil {
		return err
	}

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}
