## What and why

<!-- One or two sentences: what changes, and the problem it solves. -->

## How to verify

<!-- Commands or steps a reviewer can run. -->

## Checklist

- [ ] `go vet ./...` and `go test ./...` pass
- [ ] Frontend: `pnpm lint` (and `pnpm build` if routing or config changed)
- [ ] New probe types or endpoints that take a target apply the target policy (ADR 0004)
- [ ] New tables holding customer data are scoped to a workspace (ADR 0003)
- [ ] Schema changes are backward compatible with existing data
- [ ] An ADR is added or updated if this changes an architectural decision
- [ ] No secrets, `.env` values, or personal data in the diff
