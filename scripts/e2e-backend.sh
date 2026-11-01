#!/usr/bin/env bash
# Starts an API server set up for the Playwright suite (apps/web/tests/e2e):
#   - uses QNETWARD_TEST_POSTGRES_DSN, so e2e accounts never land in the dev database
#   - email delivery off, so signups return a preview code the specs can verify with
#   - demo login on, and e2e-admin@example.com as the platform admin the admin spec uses
#   - private targets allowed, because the ping specs probe 127.0.0.1
#
# Typical local run (three terminals, or background the first two):
#   go build -o .tmp/qnetward ./cmd/qnetward && scripts/e2e-backend.sh
#   NEXT_PUBLIC_QNETWARD_WS_URL=ws://127.0.0.1:8090/ws pnpm --dir apps/web build \
#     && QNETWARD_BACKEND_URL=http://127.0.0.1:8090 pnpm --dir apps/web start --port 3002
#   PLAYWRIGHT_BASE_URL=http://127.0.0.1:3002 pnpm --dir apps/web test:e2e
set -euo pipefail
cd "$(dirname "$0")/.."

env_file="${ENV_FILE:-.env}"
if [[ -f "$env_file" ]]; then
  while IFS= read -r line || [[ -n "$line" ]]; do
    line="${line%$'\r'}"
    [[ -z "$line" || "$line" == \#* || "$line" != *=* ]] && continue
    export "${line%%=*}=${line#*=}"
  done < "$env_file"
fi

: "${QNETWARD_TEST_POSTGRES_DSN:?set QNETWARD_TEST_POSTGRES_DSN (env or .env) to a disposable database}"
port="${E2E_API_PORT:-8090}"
web="${E2E_WEB_ORIGIN:-http://127.0.0.1:3002}"

export QNETWARD_DB_DSN="$QNETWARD_TEST_POSTGRES_DSN"
export QNETWARD_RESEND_API_KEY=""
export QNETWARD_DEMO_ENABLED=true
export QNETWARD_DEMO_NAME="Contributor Demo"
export QNETWARD_DEMO_EMAIL="demo@qnetward.local"
export QNETWARD_ADMIN_EMAILS="e2e-admin@example.com"
export QNETWARD_ALLOW_PRIVATE_TARGETS=true
export QNETWARD_PUBLIC_BASE_URL="$web"
export QNETWARD_ALLOWED_ORIGINS="$web"

bin=".tmp/qnetward"
[[ -x "$bin" || ! -x "$bin.exe" ]] || bin="$bin.exe"
exec "$bin" serve -listen "127.0.0.1:$port"
