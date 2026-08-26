#!/usr/bin/env bash
# Per-boot runtime initialization for Cursor Cloud Agents. Brings up the
# PostgreSQL cluster, ensures the .env file exists, and applies any pending
# migrations. Kept idempotent so it is safe to run on every boot.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

PG_MAJOR="17"
export PATH="$PATH:/usr/local/go/bin:/usr/lib/postgresql/${PG_MAJOR}/bin"

echo "==> [start] Starting PostgreSQL cluster"
sudo pg_ctlcluster "${PG_MAJOR}" main start 2>/dev/null || true
until pg_isready -q; do sleep 1; done
echo "==> [start] PostgreSQL is accepting connections"

echo "==> [start] Ensuring .env"
bash .cursor/ensure-env.sh

echo "==> [start] Applying database migrations"
set -a; . ./.env; set +a
(cd server && go run ./cmd/migrate up)

echo "==> [start] Ready"
