# 3. Manage schema changes with golang-migrate instead of inline DDL

## Status

Accepted

## Context

`internal/store.setupSchema` ran roughly 35 inline `CREATE TABLE`,
`ALTER TABLE`, and `CREATE INDEX` statements — all guarded with
`IF NOT EXISTS`/`ADD COLUMN IF NOT EXISTS` — on every server startup, plus
ad hoc historical fixups (a `DELETE ... USING ...` dedup, a couple of
`DROP INDEX IF EXISTS` statements cleaning up earlier naming mistakes).
This worked, but had real costs:

- The current schema had to be reconstructed by reading the accumulated
  fixups in order; there was no single file describing "the schema" at
  any past point in time.
- There was no record of *when* a column or index was added relative to
  a release, which matters for anyone trying to reason about a database
  that's a few versions behind.
- Every new schema change meant adding another idempotent statement to
  the same growing function, with no forcing function to think about
  forward/backward compatibility or rollback.

## Decision

Use [`golang-migrate/migrate`](https://github.com/golang-migrate/migrate)
with the `postgres` database driver and an `iofs` source backed by
`//go:embed migrations/*.sql`. Each schema change is a numbered pair of
`NNNN_description.up.sql` / `NNNN_description.down.sql` files in
`internal/store/migrations/`, applied via `migrate.Up()` on startup
(`internal/store/migrate.go`).

The first migration (`0001_init`) is a byte-for-byte replica of the prior
inline DDL, so it is a safe no-op against any database the old code
already created — this was required to ship the change without a
separate manual migration step for existing deployments.

`ensureCustomResultsTable` (in `store.go`) remains as a separate,
non-migration code path used only when `Config.TableName` overrides the
default `results` table name — exclusively a test-isolation mechanism, not
something production ever exercises (production always uses the default
name, which `runMigrations` handles).

## Consequences

- Every schema change from here on is a new numbered migration pair, not
  an edit to `0001_init.up.sql` or an added statement in `setupSchema`.
- Rolling back a migration (`migrate.Down()`) is possible in principle,
  but `runMigrations` only ever calls `Up()`; a manual rollback still
  requires running the down migration by hand today. Wiring an operator
  path for that is a follow-up, not part of this decision.
- Migrations run automatically on every `netwatcher serve` startup. For a
  destructive migration, this means the *first* replica to start after a
  deploy applies it — acceptable at NetWatcher's current scale, but worth
  revisiting (e.g. gating destructive migrations behind an explicit flag)
  before running multiple replicas against one database in production.
