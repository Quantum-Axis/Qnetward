# 8. One HTTP package, domain packages on demand

## Status

Accepted. Amends [ADR 0002](0002-modular-monolith-layout.md).

## Context

ADR 0002 planned to move the handlers out of `cmd/qnetward` into one package
per business domain under `internal/modules/` (identity, workspace, probe,
history, alerting), with dependencies injected instead of read from globals.

Doing that move showed how much the domains share. Every handler resolves the
session, checks the request origin, decodes JSON with the same limits, and
writes the same error shape. Alert evaluation runs inside probe handlers, and
history and probes share target parsing. Separate packages would have to
export all of that from a common package and import each other for cross-cutting
calls. That adds surface without changing what ships: one binary, one process.

The goals behind ADR 0002 were: `cmd/` holds wiring only, no globals, and
handlers can be tested in isolation.

## Decision

- `internal/api` holds the whole HTTP and WebSocket layer, one file per domain
  (`auth_handlers.go`, `workspace_handlers.go`, `probe_handlers.go`, ...).
  Handlers are methods on `*api.Server`, built by
  `api.New(api.Deps{...}, api.Options{...})`. The package has no mutable
  package-level state.
- `internal/probe` holds what the CLI and the API share about probes: input
  validation, default limits, and the JSON report shape.
- `cmd/qnetward` parses flags and environment, opens the store and adapters,
  wires `api.Server`, and runs the CLI commands. Nothing else.
- A domain gets its own package when it gets its own deploy (the planned
  ingestion and scanner services) or grows enough that a package boundary helps
  review. The extraction starts from the domain's file in `internal/api`.

## Consequences

- The goals of ADR 0002 hold: wiring-only `cmd/`, no globals, each test builds
  its own server.
- Package boundaries inside the API are by file, not enforced by the compiler.
  Reviewers watch for one domain's file reaching into another's helpers.
- `internal/modules/` doesn't exist. The dependency direction from ADR 0002
  still holds: `api` → `probe`, `store`, `engine`, `platform`, never the
  reverse.
