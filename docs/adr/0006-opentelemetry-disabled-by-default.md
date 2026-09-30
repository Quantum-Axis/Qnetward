# 6. OpenTelemetry for tracing and metrics, disabled by default

## Status

Accepted

## Context

NetWatcher had no tracing or metrics instrumentation. Diagnosing a slow
request or a probe that silently stopped firing meant reading logs
(`log.Printf` calls scattered through `internal/store` and
`cmd/netwatcher`) and guessing, with no latency breakdown across the HTTP
handler, the probe execution, and the database write, and no way to see
trends (probe success rate, alert evaluation volume) without querying the
database directly.

Any observability library that requires a collector to be running just to
start the server would be a step backward for anyone running NetWatcher
without one (which, at this stage, is the common case).

## Decision

Add `internal/telemetry`, using the OpenTelemetry Go SDK. It exports
traces and metrics over OTLP/HTTP when explicitly configured, and is a
true no-op otherwise:

- `telemetry.Setup` returns immediately (touching no global state at all)
  when `NETWATCHER_OTEL_ENDPOINT` (or the standard
  `OTEL_EXPORTER_OTLP_ENDPOINT`) is unset.
- Instrumentation call sites (`telemetry.RecordProbe`,
  `telemetry.RecordAlertEvaluation`) always call the same global
  `otel.Meter`/`otel.Tracer` API regardless of whether telemetry is
  enabled; OpenTelemetry's own global proxy makes this safe — instruments
  created before `Setup` runs are transparently upgraded once a real
  provider is installed, and stay no-ops forever if it never is.

When enabled:

- The whole HTTP server is wrapped with `otelhttp`, giving every request a
  span named `METHOD /path`.
- The Postgres connection goes through `otelsql.Open` instead of
  `sql.Open`, giving every query a span and exporting connection pool
  gauges (open/idle/in-use), with no per-call-site changes needed anywhere
  in `internal/store`.
- `netwatcher.probes.total` / `netwatcher.probe.duration` (labeled by
  `protocol`, `success`) and `netwatcher.alerts.evaluated` /
  `netwatcher.alerts.triggered` (labeled by `protocol`) cover the
  application-specific signals that generic HTTP/DB instrumentation can't
  see on its own.

This was verified against a real (mock) OTLP/HTTP collector — both via
`internal/telemetry`'s own tests and by running the actual `netwatcher
serve` binary against a standalone collector process and confirming real
HTTP traffic produced trace and metric exports.

## Consequences

- Running NetWatcher without an OTel collector configured is unaffected —
  no startup dependency, no behavior change, no performance cost beyond
  the no-op instrument calls.
- Enabling telemetry currently means every span/metric for every
  tenant flows through one shared `TracerProvider`/`MeterProvider`, tagged
  by resource attributes (service name/version) but not yet by tenant.
  Per-tenant telemetry filtering, if ever needed, is a separate decision.
- New instrumentation should go through `internal/telemetry`'s helpers
  (or the global `otel.Tracer`/`otel.Meter` API directly for ad hoc spans)
  rather than introducing a second telemetry mechanism.
