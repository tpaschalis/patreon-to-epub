#!/usr/bin/env bash
set -euo pipefail

if [ -z "${PATREON_SESSION_ID:-}" ]; then
  echo "Usage: PATREON_SESSION_ID=<your_session_id> ./run.sh [--output <dir>]"
  echo
  echo "To get your session_id:"
  echo "  1. Log in to patreon.com"
  echo "  2. Open DevTools → Application → Cookies → patreon.com"
  echo "  3. Copy the value of the 'session_id' cookie"
  exit 1
fi

cd "$(dirname "$0")"
go run ./cmd/patreon-to-epub "$@"
