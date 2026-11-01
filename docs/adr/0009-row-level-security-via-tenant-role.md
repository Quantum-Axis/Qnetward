# 9. Row-level security through a restricted tenant role

## Status

Accepted. Builds on [ADR 0003](0003-workspace-is-the-tenant.md).

## Context

Probe results and alert rules carry a `workspace_id`, and every query filters
on it. One forgotten filter would still leak one customer's data to another,
so ADR 0003 planned PostgreSQL row-level security (RLS) as a second line of
defence.

RLS does not apply to superusers, to roles with `BYPASSRLS`, or to a table's
owner unless the table forces it. On Neon, the roles created from the console
have `BYPASSRLS`, and the application role owns the tables it creates. Plain
policies would do nothing there.

We could ask every deployment to connect with a separate, unprivileged login
role. That adds setup to every environment, and the schema setup and
retention jobs would still need a second, privileged connection.

## Decision

Migration 0005 creates a `NOLOGIN` role, `qnetward_tenant`, without
`BYPASSRLS`, and grants it to the login role. The role gets row access to
workspace-owned tables, and each table has a `workspace_isolation` policy for
it that matches `workspace_id` against the `app.workspace_id` setting.

Store methods that act for a request open their transaction with
`beginWorkspaceTx`. It sets `role` and `app.workspace_id` with
`set_config(..., true)` in one round trip, so both end with the transaction
and the pooled connection goes back clean. If the setting is missing, the
policy matches nothing.

System work keeps the login role's rights:

- migrations and the results-table setup and backfill,
- retention cleanup,
- the batched results insert, which mixes workspaces in one transaction and
  takes its workspace from the server-side session rather than user input,
- account deletion, which removes rows across the user's own workspaces.

The probe results table gets its policy from `enableWorkspaceRLS` at startup,
because its name is configurable and it sits outside the versioned
migrations.

## Consequences

- A tenant-path query without a workspace filter returns only the current
  workspace's rows, and writes into another workspace fail.
- Each tenant-path call is now a short transaction with one extra statement.
- Every new workspace-owned table needs a grant to `qnetward_tenant` and a
  `workspace_isolation` policy, and its request-path queries must go through
  `beginWorkspaceTx`.
- The login role needs `CREATEROLE` when migration 0005 first runs, or the
  role must be created beforehand by an administrator.
- Code that runs as the login role is not covered. Keep that path small and
  limited to the system work above.
