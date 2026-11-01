<div align="center">

# Q N E T W A R D

### **Know your network.**

A self-hosted command center for live connectivity checks, DNS inspection, and network history.

<br />

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go&logoColor=white)
![Next.js](https://img.shields.io/badge/Next.js-16-000000?style=flat-square&logo=nextdotjs&logoColor=white)
![TypeScript](https://img.shields.io/badge/TypeScript-5-3178C6?style=flat-square&logo=typescript&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-TimescaleDB-4169E1?style=flat-square&logo=postgresql&logoColor=white)
![Redis](https://img.shields.io/badge/Cache-Redis-DC382D?style=flat-square&logo=redis&logoColor=white)

**[Quick start](#-quick-start) · [Capabilities](#-capabilities) · [Architecture](#-architecture) · [Deploy](#-deploy-with-docker)**

<br />

![Qnetward real-time network monitoring dashboard](assets/qnetward-readme-hero.jpg)

</div>

---

> **See what your network is doing.** Run a probe, follow every result as it arrives, and keep the data in infrastructure you own.

## ◈ Capabilities

| **Probe** | **Understand** | **Operate** |
|:--|:--|:--|
| ICMP and TCP ping | Live WebSocket streaming | Workspaces and team invitations |
| TCP port scans | Latency charts and result tables | Alert rules with email delivery |
| DNS lookup and inspection | Searchable probe history | Platform admin console |
| JSON and CSV exports | Redis-backed history cache | Self-hosted production stack |

## ⚡ Quick start

### Prerequisites

- Go 1.25+
- Node.js and pnpm
- PostgreSQL (TimescaleDB supported)

### 1 · Start the backend

From the repository root, build and run the Go service. Replace the sample database credentials with values for your local PostgreSQL instance.

**Windows (PowerShell)**

```powershell
go build -o qnetward.exe .\cmd\qnetward
.\qnetward.exe serve --listen 127.0.0.1:8080 --db-dsn "postgres://qnetward:change-me@127.0.0.1:5432/qnetward?sslmode=disable" --timescale
```

**Linux / macOS**

```bash
go build -o qnetward ./cmd/qnetward
./qnetward serve --listen 127.0.0.1:8080 --db-dsn "postgres://qnetward:change-me@127.0.0.1:5432/qnetward?sslmode=disable" --timescale
```

### 2 · Launch the dashboard

In a second terminal:

```powershell
cd apps/web
Copy-Item .env.example .env.local
pnpm install
pnpm dev
```

| Service | Address |
|:--|:--|
| Dashboard | [http://127.0.0.1:3000](http://127.0.0.1:3000) |
| Backend health | [http://127.0.0.1:8080/healthz](http://127.0.0.1:8080/healthz) |

> **Linux ICMP:** The binary may need raw socket access. Grant it with `sudo setcap cap_net_raw+ep ./qnetward`, or run it with the required privileges.

## ⌘ CLI

Run the probe engines directly, without opening the dashboard:

```bash
./qnetward ping 8.8.8.8
./qnetward tcpping example.com --port 443
./qnetward portscan --ports 22,80,443 example.com
```

## ◉ Architecture

```mermaid
flowchart LR
    C[CLI] --> E[Probe engines]
    B[Browser] --> W[Next.js dashboard]
    W -->|HTTP and WebSocket| A[Go API]
    A --> E
    A --> D[(PostgreSQL / TimescaleDB)]
    A --> R[(Redis)]
    A --> S[(S3-compatible storage)]
```

| Component | Role |
|:--|:--|
| `apps/web/` | Next.js dashboard and API proxy routes |
| `cmd/qnetward/` | CLI, server entry point, dependency wiring |
| `internal/api/` | HTTP and WebSocket API |
| `internal/engine/` | Network probe engines, independent of HTTP and storage |
| `internal/probe/` | Shared validation, limits, and report types |
| `internal/store/` | PostgreSQL/TimescaleDB persistence and migrations |
| `internal/platform/` | Redis cache, S3-compatible object storage, email adapter |
| `deploy/` | Production Docker Compose and proxy configuration |

## ⇄ API

| Method | Route | Description |
|:--|:--|:--|
| `POST` | `/api/ping` | ICMP probe |
| `POST` | `/api/tcpping` | TCP connectivity and latency probe |
| `POST` | `/api/portscan` | TCP port scan |
| `GET` | `/api/history` | Query saved probe results |
| `POST` | `/api/export/history` | Download or upload history export |
| `GET` | `/healthz` | Service and dependency health |
| `GET` | `/ws` | Realtime probe stream |

Example request:

```bash
curl -X POST http://127.0.0.1:8080/api/tcpping \
  -H "Content-Type: application/json" \
  -d '{"host":"example.com","port":443,"count":4,"interval_ms":1000,"timeout_ms":2000}'
```

## ⬡ Deploy with Docker

The production Compose stack brings up the dashboard, Go API, PostgreSQL/TimescaleDB, Redis, and S3-compatible storage.

```bash
cp .env.example .env
# Update .env with secure credentials and deployment-specific values
docker compose -f deploy/docker-compose.production.yml --env-file .env up --build
```

| Variable | Controls |
|:--|:--|
| `QNETWARD_DB_DSN` | PostgreSQL connection |
| `QNETWARD_TIMESCALE` | TimescaleDB integration |
| `QNETWARD_REDIS_ADDR` | Redis connection |
| `QNETWARD_S3_ENDPOINT` | S3-compatible endpoint |
| `QNETWARD_S3_ACCESS_KEY` / `QNETWARD_S3_SECRET_KEY` | Object storage credentials |
| `QNETWARD_S3_BUCKET` | Export artifact bucket |
| `QNETWARD_ADMIN_EMAILS` | Platform admin accounts |
| `QNETWARD_ALLOW_PRIVATE_TARGETS` | Private network probes for self-hosted LAN monitoring |

Use strong credentials and keep `.env` out of version control. Example credentials are for local setup only.

## ⛨ Probe safety

The API blocks loopback, private, link-local, multicast, carrier-grade NAT, reserved, and cloud metadata addresses by default. It checks resolved addresses at connection time to reduce DNS rebinding risks. For trusted LAN monitoring, set `QNETWARD_ALLOW_PRIVATE_TARGETS=true`; cloud metadata endpoints remain blocked. The local CLI does not apply the API target restriction.

If exposing Qnetward publicly, place a trusted reverse proxy or access control layer in front of it.

## 🛠 Development

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

More detail: [architecture decisions](docs/adr/README.md) · [frontend development guide](apps/web/README.md)

## ◌ Roadmap

**Available** · ICMP and TCP ping · Port scans · DNS inspection · Realtime streaming · History and exports · Alerts · Workspaces · Team invitations · Admin console

**Planned** · Multi-target monitoring · HTTP/HTTPS checks · Expanded DNS tooling (DNSSEC and propagation)

## Contributors

MOPARA PAIR AYAT · MONABBER EAMAN

<div align="center">

<sub>Self-hosted network diagnostics, built for clarity.</sub>

</div>
