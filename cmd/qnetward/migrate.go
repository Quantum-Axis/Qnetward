package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"qnetward/internal/store"
)

// runMigrate applies or reports schema migrations. `serve` also migrates on
// start, so this exists for deploy pipelines that run schema changes first.
func runMigrate(args []string) {
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	dbDSN := fs.String("db-dsn", envString("QNETWARD_DB_DSN", ""), "postgres/timescaledb DSN")
	status := fs.Bool("status", false, "list migrations and whether each is applied, without changing anything")
	timeout := fs.Duration("timeout", 2*time.Minute, "give up after this long")
	_ = fs.Parse(args)

	if err := migrateCommand(os.Stdout, strings.TrimSpace(*dbDSN), *status, *timeout); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}

func migrateCommand(out io.Writer, dsn string, statusOnly bool, timeout time.Duration) error {
	if dsn == "" {
		return fmt.Errorf("set -db-dsn or QNETWARD_DB_DSN")
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if statusOnly {
		statuses, err := store.MigrationStatuses(ctx, dsn)
		if err != nil {
			return err
		}
		for _, s := range statuses {
			state := "pending"
			if s.AppliedAt != nil {
				state = "applied " + s.AppliedAt.UTC().Format(time.RFC3339)
			}
			fmt.Fprintf(out, "%04d_%s  %s\n", s.Version, s.Name, state)
		}
		return nil
	}

	ran, err := store.Migrate(ctx, dsn)
	for _, m := range ran {
		fmt.Fprintf(out, "applied %04d_%s\n", m.Version, m.Name)
	}
	if err != nil {
		return err
	}
	if len(ran) == 0 {
		fmt.Fprintln(out, "schema is up to date")
	}
	return nil
}
