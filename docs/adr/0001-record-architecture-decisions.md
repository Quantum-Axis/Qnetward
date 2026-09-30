# 1. Record architecture decisions

## Status

Accepted

## Context

NetWatcher's `cmd/netwatcher` and `internal/store` packages carry decisions
that are not obvious from the diff alone: why schema changes go through
`golang-migrate` instead of the inline `CREATE TABLE IF NOT EXISTS`
statements they replaced, why `alert_rules` gets real database-enforced
tenant isolation but `results` doesn't, why telemetry is wired in but off
by default. Without a written record, each of these looks like it could be
"cleaned up" by a contributor who wasn't there for the reasoning, and the
reasoning has to be re-derived (or re-litigated) from scratch.

## Decision

We record architecturally significant decisions as numbered Markdown files
in `docs/adr/`, following the format Michael Nygard described for ADRs:
title, status, context, decision, consequences. A decision that gets
reversed or replaced is superseded by a new ADR, not by deleting or
silently rewriting the old one.

Not every change needs an ADR. Write one when a decision is expensive to
reverse, cuts across multiple packages, or trades off something a later
contributor would otherwise have to rediscover the hard way (e.g. by
reverting it and hitting the reason it existed).

## Consequences

- Future changes that touch one of these decisions should read the
  relevant ADR first, and either follow it or write a new ADR explaining
  the change.
- This adds a small amount of writing overhead to decisions that qualify.
  Most day-to-day changes (bug fixes, new endpoints that follow existing
  patterns) don't need one.
