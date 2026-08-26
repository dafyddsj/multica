#!/usr/bin/env bash
# Create a local .env from .env.example with generated secrets if one is
# missing. Idempotent: an existing .env (with real secrets) is left untouched.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

if [ -f .env ]; then
  echo "==> [env] .env already present"
  exit 0
fi

echo "==> [env] Creating .env from .env.example"
cp .env.example .env
JWT="$(openssl rand -hex 32)"
VCSKEY="$(openssl rand -base64 32)"
sed -i "s/^JWT_SECRET=.*/JWT_SECRET=${JWT}/" .env
sed -i "s#^MULTICA_VCS_SECRET_KEY=.*#MULTICA_VCS_SECRET_KEY=${VCSKEY}#" .env
echo "==> [env] Generated JWT_SECRET and MULTICA_VCS_SECRET_KEY"
