# Qnetward

Qnetward is a self-hosted network diagnostics toolkit built with Go and a Next.js frontend. It combines a CLI, a backend API/probe engine, realtime probe streaming, and a production stack built around PostgreSQL/TimescaleDB, Redis, and S3-compatible object storage.


## Contributors

- MOPARA PAIR AYAT
- MONABBER EAMAN

## Overview

You run the server, open the dashboard in your browser, and inspect live ICMP, TCP, or port-scan results as they arrive. The app uses:

- PostgreSQL/TimescaleDB for durable time-series history
- Redis for fast history caching
- S3-compatible object storage for exported history artifacts

Core capabilities:
- ICMP ping
- TCP ping
- TCP port scan
- Realtime WebSocket streaming
- Stop/cancel for live runs
- PostgreSQL/TimescaleDB history
- Redis-backed history cache
- Export endpoint with optional S3 upload
- Latency chart + result table
- Scrollable live result view
- Local-first and production deployment model

## Architecture

```
apps/web/              Next.js dashboard
cmd/qnetward/          CLI entrypoint and backend API/probe server
internal/engine/       pure network engines (no HTTP, no DB)
  ping/  tcpping/  portscan/  dnslookup/  report/
internal/platform/     infrastructure adapters
  cache/ (Redis)  objectstore/ (S3)  mailer/ (Resend)
internal/store/        PostgreSQL/TimescaleDB persistence, batching, retention
deploy/                production docker-compose stack
scripts/               smoke tests and tooling
```

Inside `cmd/qnetward`, code is split by domain: `auth_handlers.go`,
`session.go`, `workspace_handlers.go`, `admin_handlers.go`,
`probe_handlers.go`, `history_handlers.go`, `alert_handlers.go`, `ws.go`,
plus shared `httpx.go`, `origin.go`, `ratelimit.go`, and `config.go`.

## Tech Stack

- Go
- Next.js
- TypeScript
- PostgreSQL
- TimescaleDB
- Redis
- S3-compatible object storage
- Gorilla WebSocket
- Tailwind CSS

## Features

### CLI

```bash
./qnetward ping 8.8.8.8
./qnetward tcpping example.com --port 443
./qnetward portscan --ports 22,80,443 example.com
```

### Web UI

- Realtime results
- Live latency chart
- History loading
- Port scan mode
- Mobile-friendly layout
- Stop button for active WebSocket runs

### API

- `POST /api/ping`
- `POST /api/tcpping`
- `POST /api/portscan`
- `GET /api/history`
- `POST /api/export/history`
- `GET /healthz`
- `GET /ws`

## Quick Start

### Windows

```powershell
go mod tidy
go build -o qnetward.exe .\cmd\qnetward
.\qnetward.exe serve --listen 127.0.0.1:8080 --db-dsn "postgres://qnetward:change-me@127.0.0.1:5432/qnetward?sslmode=disable" --timescale
```

Open:

```text
backend: http://127.0.0.1:8080
frontend: http://127.0.0.1:3000
```

### Linux

```bash
go mod tidy
go build -o qnetward ./cmd/qnetward
./qnetward serve --listen 127.0.0.1:8080 --db-dsn "postgres://qnetward:change-me@127.0.0.1:5432/qnetward?sslmode=disable" --timescale
```

Open:

```text
backend: http://127.0.0.1:8080
frontend: http://127.0.0.1:3000
```

For ICMP on Linux, root or `CAP_NET_RAW` may be required:

```bash
sudo setcap cap_net_raw+ep ./qnetward
```

## Example API Requests

### TCP Ping

```bash
curl -X POST http://127.0.0.1:8080/api/tcpping \
  -H "Content-Type: application/json" \
  -d '{"host":"example.com","port":443,"count":4,"interval_ms":1000,"timeout_ms":2000}'
```

### History

```bash
curl "http://127.0.0.1:8080/api/history?type=tcpping&host=example.com&port=443&limit=20"
```

### Export History

Download JSON:

```bash
curl -X POST http://127.0.0.1:8080/api/export/history \
  -H "Content-Type: application/json" \
  -d '{"type":"tcpping","host":"example.com","port":443,"limit":50,"format":"json"}'
```

Upload CSV to configured S3-compatible object storage:

```bash
curl -X POST http://127.0.0.1:8080/api/export/history \
  -H "Content-Type: application/json" \
  -d '{"type":"tcpping","host":"example.com","port":443,"limit":50,"format":"csv","destination":"s3"}'
```

## Production Stack

Recommended production stack:

- PostgreSQL + TimescaleDB
- Redis
- S3-compatible object storage

Environment variables:

```bash
QNETWARD_DB_DSN=postgres://qnetward:change-me@timescaledb:5432/qnetward?sslmode=disable
QNETWARD_TIMESCALE=true
QNETWARD_REDIS_ADDR=redis:6379
QNETWARD_S3_ENDPOINT=minio:9000
QNETWARD_S3_ACCESS_KEY=minioadmin
QNETWARD_S3_SECRET_KEY=change-me
QNETWARD_S3_BUCKET=qnetward-artifacts
QNETWARD_S3_SSL=false
QNETWARD_DEMO_ENABLED=false
QNETWARD_ADMIN_EMAILS=admin@example.com
```

Start the production stack:

```bash
cp .env.example .env
docker compose -f deploy/docker-compose.production.yml --env-file .env up --build
```

Health check response now includes dependency status for:

- database
- redis
- object storage

## Contributor Demo Access

For temporary contributor access, enable demo mode and ask contributors to use the login page.

Demo mode:

- adds a `Continue as Demo` button on `/login`
- signs contributors into a shared temporary demo account
- allows the full dashboard and tool routes
- uses the real application data and behavior of that demo account

## Development

Build:

```bash
go build -o qnetward ./cmd/qnetward
```

Run internal tests:

```bash
go test ./internal/...
```

Run vet:

```bash
go vet ./...
```

Run the HTTP smoke test against a running local stack:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/smoke-test.ps1 -BaseUrl http://127.0.0.1:8080
```

GitHub Actions runs on every pull request and every push to `main`:

- `go vet ./...`
- `go test -race -p 1 ./...` against a TimescaleDB service, so Postgres-backed tests run too
- `apps/web` `pnpm lint` and `pnpm build`

Playwright E2E and the Docker smoke test are run locally (see above).

## Notes

- This project is built for self-hosted use.
- If you expose it publicly, put a trusted reverse proxy, tunnel, or access control layer in front of it.
- Runtime artifacts such as `*.log` and `*.exe` are ignored by `.gitignore`.
- WebSocket is the primary realtime path. HTTP is used as a fallback path.
- Production mode is designed around PostgreSQL/TimescaleDB + Redis + S3-compatible storage.

## Current Scope

Implemented now:
- ICMP ping
- TCP ping
- TCP port scan
- Realtime Web UI
- Realtime result streaming
- History storage
- Stop/cancel for active live runs
- DNS lookup / inspector
- Alert rules with email delivery
- Workspaces, team invitations, and platform admin console

Planned next:
- Multi-target monitoring
- HTTP/HTTPS checks
- Full DNS toolkit (DNSSEC, propagation)
