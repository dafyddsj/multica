# Goose contract evidence

Reference snapshot from this planning run. Re-run the commands if the implementer has a newer binary. Do not invent flags.

## Binary

- Product name. Goose
- Source. [block/goose](https://github.com/block/goose) redirects to [aaif-goose/goose](https://github.com/aaif-goose/goose)
- License. Apache-2.0, download allowed
- Captured version. `1.48.0`
- Asset. `goose-x86_64-unknown-linux-gnu.tar.gz` from the `v1.48.0` GitHub release
- Help files. [help/](help/)

```
$ goose --version
 1.48.0
```

## Execute

`goose run` is the non-interactive path. Confirmed by `goose run --help`.

| Need | Flag or env | Notes |
|---|---|---|
| Prompt on stdin | `-i -` / `--instructions -` | Documented. Use this. Close stdin after the prompt. |
| Prompt on argv | `-t` / `--text` | Exists. Do not use it for Multica prompts. |
| Prompt from file | `-i FILE` | Exists. Stdin is enough. |
| Stream events | `--output-format stream-json` | Values are `text`, `json`, `stream-json`. Default is `text`. |
| Quiet text | `-q` / `--quiet` | Suppresses non-response output. Leave off until a capture proves stream-json still arrives. |
| System extra | `--system` | Exists. Do not use it for the Multica brief until the AGENTS.md canary fails. |
| Provider | `--provider` / `GOOSE_PROVIDER` | Real. |
| Model | `--model` / `GOOSE_MODEL` | Real. |
| Unattended mode | `GOOSE_MODE=auto` | Documented env. `goose run` is already non-interactive. Canary whether the env is required. |

Do not assume a `-p` flag. Goose does not have one.

## Session and resume

| Need | Flag | Notes |
|---|---|---|
| Name a session | `-n` / `--name` | Optional. Not an id. |
| Resume a named session | `--resume --name <name>` | Name is not the stored Multica session id. |
| Resume by id | `--resume --session-id <id>` | `--session-id` requires `--resume`. |
| Resume latest | `--resume` alone | Continues the most recently used session. Never emit this. |
| Discard history | `--no-session` | Discards the session. Do not use it for Multica chats. |
| Legacy path | `--path` | Extracts an id from a `.jsonl` filename. Legacy. |

Documented id examples:

- `20250325_200615` from `goose run --help` `--path`
- `20251108_1` from the CLI commands guide

Storage is SQLite from Goose 1.10.0. `goose info` on 1.48.0 reports `~/.local/share/goose/sessions/sessions.db`.

No archive-on-execute flag appears in `goose run --help`. The discard path is `--no-session`, which we will not send. Canary a follow-up `goose run --resume --session-id <id>` after a first `run` before claiming resume is safe.

## Skills

`goose skills list` on a clean machine printed two builtin skills and this empty-state string in the binary:

```
No skills installed.
  - ~/.agents/skills/ (global)
  - ~/.agents/plugins/*/skills/ (installed plugins)
  - .agents/skills/ (in current project)
```

Write Multica skills to `{workDir}/.agents/skills/<name>/SKILL.md`. That is the project path the 1.48.0 binary names. Do not invent `.goose/skills` as the Multica write path unless a canary shows 1.48.0 scans it and does not scan `.agents/skills/`. Older community docs still mention `./.goose/skills/` and `~/.config/goose/skills/`. Those are legacy or extra roots. Confirm with `goose skills list` after writing one file.

Do not write into the operator's home directories.

The skills platform extension is enabled by default (`goose info -v`).

## MCP and extensions

| Flag | Meaning |
|---|---|
| `--with-extension <COMMAND>` | Add a stdio MCP-style extension for this run. Repeatable. Optional `name:` prefix. |
| `--with-streamable-http-extension <URL>` | Add a streamable HTTP extension. Repeatable. |
| `--with-builtin <NAME>` | Enable bundled builtins. Comma-separated. |
| `--no-profile` | Ignore the user's default extensions. Use only CLI-specified ones. |
| `--container <id>` | Run extensions inside Docker. Out of scope. |

Isolated Multica MCP is `--no-profile --with-extension ...` after a canary. Hold the MCP UI tab until that canary. `goose mcp` runs Goose's own bundled MCP servers. That is not the Multica forwarding path.

## Auth

- Interactive. `goose configure`
- Env. `GOOSE_PROVIDER`, `GOOSE_MODEL`, plus the vendor key for that provider (`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, and others)
- Config file. `~/.config/goose/config.yaml`
- Secrets. system keyring, or `~/.config/goose/secrets.yaml`

Multica is not the Goose IdP.

## ACP

`goose acp` exists. It speaks Agent Client Protocol on stdio. Out of scope for v1. Block `acp` and `serve` in custom args.

## Official docs used

- [CLI commands](https://github.com/block/goose/blob/main/documentation/docs/guides/goose-cli-commands.md)
- [Running tasks](https://github.com/block/goose/blob/main/documentation/docs/guides/running-tasks.md)
- [Sessions](https://block-goose.mintlify.app/concepts/sessions)
- Captured local `--help` in [help/](help/)
