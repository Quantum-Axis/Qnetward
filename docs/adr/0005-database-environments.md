# 5. Neon for development, TimescaleDB optional

## Status

Accepted

## Context

The application needs PostgreSQL. Probe results benefit from TimescaleDB
hypertables, but nothing else depends on Timescale features, and old results
are deleted by the application rather than by Timescale retention policies.

The primary development machine can't run Docker, so the bundled Compose stack
isn't available for day-to-day work.

## Decision

- Development uses a Neon project (`qnetward`, region `aws-ap-southeast-1`)
  with two branches: `main` for running the app and `test` for
  Postgres-backed tests (`QNETWARD_TEST_POSTGRES_DSN`).
- `QNETWARD_TIMESCALE=false` is the default for development; the results table
  is a plain table there.
- The application connects as a non-superuser role (`qnetward_app`), so
  Row-Level Security will behave the same as in production.
- Production is expected to run PostgreSQL with TimescaleDB, on Timescale
  Cloud or self-managed on AWS, per the product plan.
- CI runs the Postgres-backed tests against a TimescaleDB service container.

## Consequences

- Tests over a remote database are slower; test waits must tolerate network
  latency instead of assuming a local database.
- Code must not rely on Timescale-only SQL outside the code path guarded by
  `QNETWARD_TIMESCALE`.
- Connection strings live only in `.env` (git-ignored), never in the repo.
