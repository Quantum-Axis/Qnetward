package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strconv"
	"time"
)

// Versioned, forward-only migrations for the shared relational schema.
//
// Files live in migrations/ as NNNN_name.sql and are embedded in the binary.
// Each runs once, inside its own transaction, and is recorded in
// schema_migrations. Applied files must never be edited: add a new file instead.
//
// The time-series results table is not migrated here. Its name is configurable
// (tests give each run its own table), so the store ensures it separately; see
// ensureResultsTable.

//go:embed migrations/*.sql
var migrationFiles embed.FS

var migrationName = regexp.MustCompile(`^(\d{4})_([a-z0-9_]+)\.sql$`)

// Migration is one versioned schema change.
type Migration struct {
	Version int
	Name    string
	SQL     string
}

// MigrationStatus reports whether a migration has been applied, and when.
type MigrationStatus struct {
	Migration
	AppliedAt *time.Time
}

func loadMigrations() ([]Migration, error) {
	entries, err := fs.ReadDir(migrationFiles, "migrations")
	if err != nil {
		return nil, err
	}
	var out []Migration
	seen := map[int]string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		match := migrationName.FindStringSubmatch(entry.Name())
		if match == nil {
			return nil, fmt.Errorf("migration %q: name must look like 0001_add_table.sql", entry.Name())
		}
		version, _ := strconv.Atoi(match[1])
		if prev, dup := seen[version]; dup {
			return nil, fmt.Errorf("migration version %04d used by both %s and %s", version, prev, entry.Name())
		}
		seen[version] = entry.Name()
		body, err := migrationFiles.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return nil, err
		}
		out = append(out, Migration{Version: version, Name: match[2], SQL: string(body)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}

func ensureMigrationTable(ctx context.Context, conn *sql.Conn) error {
	_, err := conn.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version BIGINT PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	return err
}

func appliedMigrations(ctx context.Context, conn *sql.Conn) (map[int]time.Time, error) {
	rows, err := conn.QueryContext(ctx, `SELECT version, applied_at FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	applied := map[int]time.Time{}
	for rows.Next() {
		var version int
		var at time.Time
		if err := rows.Scan(&version, &at); err != nil {
			return nil, err
		}
		applied[version] = at
	}
	return applied, rows.Err()
}

// runMigrations applies every pending migration in order. The caller must hold
// the schema advisory lock so concurrent instances don't race.
func runMigrations(ctx context.Context, conn *sql.Conn) ([]Migration, error) {
	migrations, err := loadMigrations()
	if err != nil {
		return nil, err
	}
	if err := ensureMigrationTable(ctx, conn); err != nil {
		return nil, fmt.Errorf("create schema_migrations: %w", err)
	}
	applied, err := appliedMigrations(ctx, conn)
	if err != nil {
		return nil, fmt.Errorf("read schema_migrations: %w", err)
	}

	var ran []Migration
	for _, m := range migrations {
		if _, ok := applied[m.Version]; ok {
			continue
		}
		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return ran, err
		}
		if _, err := tx.ExecContext(ctx, m.SQL); err != nil {
			_ = tx.Rollback()
			return ran, fmt.Errorf("migration %04d_%s: %w", m.Version, m.Name, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version, name) VALUES ($1, $2)`, m.Version, m.Name); err != nil {
			_ = tx.Rollback()
			return ran, fmt.Errorf("record migration %04d_%s: %w", m.Version, m.Name, err)
		}
		if err := tx.Commit(); err != nil {
			return ran, fmt.Errorf("commit migration %04d_%s: %w", m.Version, m.Name, err)
		}
		ran = append(ran, m)
	}
	return ran, nil
}

// withSchemaLock runs fn on a dedicated connection holding the schema advisory lock.
func withSchemaLock(ctx context.Context, db *sql.DB, fn func(conn *sql.Conn) error) error {
	if err := db.PingContext(ctx); err != nil {
		return err
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, schemaSetupLockID); err != nil {
		return err
	}
	defer func() {
		_, _ = conn.ExecContext(context.Background(), `SELECT pg_advisory_unlock($1)`, schemaSetupLockID)
	}()
	return fn(conn)
}

// Migrate applies pending migrations without opening a full Store. It backs the
// `qnetward migrate` command, so schema changes can run as a deploy step.
func Migrate(ctx context.Context, dsn string) ([]Migration, error) {
	db, err := openDB(applyDefaults(Config{DSN: dsn}))
	if err != nil {
		return nil, err
	}
	defer db.Close()
	var ran []Migration
	err = withSchemaLock(ctx, db, func(conn *sql.Conn) error {
		var runErr error
		ran, runErr = runMigrations(ctx, conn)
		return runErr
	})
	return ran, err
}

// MigrationStatuses lists every known migration and whether it has been applied.
func MigrationStatuses(ctx context.Context, dsn string) ([]MigrationStatus, error) {
	migrations, err := loadMigrations()
	if err != nil {
		return nil, err
	}
	db, err := openDB(applyDefaults(Config{DSN: dsn}))
	if err != nil {
		return nil, err
	}
	defer db.Close()
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if err := ensureMigrationTable(ctx, conn); err != nil {
		return nil, err
	}
	applied, err := appliedMigrations(ctx, conn)
	if err != nil {
		return nil, err
	}
	out := make([]MigrationStatus, 0, len(migrations))
	for _, m := range migrations {
		status := MigrationStatus{Migration: m}
		if at, ok := applied[m.Version]; ok {
			at := at
			status.AppliedAt = &at
		}
		out = append(out, status)
	}
	return out, nil
}
