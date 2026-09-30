# 2. Split cmd/netwatcher/main.go into domain files

## Status

Accepted

## Context

`cmd/netwatcher/main.go` had grown to 3684 lines, mixing the CLI entry
point, HTTP route table and bootstrap, request validation, every HTTP
handler (probes, history, alerts, auth, websocket), rate limiting, CORS
origin checks, and small JSON/HTTP helpers in one file. Finding the code
for a given concern meant scrolling or grepping through unrelated code,
and reviewing a change to, say, alert rules pulled in a diff hunk from a
file that also contained the websocket upgrade logic.

## Decision

Split `main.go` along existing conceptual boundaries into separate files
in the same package (`cmd/netwatcher`), with **zero behavior change**:

- `main.go` — entrypoint, `runServe` (route table + bootstrap), env/flag
  helpers, package-level globals.
- `cli.go` — the `ping`/`tcpping`/`portscan`/`dns` CLI subcommands.
- `validate.go` — request normalization/validation shared by handlers.
- `probes_http.go` — `/api/ping`, `/api/tcpping`, `/api/portscan`,
  `/api/dnslookup` handlers.
- `history.go` — `/api/history`, recent-targets, export.
- `alerts.go` — `/api/alerts/rules` and alert evaluation.
- `auth.go` — signup/login/session/password-reset/email-verification.
- `websocket.go` — the `/ws` streaming protocol.
- `ratelimit.go`, `origin.go`, `httputil.go` — small cross-cutting
  helpers.

The split was done mechanically, extracting exact top-level declaration
line ranges (via `go/parser`/`go/ast`) rather than by hand, specifically
to avoid introducing incidental behavior changes while moving thousands of
lines. It was verified against the pre-split behavior by running the full
test suite, `go vet`, and `go test -race` before and after and confirming
identical results.

## Consequences

- A change to one concern (e.g. alert rules) now touches one file instead
  of one region of a 3684-line file, and its diff doesn't intermix with
  unrelated handlers.
- All files stay in `package main`; this is a file-level reorganization
  within `cmd/netwatcher`, not a package boundary or module restructuring.
  Introducing internal packages for these concerns (e.g. an `internal/api`
  package) is a separate, larger decision, deliberately not taken here.
- New handlers for a concern should go in that concern's file rather than
  back into `main.go`.
