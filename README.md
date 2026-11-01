<div align="center">

# Qnetward

### **Network clarity, in real time.**

Self-hosted diagnostics and monitoring for the systems you run.

<br />

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/)
[![Next.js](https://img.shields.io/badge/Next.js-16-000000?style=flat-square&logo=nextdotjs&logoColor=white)](https://nextjs.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-5-3178C6?style=flat-square&logo=typescript&logoColor=white)](https://www.typescriptlang.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-TimescaleDB-4169E1?style=flat-square&logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Cache-Redis-DC382D?style=flat-square&logo=redis&logoColor=white)](https://redis.io/)

<br />

**[Get running](#-get-running)　·　[What it does](#-built-for-network-visibility)　·　[How it works](#-under-the-hood)　·　[Deploy](#-production-deployment)**

<br />

<img src="assets/qnetward-readme-hero.jpg" alt="Qnetward network monitoring dashboard with live latency charts and target health" width="100%" />

<br />

`PROBE` &nbsp;→&nbsp; `STREAM` &nbsp;→&nbsp; `UNDERSTAND` &nbsp;→&nbsp; `ACT`

</div>

---

## Built for network visibility

| &nbsp; | Capability | What you get |
|:--:|:--|:--|
| ◉ | **Connectivity probes** | ICMP ping, TCP ping, port scanning, and DNS inspection |
| ∿ | **Live observability** | Streaming results, latency charts, and target health at a glance |
| ◷ | **Durable history** | Search prior runs, inspect trends, and export JSON or CSV |
| ⌁ | **Team operations** | Workspaces, invitations, alert rules, and email delivery |
| ⬡ | **Your infrastructure** | Self-host the dashboard, API, database, and supporting services |

Qnetward combines a Go probe engine with a Next.js dashboard. Start with a single local instance, then add persistent storage, caching, and object storage as your deployment grows.

## Get running

### You’ll need

- Go 1.25+
- Node.js and pnpm
- PostgreSQL (TimescaleDB supported)

### 01 — Start the API

Run from the repository root. Replace the sample database credentials with your local PostgreSQL values.

<details open>
<summary><strong>Windows · PowerShell</strong></summary>

```powershell
go build -o qnetward.exe .\cmd\qnetward
.\qnetward.exe serve --listen 127.0.0.1:8080 --db-dsn "postgres://qnetward:change-me@127.0.0.1:5432/qnetward?sslmode=disable" --timescale
```

</details>

<details>
<summary><strong>Linux · macOS</strong></summary>

```bash
go build -o qnetward ./cmd/qnetward
./qnetward serve --listen 127.0.0.1:8080 --db-dsn "postgres://qnetward:change-me@127.0.0.1:5432/qnetward?sslmode=disable" --timescale
```

</details>

### 02 — Launch the dashboard

In a second terminal:

```powershell
cd apps/web
Copy-Item .env.example .env.local
pnpm install
pnpm dev
```

| Open | URL |
|:--|:--|
| **Dashboard** | [http://127.0.0.1:3000](http://127.0.0.1:3000) |
| **API health** | [http://127.0.0.1:8080/healthz](http://127.0.0.1:8080/healthz) |

> **Linux ICMP:** The binary may need raw socket permissions. Grant access with `sudo setcap cap_net_raw+ep ./qnetward`, or run with the required privileges.

## One binary. Three quick checks.

```bash
./qnetward ping 8.8.8.8
./qnetward tcpping example.com --port 443
./qnetward portscan --ports 22,80,443 example.com
```

## Under the hood

```mermaid
flowchart LR
    CLI[CLI] --> E[Probe engines]
    Browser[Browser] --> Web[Next.js dashboard]
    Web -->|HTTP / WebSocket| API[Go API]
    API --> E
    API --> DB[(PostgreSQL / TimescaleDB)]
    API --> Redis[(Redis cache)]
    API --> S3[(S3-compatible storage)]
```

| Path | What lives there |
|:--|:--|
| `apps/web/` | Next.js dashboard and API proxy routes |
| `cmd/qnetward/` | CLI entry point, server startup, dependency wiring |
| `internal/api/` | HTTP and WebSocket handlers |
| `internal/engine/` | Probe engines, independent of HTTP and storage |
| `internal/probe/` | Shared validation, limits, and report types |
| `internal/store/` | PostgreSQL/TimescaleDB persistence and migrations |
| `internal/platform/` | Redis, S3-compatible storage, and email adapters |
| `deploy/` | Production Docker Compose and proxy configuration |

## API surface

| Method | Route | Use |
|:--|:--|:--|
| `POST` | `/api/ping` | ICMP probe |
| `POST` | `/api/tcpping` | TCP connectivity and latency |
| `POST` | `/api/portscan` | TCP port scan |
| `GET` | `/api/history` | Query saved results |
| `POST` | `/api/export/history` | Download or upload history export |
| `GET` | `/healthz` | Service and dependency status |
| `GET` | `/ws` | Realtime probe stream |

Try a TCP ping:

```bash
curl -X POST http://127.0.0.1:8080/api/tcpping \
  -H "Content-Type: application/json" \
  -d '{"host":"example.com","port":443,"count":4,"interval_ms":1000,"timeout_ms":2000}'
```

## Production deployment

The production Compose stack includes the dashboard, Go API, PostgreSQL/TimescaleDB, Redis, and S3-compatible object storage.

```bash
cp .env.example .env
# Set secure credentials and deployment-specific values in .env
docker compose -f deploy/docker-compose.production.yml --env-file .env up --build
```

| Setting | Purpose |
|:--|:--|
| `QNETWARD_DB_DSN` | PostgreSQL connection string |
| `QNETWARD_TIMESCALE` | Enable TimescaleDB integration |
| `QNETWARD_REDIS_ADDR` | Redis address |
| `QNETWARD_S3_ENDPOINT` | S3-compatible endpoint |
| `QNETWARD_S3_ACCESS_KEY` / `QNETWARD_S3_SECRET_KEY` | Object storage credentials |
| `QNETWARD_S3_BUCKET` | Export artifact bucket |
| `QNETWARD_ADMIN_EMAILS` | Platform administrator accounts |
| `QNETWARD_ALLOW_PRIVATE_TARGETS` | Permit private targets for trusted LAN monitoring |

Keep secrets out of version control; example credentials are for local setup only.

## Safe by default

The API blocks loopback, private, link-local, multicast, carrier-grade NAT, reserved, and cloud metadata addresses by default. TCP probes validate resolved addresses at connection time to reduce DNS rebinding risks. Trusted self-hosted LAN monitoring can enable `QNETWARD_ALLOW_PRIVATE_TARGETS=true`; cloud metadata endpoints remain blocked. The local CLI does not apply the API target restriction.

If exposing Qnetward publicly, place a trusted reverse proxy or access control layer in front of it.

## Development

```bash
go build -o qnetward ./cmd/qnetward
go test ./internal/...
go vet ./...
```

From `apps/web/`:

```bash
pnpm lint
pnpm build
pnpm test:e2e
```

HTTP smoke test (with the local stack running):

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/smoke-test.ps1 -BaseUrl http://127.0.0.1:8080
```

**More detail:** [Architecture decisions](docs/adr/README.md) · [Frontend development guide](apps/web/README.md)

## Project status

**Available**　ICMP and TCP ping · Port scans · DNS inspection · Realtime streaming · History and exports · Alerts · Workspaces · Team invitations · Admin console

**Planned**　Multi-target monitoring · HTTP/HTTPS checks · Expanded DNS tooling (DNSSEC and propagation)

## Contributors

MOPARA PAIR AYAT · MONABBER EAMAN

<div align="center">

---

<sub>SEE THE SIGNAL. OWN THE STACK.</sub>

</div>
