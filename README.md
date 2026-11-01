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

## ◉ Our Team

<table width="100%">
<tr>
<td align="center" width="25%">
  <a href="https://github.com/Fatemabintemariam">
    <img src="https://avatars.githubusercontent.com/u/270080358?v=4" width="180" height="180" alt="Fatema Binte Mariam's GitHub avatar" />
  </a><br />
  <strong><a href="https://github.com/Fatemabintemariam">Fatema Binte Mariam</a></strong><br />
  Qnetward contributor
</td>
<td align="center" width="25%">
  <a href="https://github.com/Moparapairayat">
    <img src="https://avatars.githubusercontent.com/u/83800319?v=4" width="180" height="180" alt="Mopara Pair Ayat's GitHub avatar" />
  </a><br />
  <strong><a href="https://github.com/Moparapairayat">Mopara Pair Ayat</a></strong><br />
  Organization member
</td>
<td align="center" width="25%">
  <a href="https://github.com/halitsever">
    <img src="https://avatars.githubusercontent.com/u/34791459?v=4" width="180" height="180" alt="Halit Sever's GitHub avatar" />
  </a><br />
  <strong><a href="https://github.com/halitsever">Halit Sever</a></strong><br />
  Qnetward contributor
</td>
<td align="center" width="25%">
  <a href="https://github.com/monabber444">
    <img src="https://avatars.githubusercontent.com/u/185014085?v=4" width="180" height="180" alt="Monabber Eaman's GitHub avatar" />
  </a><br />
  <strong><a href="https://github.com/monabber444">Monabber Eaman</a></strong><br />
  Qnetward contributor
</td>
</tr>
<tr>
<td align="center" width="25%">
  <a href="https://github.com/OrhanAsantr">
    <img src="https://avatars.githubusercontent.com/u/247149651?v=4" width="180" height="180" alt="Orhan Asan's GitHub avatar" />
  </a><br />
  <strong><a href="https://github.com/OrhanAsantr">Orhan Asan</a></strong><br />
  Qnetward contributor
</td>
<td align="center" width="25%">
  <a href="https://github.com/NurhayatYurtaslan">
    <img src="https://avatars.githubusercontent.com/u/80510115?v=4" width="180" height="180" alt="Nurhayat Yurtaslan's GitHub avatar" />
  </a><br />
  <strong><a href="https://github.com/NurhayatYurtaslan">Nurhayat Yurtaslan</a></strong><br />
  Qnetward contributor
</td>
<td align="center" width="25%">
  <a href="https://github.com/needim">
    <img src="https://avatars.githubusercontent.com/u/271912?v=4" width="180" height="180" alt="Nedim Arabacı's GitHub avatar" />
  </a><br />
  <strong><a href="https://github.com/needim">Nedim Arabacı</a></strong><br />
  Qnetward contributor
</td>
<td align="center" width="25%">
  <a href="https://github.com/CodeByPinar">
    <img src="https://avatars.githubusercontent.com/u/107423523?v=4" width="180" height="180" alt="Pınar's GitHub avatar" />
  </a><br />
  <strong><a href="https://github.com/CodeByPinar">Pınar</a></strong><br />
  Qnetward contributor
</td>
</tr>
<tr>
<td align="center" width="25%">
  <a href="https://github.com/edayavuz6">
    <img src="https://avatars.githubusercontent.com/u/249468949?v=4" width="180" height="180" alt="Eda Yavuz's GitHub avatar" />
  </a><br />
  <strong><a href="https://github.com/edayavuz6">Eda Yavuz</a></strong><br />
  Qnetward contributor
</td>
<td align="center" width="25%">
  <a href="https://github.com/zeynepecekiris">
    <img src="https://avatars.githubusercontent.com/u/174215159?v=4" width="180" height="180" alt="Zeynep Ece Kiris's GitHub avatar" />
  </a><br />
  <strong><a href="https://github.com/zeynepecekiris">Zeynep Ece Kiris</a></strong><br />
  Qnetward contributor
</td>
<td align="center" width="25%">
  <a href="https://github.com/BerattCelikk">
    <img src="https://avatars.githubusercontent.com/u/138959468?v=4" width="180" height="180" alt="Berat Erol ÇELİK's GitHub avatar" />
  </a><br />
  <strong><a href="https://github.com/BerattCelikk">Berat Erol ÇELİK</a></strong><br />
  Qnetward contributor
</td>
<td width="25%"></td>
</tr>
</table>

## ⌘ Technology Stack

### ⚡ Backend & Network Engine

<img src="https://skillicons.dev/icons?i=go&theme=dark" alt="Go" />

**Go** · **HTTP API** · **WebSockets** · **ICMP / TCP probes** · **DNS**

### 🎨 Frontend & Web Experience

<img src="https://skillicons.dev/icons?i=nextjs,react,ts,tailwind&theme=dark" alt="Next.js, React, TypeScript, and Tailwind CSS" />

**Next.js** · **React** · **TypeScript** · **Tailwind CSS**

### 🗄️ Data & Platform

<img src="https://skillicons.dev/icons?i=postgres,redis,docker&theme=dark" alt="PostgreSQL, Redis, and Docker" />

**PostgreSQL** · **TimescaleDB** · **Redis** · **Docker Compose** · **S3-compatible storage** · **MinIO**

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
```

End-to-end tests run against a separate API on the test database, with email
delivery off so signups return a verification code the specs can use:

```bash
make e2e-api                                    # API on :8090 (needs QNETWARD_TEST_POSTGRES_DSN)
NEXT_PUBLIC_QNETWARD_WS_URL=ws://127.0.0.1:8090/ws pnpm --dir apps/web build
QNETWARD_BACKEND_URL=http://127.0.0.1:8090 pnpm --dir apps/web start --port 3002
PLAYWRIGHT_BASE_URL=http://127.0.0.1:3002 pnpm --dir apps/web test:e2e
```

HTTP smoke test (with the local stack running):

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/smoke-test.ps1 -BaseUrl http://127.0.0.1:8080
```

**More detail:** [Architecture decisions](docs/adr/README.md) · [Frontend development guide](apps/web/README.md)

## Project status

**Available**　ICMP and TCP ping · Port scans · DNS inspection · Realtime streaming · History and exports · Alerts · Workspaces · Team invitations · Admin console

**Planned**　Multi-target monitoring · HTTP/HTTPS checks · Expanded DNS tooling (DNSSEC and propagation)



<div align="center">

---

<sub>SEE THE SIGNAL. OWN THE STACK.</sub>

</div>
