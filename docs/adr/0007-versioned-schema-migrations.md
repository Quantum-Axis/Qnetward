# 7. Versioned, embedded schema migrations

## Status

Accepted

## Context

The schema was created by `setupSchema` in `internal/store`: a long run of
`CREATE ... IF NOT EXISTS` and `ALTER ... ADD COLUMN IF NOT EXISTS` statements
executed on every start. Nothing recorded which changes a database had seen.
That works for adding columns. It breaks down for renames, backfills, and
anything that must run exactly once, and those are coming: row-level security
for tenants (ADR 0003), team-owned history, and the monitor and agent tables in
Phase 2.

The plan named golang-migrate or Atlas. Both expect one static schema per
database. Our results table has a configurable name, and the Postgres-backed
tests give each test its own results table in a shared database.

## Decision

- Shared relational tables (users, workspaces, sessions, tokens, alert rules,
  and everything added later) are changed only through numbered SQL files in
  `internal/store/migrations/` (`0001_baseline.sql`, `0002_...`).
- The files are embedded in the binary. A small runner in
  `internal/store/migrate.go` applies pending files in order, each in its own
  transaction, and records them in `schema_migrations`. A Postgres advisory
  lock keeps concurrent instances from racing.
- Migrations are forward-only. An applied file is never edited; a mistake is
  fixed by a new migration. Rollback is a new migration or a restore.
- `0001_baseline.sql` is the schema as `setupSchema` left it. Its statements
  are idempotent, so existing databases adopt it without changes.
- The time-series results table stays outside the versioned migrations. The
  store ensures it under its configured name on open, together with the
  TimescaleDB hypertable when `QNETWARD_TIMESCALE` is on.
- `qnetward serve` migrates on start. `qnetward migrate` applies migrations as
  a separate deploy step, and `qnetward migrate -status` lists them.

## Consequences

- Schema history is in git and in each database, and a deploy can run
  migrations before new code starts.
- No third-party migration dependency, at the cost of about 200 lines we
  maintain.
- Changes to the results table (new columns, indexes) go in
  `ensureResultsTable` and must stay idempotent. If that table ever needs a
  one-time data change, it should get a fixed name and move into the versioned
  migrations.
- A migration that can't run inside a transaction, such as
  `CREATE INDEX CONCURRENTLY`, needs runner support before it can be added.
