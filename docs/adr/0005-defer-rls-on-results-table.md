# 5. Defer RLS enforcement on the results table

## Status

Accepted

## Context

ADR 4 enforces tenant isolation on `alert_rules` at the database level
using Row-Level Security with a per-transaction `set_config('app.tenant_id', ...)`
session variable. The `results` table — every ping/tcpping probe result —
is the highest-write-volume table in the system, and the natural next
candidate for the same protection.

`results` is written by `Store.worker()`, a single background goroutine
per `Store` that batches `ResultRecord`s pulled off a channel into one
multi-row `INSERT` per flush interval (`insertBatch`). Requests from many
different users, and therefore many different tenants, enqueue records
onto the same channel and get flushed together in whatever batch happens
to be open at flush time. RLS's `current_setting('app.tenant_id', true)`
mechanism is inherently per-transaction and per-session: it answers "which
single tenant is this transaction allowed to touch," which has no correct
value for a batch that mixes rows from several tenants in one `INSERT`.

Making `results` RLS-safe under the current write path would mean either:

1. One transaction (and one `set_config` call) per tenant represented in
   a batch, splitting each flush into N sub-batches — adds real latency
   and complexity to the hot write path for every probe result, not just
   alert rule changes (which are comparatively rare).
2. Moving to a per-tenant write queue/worker — a bigger structural change
   to `internal/store`.

Neither is justified by the actual risk today: every read of `results`
(`QueryHistory`, `QueryRecentHistoryTargets`) already filters by `user_id`,
which is unique across tenants, so there is no live cross-tenant read path
through the existing API that this would additionally close.

## Decision

Add `tenant_id` to `results` (`NOT NULL`, backfilled by migration
`0002_multitenant`, indexed) but do **not** enable Row-Level Security on
it. The column exists now so that:

- Future per-tenant views/aggregations (e.g. "history across everyone on
  my team") don't need another migration to add it later.
- If the write path is ever restructured (option 1 or 2 above), enabling
  RLS at that point is a policy addition, not another schema migration.

This is a deliberate, documented gap, not an oversight: `insertBatch` and
the `INSERT INTO ... VALUES ($1,...),($1,...),...` it generates are
unrelated to `set_config`, so nothing here silently assumes protection
that isn't there.

## Consequences

- `results` isolation remains enforced only by the existing `user_id`
  filter in application code (`QueryHistory`, `QueryRecentHistoryTargets`,
  the export handler). A new read path added later must remember to filter
  by `user_id` (or `tenant_id`, once relevant) itself — the database will
  not catch a missing filter here the way it does for `alert_rules`.
- If `results` ever needs a "read across my whole tenant" feature, revisit
  this ADR: that's exactly the point at which the async batch-write
  incompatibility above needs a real answer, not just a documented gap.
