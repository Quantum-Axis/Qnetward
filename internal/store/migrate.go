package store

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// runMigrations applies every pending migration in internal/store/migrations
// to db, using golang-migrate's own schema_migrations table to track what
// has already run. It covers the fixed-name tables (users, sessions,
// password_reset_tokens, email_verification_codes, alert_rules) and the
// default-named "results" table; a non-default results table name (used
// only by tests, via Config.TableName) is handled separately by
// ensureCustomResultsTable, since a migration file can't be parameterized
// by a runtime-chosen identifier.
func runMigrations(db *sql.DB) error {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("load embedded migrations: %w", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("init migration driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return fmt.Errorf("init migrator: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
