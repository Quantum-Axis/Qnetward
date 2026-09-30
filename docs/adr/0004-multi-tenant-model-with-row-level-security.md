# 4. Multi-tenant data model enforced with PostgreSQL Row-Level Security

## Status

Accepted

## Context

NetWatcher's data model was single-tenant: every row (`users`,
`alert_rules`, `results`) was scoped only by `user_id`, with no concept of
an organization or workspace above the individual user. Supporting teams —
multiple people sharing alert rules and history for the same monitored
targets — needs a tenant concept, and once that concept exists, the more
important question is *where* cross-tenant isolation is enforced.

Filtering by `tenant_id` in every Go query is the obvious first approach,
but it puts the entire isolation guarantee on remembering to add that
`WHERE` clause correctly in every current and future query — one missed
clause in one handler is a cross-tenant data leak, and nothing catches it
until it's exploited or reported.

## Decision

Add a `tenants` table. Every user belongs to exactly one tenant,
auto-provisioned at signup (`Store.CreateUser` creates a tenant named
`"{name}'s workspace"` and the user in one transaction). `alert_rules`
gets a `tenant_id` column with Row-Level Security **enabled and forced**
(`ALTER TABLE alert_rules FORCE ROW LEVEL SECURITY`), plus a policy that
compares `tenant_id` against a per-transaction session variable:

```sql
CREATE POLICY tenant_isolation ON alert_rules
    USING (tenant_id = current_setting('app.tenant_id', true)::bigint)
    WITH CHECK (tenant_id = current_setting('app.tenant_id', true)::bigint);
```

The application sets that session variable per request via
`Store.withTenant` (`internal/store/tenant.go`), which wraps the operation
in a transaction and runs `set_config('app.tenant_id', $1, true)` — scoped
to the transaction (`is_local = true`) so it can never leak into a pooled
connection reused by a different tenant's request.

This means the database itself refuses a cross-tenant `alert_rules` row —
verified directly: connected as the application's own Postgres role (not a
superuser, see below), a session scoped to one tenant sees zero rows and
updates zero rows when it names another tenant's alert rule by ID.

**This requires the connecting role not be a Postgres superuser.**
Superusers bypass Row-Level Security unconditionally, regardless of
`FORCE ROW LEVEL SECURITY` — this is documented Postgres behavior, not a
bug. The `netwatcher` role used in this environment was created as a
superuser, which would have made any RLS test give a false pass (every
query would succeed regardless of the policy). It was demoted
(`ALTER ROLE netwatcher NOSUPERUSER`) specifically so that the enforcement
test above is meaningful; any production deployment must use a non-
superuser role for the application's own database connection for the same
reason.

`users` also gets a `tenant_id` column (`NOT NULL`, backfilled) but no RLS
policy: today, a user's own row is always looked up by session/email, not
listed across tenants, so there's no cross-tenant read path to close yet.

Migration `0002_multitenant` backfills `tenant_id` for any existing users
and alert rules — one auto-named tenant per existing user — before making
the column `NOT NULL`, so it applies cleanly to a database that predates
tenants.

## Consequences

- Every `Store` method that reads or writes `alert_rules` now takes a
  `tenantID` and goes through `withTenant` (or manages the same
  `set_config` call itself, as `EvaluateAlertRules` does, since it needs
  the transaction for other statements too).
- A future query against `alert_rules` that forgets to scope by tenant in
  Go is still safe: the database rejects the cross-tenant rows regardless.
  A query that forgets tenant scoping against `users` or `results` is
  *not* protected this way (see ADR 5 for `results`).
- There is no "invite a teammate into an existing tenant" flow yet — every
  signup gets its own new tenant. A members/invitations join table is a
  follow-up, not part of this decision.
