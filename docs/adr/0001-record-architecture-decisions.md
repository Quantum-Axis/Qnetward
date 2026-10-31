# 1. Record architecture decisions

## Status

Accepted

## Context

Several choices in this codebase aren't obvious from the code alone: why
probes refuse private addresses, why handlers are moving out of `cmd/`, why
the dashboard lives at `/overview`. Without a written record, a later
contributor can "clean up" one of these and only rediscover the reason after
it breaks.

## Decision

Record architecturally significant decisions as numbered Markdown files in
`docs/adr/`, using the context / decision / consequences format. A reversed
decision is superseded by a new ADR, not deleted.

## Consequences

- Before changing something an ADR covers, read it, then either follow it or
  write a new ADR explaining the change.
- A small writing cost on the decisions that qualify. Most changes need none.
