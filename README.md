<div align="center">

# Qnetward

### Network visibility, without the guesswork.

Run live network diagnostics, watch results stream in real time, and keep a searchable history — on infrastructure you control.

<br />

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go&logoColor=white)
![Next.js](https://img.shields.io/badge/Next.js-16-000000?style=flat-square&logo=nextdotjs&logoColor=white)
![TypeScript](https://img.shields.io/badge/TypeScript-5-3178C6?style=flat-square&logo=typescript&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-TimescaleDB-4169E1?style=flat-square&logo=postgresql&logoColor=white)
![Redis](https://img.shields.io/badge/Cache-Redis-DC382D?style=flat-square&logo=redis&logoColor=white)
![License](https://img.shields.io/badge/License-See_repository-64748B?style=flat-square)

<br />

[Get started](#-get-started) · [Explore features](#-what-you-can-do) · [Architecture](#-how-it-fits-together) · [Production](#-production-deployment)

</div>

---

## ✨ What you can do

| Diagnose | Observe | Operate |
| --- | --- | --- |
| ICMP and TCP ping | Live WebSocket result streams | Workspaces and team invitations |
| TCP port scans | Latency charts and result tables | Alert rules with email delivery |
| DNS lookup and inspection | Searchable probe history | Platform admin console |
| Export results as JSON or CSV | Redis-backed history cache | Self-host the full stack |

Qnetward pairs a Go probe engine with a Next.js dashboard. Run it locally for a focused diagnostic session, or deploy the production stack for persistent history, alerts, and team workflows.

## 🚀 Get started

### Requirements

- Go 1.25+
- Node.js and pnpm for the web dashboard
- PostgreSQL for persistent history (TimescaleDB is supported)

### 1. Start the Go backend

From the repository root:

```powershell
go build -o qnetward.exe .\cmd\qnetward
.\qnetward.exe serve --listen 127.0.0.1:8080 --db-dsn "postgres://qnetward:change-me@127.0.0.1:5432/qnetward?sslmode=disable" --timescale
```

On Linux or macOS:

```bash
go build -o qnetward ./cmd/qnetward
./qnetward serve --listen 127.0.0.1:8080 --db-dsn "postgres://qnetward:change-me@127.0.0.1:5432/qnetward?sslmode=disable" --timescale
```

### 2. Start the dashboard

In another terminal:

```powershell
cd apps/web
Copy-Item .env.example .env.local
pnpm install
pnpm dev
```

Open **http://127.0.0.1:3000**. The backend health endpoint is at **http://127.0.0.1:8080/healthz**.

> For ICMP probes on Linux, grant the binary raw socket access with `sudo setcap cap_net_raw+ep ./qnetward` or run it with the required privileges.

## 🖥️ CLI

Use the same probe engines directly from the command line:

```bash
./qnetward ping 8.8.8.8
./qnetward tcpping example.com --port 443
./qnetward portscan --ports 22,80,443 example.com
```

## 🧭 How it fits together

```mermaid
flowchart LR
    U[Browser or CLI] --> W[Next.js dashboard]
    W -->|HTTP / WebSocket| A[Go API]
    A --> E[Probe engines]
    A --> DB[(PostgreSQL / TimescaleDB)]
    A --> R[(Redis cache)]
    A --> S[(S3-compatible storage)]
```

| Area | Responsibility |
| --- | --- |
| `apps/web/` | Next.js dashboard and API proxy routes |
| `cmd/qnetward/` | CLI entry point, server startup, dependency wiring |
| `internal/api/` | HTTP and WebSocket endpoints |
| `internal/engine/` | Network probe engines, independent of HTTP and storage |
| `internal/probe/` | Shared probe validation, limits, and report types |
| `internal/store/` | PostgreSQL/TimescaleDB persistence and migrations |
| `internal/platform/` | Redis cache, S3-compatible object storage, email adapter |
| `deploy/` | Production Docker Compose and proxy configuration |

## 🌐 API at a glance

| Method | Endpoint | Purpose |
| --- | --- | --- |
| `POST` | `/api/ping` | ICMP probe |
| `POST` | `/api/tcpping` | TCP connectivity and latency probe |
| `POST` | `/api/portscan` | TCP port scan |
| `GET` | `/api/history` | Query saved probe results |
| `POST` | `/api/export/history` | Download or upload a history export |
| `GET` | `/healthz` | Service and dependency health |
| `GET` | `/ws` | Realtime probe stream |

Example TCP ping request:

```bash
curl -X POST http://127.0.0.1:8080/api/tcpping \
  -H "Content-Type: application/json" \
  -d '{"host":"example.com","port":443,"count":4,"interval_ms":1000,"timeout_ms":2000}'
```

## 🐳 Production deployment

The production Compose stack includes the web app, Go API, PostgreSQL/TimescaleDB, Redis, and S3-compatible object storage.

```bash
cp .env.example .env
# Set secure credentials and deployment-specific values in .env
docker compose -f deploy/docker-compose.production.yml --env-file .env up --build
```

Common settings:

| Variable | Purpose |
| --- | --- |
| `QNETWARD_DB_DSN` | PostgreSQL connection string |
| `QNETWARD_TIMESCALE` | Enable TimescaleDB integration |
| `QNETWARD_REDIS_ADDR` | Redis address |
| `QNETWARD_S3_ENDPOINT` | S3-compatible endpoint |
| `QNETWARD_S3_ACCESS_KEY` / `QNETWARD_S3_SECRET_KEY` | Object storage credentials |
| `QNETWARD_S3_BUCKET` | Bucket for exported artifacts |
| `QNETWARD_ADMIN_EMAILS` | Platform administrator accounts |
| `QNETWARD_ALLOW_PRIVATE_TARGETS` | Allow private network probes for self-hosted LAN monitoring |

Keep secrets out of version control. The example values are for local setup only.

## 🛡️ Probe safety

The API blocks loopback, private, link-local, multicast, carrier-grade NAT, reserved, and cloud metadata targets by default. TCP probes validate resolved addresses at connection time to reduce DNS rebinding risks. For trusted, self-hosted LAN monitoring, set `QNETWARD_ALLOW_PRIVATE_TARGETS=true`; cloud metadata endpoints remain blocked. The CLI runs locally and does not apply the API target restriction.

If you expose Qnetward publicly, put a trusted reverse proxy or access control layer in front of it.

## 🧰 Development

```bash
go build -o qnetward ./cmd/qnetward
go test ./internal/...
go vet ./...
```

Frontend checks, from `apps/web/`:

```bash
pnpm lint
pnpm build
pnpm test:e2e
```

For the HTTP smoke test, start the local stack first, then run:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/smoke-test.ps1 -BaseUrl http://127.0.0.1:8080
```

Architecture decisions live in [`docs/adr/`](docs/adr/README.md). The frontend setup details are in [`apps/web/README.md`](apps/web/README.md).

## 🗺️ Project status

**Available:** ICMP and TCP ping, port scans, DNS inspection, realtime streaming, history and export, alert rules, workspaces, team invitations, and admin console.

**Planned:** Multi-target monitoring, HTTP/HTTPS checks, and expanded DNS tooling such as DNSSEC and propagation checks.

## Contributors

- MOPARA PAIR AYAT
- MONABBER EAMAN

<div align="center">

<sub>Built for operators who want a clearer view of their network.</sub>

</div>
