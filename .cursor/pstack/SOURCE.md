# Vendored pstack

Cloud Agent VMs do not reliably receive the Cursor Marketplace `pstack`
plugin. The plugin cache is often empty on first boot, and when the plugin
does land, injected skill paths can point at `/home/cursor/...` while the
runtime user is `ubuntu`.

This tree is the project-local copy Cloud Agents actually load:

- Skills: `.cursor/skills/pstack/` (discovered as project skills)
- Agents: `.cursor/agents/` (`poteto-agent`, `comment-sicko`)
- Model seats: `.cursor/rules/pstack-models.mdc` (already always-applied)

Pinned from the public mirror of `cursor/plugins/pstack`:

- Upstream: https://github.com/backnotprop/pstack
- Official: https://github.com/cursor/plugins/tree/main/pstack
- Version: 0.14.1
- Commit: 18e0e908a13553b0e58d065ab26dbc9a972ec8ba
- License: MIT (see `LICENSE`)

Refresh with `.cursor/sync-pstack.sh`.
