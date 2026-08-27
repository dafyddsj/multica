#!/usr/bin/env bash
# Make pstack usable on Cursor Cloud Agent VMs when marketplace plugin
# sync is empty or injects /home/cursor paths. Idempotent. Safe locally.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SKILLS_SRC="$REPO_ROOT/.cursor/skills/pstack"
AGENTS_SRC="$REPO_ROOT/.cursor/agents"
PLUGIN_JSON="$REPO_ROOT/.cursor/pstack/.cursor-plugin/plugin.json"

if [ ! -f "$SKILLS_SRC/poteto-mode/SKILL.md" ]; then
  echo "==> [pstack] Missing $SKILLS_SRC/poteto-mode/SKILL.md" >&2
  exit 1
fi

# Cloud plugin skill fullPaths are sometimes /home/cursor/... while the
# runtime user is ubuntu and the files live under $HOME. A home symlink
# makes both the injected path and the real path resolve.
if [ ! -e /home/cursor ] && [ "$(id -un 2>/dev/null || true)" = "ubuntu" ] && [ -d "$HOME" ]; then
  if command -v sudo >/dev/null 2>&1 && sudo -n true 2>/dev/null; then
    sudo ln -sfn "$HOME" /home/cursor
    echo "==> [pstack] Linked /home/cursor -> $HOME"
  else
    echo "==> [pstack] /home/cursor missing; no passwordless sudo to create it"
  fi
elif [ -e /home/cursor ]; then
  echo "==> [pstack] /home/cursor already exists"
fi

# Local plugin install so poteto-agent is discoverable when the Cloud
# marketplace cache is empty (this VM's .cloud-plugin-manifest.json is []).
LOCAL_PLUGIN="${HOME}/.cursor/plugins/local/pstack"
if [ -f "$PLUGIN_JSON" ] && [ -d "$AGENTS_SRC" ]; then
  mkdir -p "$LOCAL_PLUGIN/.cursor-plugin"
  ln -sfn "$PLUGIN_JSON" "$LOCAL_PLUGIN/.cursor-plugin/plugin.json"
  ln -sfn "$SKILLS_SRC" "$LOCAL_PLUGIN/skills"
  ln -sfn "$AGENTS_SRC" "$LOCAL_PLUGIN/agents"
  echo "==> [pstack] Installed local plugin at $LOCAL_PLUGIN"
fi

echo "==> [pstack] Ready ($(find "$SKILLS_SRC" -name SKILL.md | wc -l | tr -d ' ') skills)"
