#!/usr/bin/env bash
# Refresh the vendored pstack copy from the public mirror.
# Usage: .cursor/sync-pstack.sh [ref]
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REF="${1:-main}"
SRC="$(mktemp -d "${TMPDIR:-/tmp}/pstack-sync.XXXXXX")"
cleanup() { rm -rf "$SRC"; }
trap cleanup EXIT

git clone --depth 1 --branch "$REF" https://github.com/backnotprop/pstack.git "$SRC"
COMMIT="$(git -C "$SRC" rev-parse HEAD)"
VERSION="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1])).get("version","unknown"))' "$SRC/.cursor-plugin/plugin.json")"

rm -rf "$REPO_ROOT/.cursor/skills/pstack"
mkdir -p "$REPO_ROOT/.cursor/skills/pstack" "$REPO_ROOT/.cursor/agents" "$REPO_ROOT/.cursor/pstack/.cursor-plugin"
cp -a "$SRC/skills/." "$REPO_ROOT/.cursor/skills/pstack/"
cp -a "$SRC/agents/." "$REPO_ROOT/.cursor/agents/"
cp "$SRC/.cursor-plugin/plugin.json" "$REPO_ROOT/.cursor/pstack/.cursor-plugin/plugin.json"
cp "$SRC/LICENSE" "$REPO_ROOT/.cursor/pstack/LICENSE"

cat > "$REPO_ROOT/.cursor/pstack/SOURCE.md" <<EOF
# Vendored pstack

Cloud Agent VMs do not reliably receive the Cursor Marketplace \`pstack\`
plugin. The plugin cache is often empty on first boot, and when the plugin
does land, injected skill paths can point at \`/home/cursor/...\` while the
runtime user is \`ubuntu\`.

This tree is the project-local copy Cloud Agents actually load:

- Skills: \`.cursor/skills/pstack/\` (discovered as project skills)
- Agents: \`.cursor/agents/\` (\`poteto-agent\`, \`comment-sicko\`)
- Model seats: \`.cursor/rules/pstack-models.mdc\` (already always-applied)

Pinned from the public mirror of \`cursor/plugins/pstack\`:

- Upstream: https://github.com/backnotprop/pstack
- Official: https://github.com/cursor/plugins/tree/main/pstack
- Version: ${VERSION}
- Commit: ${COMMIT}
- License: MIT (see \`LICENSE\`)

Refresh with \`.cursor/sync-pstack.sh\`.
EOF

echo "==> [pstack] Synced ${VERSION} at ${COMMIT}"
