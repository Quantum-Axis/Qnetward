#!/usr/bin/env bash
# Runs a command with the variables from .env exported.
# Values are taken literally (no shell expansion), so DSNs with & or ? are safe.
# Usage: scripts/with-env.sh <command> [args...]
set -euo pipefail
env_file="${ENV_FILE:-.env}"
if [[ -f "$env_file" ]]; then
  while IFS= read -r line || [[ -n "$line" ]]; do
    line="${line%$'\r'}"
    [[ -z "$line" || "$line" == \#* || "$line" != *=* ]] && continue
    export "${line%%=*}=${line#*=}"
  done < "$env_file"
fi
exec "$@"
