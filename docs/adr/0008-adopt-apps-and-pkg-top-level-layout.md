# 8. Adopt `apps/` and `pkg/` top-level directories

## Status

Accepted

## Context

An earlier planning report sketched a target monorepo layout for this
project at full scale — `apps/`, `services/` (one directory per future
microservice: `api-gateway`, `identity`, `billing`, `inventory`,
`scheduler`, `ingestion`, `security-engine`, `scanner-workers`, …),
`agents/`, `drivers/`, `pkg/`, `proto/`, `sdk/`, `plugins/`. That same
report's own architecture section (Section 3) recommends the opposite of
building that out immediately: *"start with a modular monolith, not
microservices from day one. Only the parts that need separate scale
(ingestion, scanner) become separate services later."*

Creating the full `services/*` tree now would mean a dozen empty
directories with no code in them — every one of `identity`, `billing`,
`inventory`, `scheduler`, `ingestion`, `security-engine`,
`scanner-workers`, `connector-manager`, `notifier`, `reporting` names a
feature domain that doesn't exist in this codebase yet. That's scaffolding
for a shape the project hasn't grown into, not a structural fix for a real
problem today.

## Decision

Adopt only the parts of the target layout that correspond to something
that actually exists and benefits from the move now:

- `frontend/` → `apps/web/`. The target layout's `apps/web` ("Next.js
  dashboard") is exactly what this directory already is; moving it costs
  nothing structurally and matches the target layout's naming today, not
  hypothetically.
- `internal/telemetry/` → `pkg/telemetry/`. The target layout's `pkg/`
  is described as "shared: telemetry, errors, crypto, tenancy" — code with
  no dependency on this service's own business logic, meant to be reused
  if and when a package ever moves into a separate service. `telemetry`
  is exactly that: it depends on nothing in `internal/store` or
  `cmd/netwatcher`, and every call site reaches it through the global
  `otel.Tracer`/`otel.Meter` API (see ADR 6), so nothing about moving it
  changes behavior.
- Everything else — `cmd/netwatcher`, `internal/store`, `internal/ping`,
  etc. — stays exactly where it is. The backend remains one deployable,
  modular monolith, per the report's own recommendation and per ADR 2.
  No `services/*`, `agents/*`, `drivers/*`, `proto/*`, `sdk/*`, or
  `plugins/*` directories are created: there is nothing to put in them
  yet, and an empty directory documents an aspiration, not an engineering
  decision.

Docker Compose service names (`frontend`, `netwatcher`) and the compose
file's own structure are unchanged — only the build `context:` path for
`frontend` moved from `../frontend` to `../apps/web`. Renaming compose
service identities was judged unnecessary churn for a source-layout move.

## Consequences

- `go.mod`'s module path (`netwatcher`) is unchanged; import paths for
  moved code update to `netwatcher/pkg/telemetry`, everything else
  unaffected.
- ADR 2 and ADR 6 describe decisions made before this move and are left
  as written — rewriting a past ADR's own text to match a later rename
  would misrepresent what was true when it was made (see ADR 1). Where
  they say `internal/telemetry`, read `pkg/telemetry`: the decision they
  record (splitting `main.go`; OpenTelemetry disabled by default) is
  unaffected by where the code physically lives.
- The next real trigger for creating a `services/*` entry is the same one
  the original report named: a part of this monolith that needs to scale
  or deploy independently (its own example: ingestion, the scanner). Until
  that need is concrete, new feature domains (billing, security-engine,
  etc.) are built as packages inside this monolith, not as new empty
  top-level directories.
