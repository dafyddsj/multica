# Devin contract evidence

Reference snapshot from this planning run. Re-run the commands if the implementer has a newer binary. Do not invent flags.

## Binary

- Product name. Devin
- Vendor. Cognition
- Captured version. `devin 3000.6.2 (ce8ebcc1)`
- Manifest. `https://static.devin.ai/cli/current/manifest.json`
- Linux asset. `https://static.devin.ai/cli/3000.6.2/devin-3000.6.2-x86_64-unknown-linux.tar.gz`
- Install docs. [docs.devin.ai/cli](https://docs.devin.ai/cli)
- Help files. [help/](help/)

`devin auth status` without login printed `Not logged in` and `Credentials path: ~/.local/share/devin/credentials.toml`.

## Execute

`devin --print` / `-p` is the non-interactive path. Confirmed by `devin --help` 3000.6.2.

| Need | Flag or env | Notes |
|---|---|---|
| Print and exit | `-p` / `--print [PROMPT]` | Optional inline prompt. Do not use the inline form for Multica text. |
| Prompt from file | `--prompt-file <FILE>` | Verified. Use this. 0600 temp file, then delete. |
| Prompt after `--` | `devin -- <PROMPT>` | Starts an interactive session. Do not use it. |
| Stdin prompt | not documented | Do not assume it. Canary before switching off `--prompt-file`. |
| Permission | `--permission-mode` / `DEVIN_PERMISSION_MODE` | Values include `auto` (default), `accept-edits`, `smart`, `dangerous` (aliases `yolo`, `bypass`). Docs also name `autonomous` with `--sandbox`. |
| Trust | `--respect-workspace-trust false` | Required for `--print` in an untrusted directory. |
| Model | `--model` / `DEVIN_MODEL` | Real. Examples in help. `opus`, `claude-sonnet-4`. |
| Sandbox | `--sandbox` / `DEVIN_SANDBOX` | Research preview. Linux needs `bwrap` and `socat`. Out of v1. |
| Export | `--export [PATH]` | ATIF after each turn. Optional recovery path for session id. |
| Config file | `--config <PATH>` | Overrides `~/.config/devin/config.json`. Possible isolated MCP later. |

## Session and resume

| Need | Flag | Notes |
|---|---|---|
| Resume latest in cwd | `-c` / `--continue` | Shared-daemon race. Never emit. |
| Resume by id | `-r` / `--resume <SESSION_ID>` | Required for Multica follow-up. |
| Resume picker | `-r` / `--resume` with no id | Interactive. Never emit. |
| List | `devin list --format json` | Machine-readable sessions. Use this to learn the real id shape. |
| Delete | `devin rm` | Out of scope. |

Official examples. `brisk-otter` from essential-commands. `abc12345` from commands reference. Cloud DRS uses `devin-abc123…` for box ids. Treat cloud box ids as a different token until a canary proves otherwise.

No archive-on-execute flag in `--help`. Canary a second `--print --resume <id>` after the first print.

## Skills

`devin skills paths` on 3000.6.2 printed:

```
User skills (global):
  ~/.config/devin/skills/<skill-name>/SKILL.md
  ~/.config/cognition/skills/<skill-name>/SKILL.md
  ~/.agents/skills/<skill-name>/SKILL.md

Project skills:
  .devin/skills/<skill-name>/SKILL.md
  .cognition/skills/<skill-name>/SKILL.md
  .agents/skills/<skill-name>/SKILL.md
```

Write Multica skills to `{workDir}/.devin/skills/<name>/SKILL.md`. That is the Devin-native project path from the binary. `.agents/skills/` is also scanned and is shared with Goose and Amp. Do not write Multica-managed Devin skills there. Official docs also mention `.windsurf/skills/`. The binary paths command did not list it. Prefer the binary.

Do not write into the operator's home directories.

## MCP

`devin mcp add|list|get|remove|login|logout|enable|disable` manages durable MCP servers at local, project, or user scope. That is not a per-task isolated config.

`--config` can point at a temp file. Canary whether a Multica-owned config with an MCP server is honored for one `--print` run and discarded after. Hold the MCP UI tab until that canary.

## Auth

- `devin auth login`
- `devin auth logout`
- `devin auth status`
- `--force-manual-token-flow` for SSH
- Enterprise SSO and RBAC through the Devin dashboard
- Cloud API uses `cog_` service-user keys. That is not the CLI login.

Multica is not the Devin IdP.

## ACP and cloud, out of v1

- `devin acp` is an ACP server on stdio for Windsurf and Zed.
- `devin cloud drs` manages blueprints, sandbox sessions, and builds.
- `devin ssh` opens a cloud box.
- Organization API v3 creates cloud sessions at `https://api.devin.ai/v3/organizations/{org}/sessions`.

None of these is the daemon-host execute path.

## Official docs used

- [Commands and flags](https://docs.devin.ai/cli/reference/commands)
- [Essential commands](https://docs.devin.ai/cli/essential-commands)
- [Permissions](https://docs.devin.ai/cli/reference/permissions)
- [Skills](https://docs.devin.ai/cli/extensibility/skills/overview)
- [Auth](https://docs.devin.ai/cli/enterprise/devin-auth)
- [API overview](https://docs.devin.ai/api-reference/overview)
- Captured local `--help` in [help/](help/)
