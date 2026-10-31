# 3. The workspace is the tenant boundary

## Status

Accepted

## Context

Qnetward is SaaS-first and must serve teams, MSPs with many clients, and
eventually enterprises. Data has to be isolated between customers, and a user
can belong to more than one customer's account.

The product plan describes an Organization → Project → Environment hierarchy.
The code already has workspaces with members, roles (owner, admin, member),
invitations, and an active workspace per session.

## Decision

A workspace is the unit of tenancy. Every customer-owned row will carry the
workspace it belongs to, and authorization checks are made against the user's
role in that workspace. A user can belong to several workspaces and switches
between them; the active workspace is stored on the session.

Projects and environments, if needed later, will live inside a workspace
rather than above it.

Isolation is enforced in application queries today. Moving the most sensitive
tables to PostgreSQL Row-Level Security is planned for Phase 1; it requires
the application's database role to not be a superuser, because superusers
bypass RLS.

## Consequences

- Every new table that holds customer data needs a workspace reference and
  scoped queries from day one.
- Probe results and alert rules are still scoped by user, not workspace. They
  need migrating before shared team history is possible.
- Store helpers named `...ByUserID` pick the user's own workspace implicitly.
  They exist for tests only; handlers must pass an explicit workspace ID.
