package store

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
)

// tenantRole is the restricted database role that workspace-scoped work runs
// as. Row-level security policies on workspace-owned tables apply to it, so a
// query that forgets its workspace filter still can't read another
// workspace's rows. See ADR 0009 and migration 0005.
const tenantRole = "qnetward_tenant"

// beginWorkspaceTx starts a transaction that runs as tenantRole with
// app.workspace_id set. Both settings end with the transaction, so the pooled
// connection goes back clean. The caller must commit or roll back.
func (s *Store) beginWorkspaceTx(ctx context.Context, workspaceID int64) (*sql.Tx, error) {
	if workspaceID <= 0 {
		return nil, errors.New("workspace id is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx,
		`SELECT set_config('role', $1, true), set_config('app.workspace_id', $2, true)`,
		tenantRole, strconv.FormatInt(workspaceID, 10),
	); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	return tx, nil
}

// enableWorkspaceRLS turns on row-level security for a table with a
// workspace_id column and lets tenantRole use it within its workspace. It is
// safe to run on every start: an existing policy is left alone.
func enableWorkspaceRLS(ctx context.Context, conn *sql.Conn, table, policy string) error {
	var exists bool
	if err := conn.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM pg_policy WHERE polname = $1 AND polrelid = $2::regclass)`,
		policy, table,
	).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	quotedPolicy := quoteIdentifier(policy)
	statements := []string{
		`GRANT SELECT, INSERT, UPDATE, DELETE ON ` + table + ` TO ` + tenantRole,
		`ALTER TABLE ` + table + ` ENABLE ROW LEVEL SECURITY`,
		`CREATE POLICY ` + quotedPolicy + ` ON ` + table + ` TO ` + tenantRole + `
			USING (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::bigint)
			WITH CHECK (workspace_id = NULLIF(current_setting('app.workspace_id', true), '')::bigint)`,
	}
	for _, statement := range statements {
		if _, err := conn.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}
