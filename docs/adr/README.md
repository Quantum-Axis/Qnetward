# Architecture Decision Records

This directory records the architecturally significant decisions made in
this codebase: the ones that are expensive to reverse, or that a future
contributor needs the *reasoning* for, not just the resulting code.

Format: one Markdown file per decision, numbered sequentially, using the
lightweight template in [0001](0001-record-architecture-decisions.md)
(context / decision / consequences). A decision is superseded by writing a
new ADR that says so and updating the old one's status — never by editing
history in place, since the point is to keep the trail of *why* intact.

## Index

| # | Title | Status |
|---|-------|--------|
| [0001](0001-record-architecture-decisions.md) | Record architecture decisions | Accepted |
| [0002](0002-split-main-into-domain-files.md) | Split cmd/netwatcher/main.go into domain files | Accepted |
| [0003](0003-schema-migrations-with-golang-migrate.md) | Manage schema changes with golang-migrate instead of inline DDL | Accepted |
| [0004](0004-multi-tenant-model-with-row-level-security.md) | Multi-tenant data model enforced with PostgreSQL Row-Level Security | Accepted |
| [0005](0005-defer-rls-on-results-table.md) | Defer RLS enforcement on the results table | Accepted |
| [0006](0006-opentelemetry-disabled-by-default.md) | OpenTelemetry for tracing and metrics, disabled by default | Accepted |
| [0007](0007-replace-minio-with-seaweedfs.md) | Replace MinIO with SeaweedFS in the production stack | Accepted |
| [0008](0008-adopt-apps-and-pkg-top-level-layout.md) | Adopt `apps/` and `pkg/` top-level directories | Accepted |
