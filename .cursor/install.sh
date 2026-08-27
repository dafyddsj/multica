#!/usr/bin/env bash
# Idempotent repository bootstrap for Cursor Cloud Agents.
# Installs the toolchain the default image lacks (Go 1.26, PostgreSQL 17 +
# pgvector), installs JS dependencies, prepares a local .env, and runs the
# database migrations so the baseline snapshot boots with a ready database.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

GO_VERSION="1.26.6"
PG_MAJOR="17"

echo "==> [install] Ensuring Go ${GO_VERSION}"
if [ "$(/usr/local/go/bin/go version 2>/dev/null | awk '{print $3}')" != "go${GO_VERSION}" ]; then
  arch="$(dpkg --print-architecture)"
  curl -fsSL -o /tmp/go.tar.gz "https://go.dev/dl/go${GO_VERSION}.linux-${arch}.tar.gz"
  sudo rm -rf /usr/local/go
  sudo tar -C /usr/local -xzf /tmp/go.tar.gz
  rm -f /tmp/go.tar.gz
fi

echo "==> [install] Ensuring PostgreSQL ${PG_MAJOR} + pgvector"
if [ ! -x "/usr/lib/postgresql/${PG_MAJOR}/bin/postgres" ]; then
  sudo DEBIAN_FRONTEND=noninteractive apt-get update -y
  sudo DEBIAN_FRONTEND=noninteractive apt-get install -y curl ca-certificates gnupg lsb-release
  sudo install -d /usr/share/postgresql-common/pgdg
  sudo curl -fsSL -o /usr/share/postgresql-common/pgdg/apt.postgresql.org.asc \
    https://www.postgresql.org/media/keys/ACCC4CF8.asc
  echo "deb [signed-by=/usr/share/postgresql-common/pgdg/apt.postgresql.org.asc] https://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" \
    | sudo tee /etc/apt/sources.list.d/pgdg.list >/dev/null
  sudo DEBIAN_FRONTEND=noninteractive apt-get update -y
  sudo DEBIAN_FRONTEND=noninteractive apt-get install -y \
    "postgresql-${PG_MAJOR}" "postgresql-${PG_MAJOR}-pgvector"
fi

echo "==> [install] Exposing Go and PostgreSQL binaries on PATH for future shells"
sudo tee /etc/profile.d/multica-dev.sh >/dev/null <<EOF
export PATH="\$PATH:/usr/local/go/bin:/usr/lib/postgresql/${PG_MAJOR}/bin"
EOF
export PATH="$PATH:/usr/local/go/bin:/usr/lib/postgresql/${PG_MAJOR}/bin"

echo "==> [install] Starting PostgreSQL cluster"
sudo pg_ctlcluster "${PG_MAJOR}" main start 2>/dev/null || true
until pg_isready -q; do sleep 1; done

echo "==> [install] Ensuring multica role and database"
sudo -u postgres psql -v ON_ERROR_STOP=1 <<'SQL'
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'multica') THEN
    CREATE ROLE multica WITH LOGIN PASSWORD 'multica' CREATEDB;
  END IF;
END$$;
SQL
if ! sudo -u postgres psql -tAc "SELECT 1 FROM pg_database WHERE datname='multica'" | grep -q 1; then
  sudo -u postgres createdb -O multica multica
fi
sudo -u postgres psql -d multica -c "ALTER DATABASE multica OWNER TO multica; GRANT ALL ON SCHEMA public TO multica;" >/dev/null

echo "==> [install] Preparing .env"
bash .cursor/ensure-env.sh

echo "==> [install] Ensuring pstack"
bash .cursor/ensure-pstack.sh

echo "==> [install] Installing JS dependencies"
corepack pnpm install --frozen-lockfile

echo "==> [install] Running database migrations"
set -a; . ./.env; set +a
(cd server && go run ./cmd/migrate up)

echo "==> [install] Done"
