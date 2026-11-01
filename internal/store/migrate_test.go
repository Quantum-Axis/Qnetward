package store

import (
	"context"
	"strings"
	"testing"
)

func TestLoadMigrationsAreOrderedAndNamed(t *testing.T) {
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatalf("loadMigrations: %v", err)
	}
	if len(migrations) == 0 {
		t.Fatal("expected at least the baseline migration")
	}
	if migrations[0].Version != 1 || migrations[0].Name != "baseline" {
		t.Fatalf("first migration = %04d_%s, want 0001_baseline", migrations[0].Version, migrations[0].Name)
	}
	for i := 1; i < len(migrations); i++ {
		if migrations[i].Version <= migrations[i-1].Version {
			t.Fatalf("migrations out of order: %d after %d", migrations[i].Version, migrations[i-1].Version)
		}
	}
	for _, m := range migrations {
		if strings.TrimSpace(m.SQL) == "" {
			t.Fatalf("migration %04d_%s is empty", m.Version, m.Name)
		}
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	s := openTestStore(t, Config{})
	ctx := context.Background()

	// The store already migrated on open; running again must be a no-op.
	ran, err := Migrate(ctx, s.cfg.DSN)
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	if len(ran) != 0 {
		t.Fatalf("expected no pending migrations, ran %d", len(ran))
	}

	statuses, err := MigrationStatuses(ctx, s.cfg.DSN)
	if err != nil {
		t.Fatalf("MigrationStatuses: %v", err)
	}
	for _, st := range statuses {
		if st.AppliedAt == nil {
			t.Fatalf("migration %04d_%s not applied", st.Version, st.Name)
		}
	}
}
