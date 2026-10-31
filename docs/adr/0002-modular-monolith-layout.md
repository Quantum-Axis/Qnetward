# 2. Modular monolith with engine / platform / modules layers

## Status

Accepted

## Context

All HTTP handlers, configuration, and wiring lived in a single 4,600-line
`cmd/qnetward/main.go`. Ownership was unclear, merges conflicted, and handlers
depended on package-level globals, which made them hard to test in isolation.

The roadmap eventually needs separately scaled services (ingestion, scanner
workers), but the team is small and a single deployable is far cheaper to run.

## Decision

Keep one deployable binary and enforce boundaries with packages:

- `internal/engine/` holds pure network logic (ping, tcpping, portscan,
  dnslookup, report). It imports nothing else from `internal/`.
- `internal/platform/` holds infrastructure adapters (cache, objectstore,
  mailer, netguard). It knows nothing about business rules.
- `internal/modules/` (in progress) will hold business domains (identity,
  workspace, probe, history, alerting), each owning its handlers and SQL,
  with dependencies injected instead of read from globals.
- `cmd/qnetward/` ends up as wiring only.
- `apps/web/` holds the Next.js frontend.

Dependencies point one way: `modules` → `engine`, `platform`. Modules don't
reach into each other's internals.

`main.go` was first split into domain files within `package main` as a pure
move, so the refactor stays reviewable. Moving those files into
`internal/modules/` comes next.

## Consequences

- A module can later become its own service by adding a new `cmd/` entry
  point, without reshaping the code.
- Until the module move finishes, handlers still read package-level globals
  in `cmd/qnetward`.
