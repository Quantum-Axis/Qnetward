# 10. Uptime monitors run inside the API process

## Status

Accepted.

## Context

Uptime monitors check a target on a schedule whether or not anyone has the
console open. Something has to own that schedule. The long-term plan has a
NATS event bus and outbound probe agents, but neither exists yet, and the
first monitors only need ICMP, TCP, and HTTP(S) checks from the server itself.

## Decision

`api.Server.RunMonitors` runs in the serve process, started by
`cmd/qnetward` unless `-monitors=false` (`QNETWARD_MONITORS=false`) is set.

- The `monitors` table holds `next_run_at`. Every five seconds the runner
  claims due monitors with `UPDATE ... WHERE id IN (SELECT ... FOR UPDATE
  SKIP LOCKED)` and moves each one an interval ahead in the same statement.
  Several instances can run side by side without checking a monitor twice.
- An ICMP or TCP check sends three probes. Any reply means up; no replies, or
  an error such as a failed lookup or a blocked address, means down. Each
  sample is written to the results table, so it shows in History, and the
  summary goes through the workspace's alert rules.
- An HTTP check is one GET through `internal/engine/httpcheck`, following up
  to five redirects, with a 10-second limit. It is up when the final status
  matches the expected one, or is any 2xx or 3xx when none is set. The
  transport uses no proxy and dials through the target policy's
  `DialControl`, so every connection, redirects included, is checked against
  the address actually used. Credentials in URLs are refused because monitors
  are visible to the whole workspace.
- HTTPS checks record the certificate's expiry. The first check that sees a
  certificate within 14 days of expiring sends one warning; a different
  certificate can warn again.
- At most eight checks run at once, separately from the limit on
  user-started probes.
- A change between up and down emails the monitor's notify address, if one is
  set.
- Checks are kept for 30 days in `monitor_checks`; the runner prunes older
  rows hourly.
- Monitors and checks belong to a workspace and are covered by the
  `workspace_isolation` policy ([ADR 0009](0009-row-level-security-via-tenant-role.md)).
  Claiming and pruning work across workspaces and run as the login role;
  recording a check uses a workspace transaction.

Intervals are limited to 1, 5, 10, 30, and 60 minutes, and a workspace can have
up to 20 monitors.

## Consequences

- Monitors work in every deployment with no extra service.
- Checks run from wherever the API runs, so there is one vantage point.
  Multi-region checks arrive with probe agents, which will claim work the same
  way.
- A claimed monitor whose process dies mid-check waits one interval for its
  next check.
- A single failed check marks a monitor down. If that proves noisy, a
  confirmation re-check can be added before the status flips.
