# Testing

Back to [overview](overview.md).

## Default CI

No real Devin binary. Tests install a fake executable that prints fixture text or JSON and exits 0.

Required coverage, next to `devin.go`:

- argv for a fresh turn (`--print --prompt-file … --permission-mode dangerous --respect-workspace-trust false`)
- prompt is in the file, not on argv
- argv for resume (`--resume <id>`, never `-c`, never bare `--resume`)
- blocked custom args cannot break print mode, permissions, or resume
- ExtraArgs precede CustomArgs
- `Result.SessionID` is a parsed CLI id or empty
- a flag-like or empty `ResumeSessionID` fails `Execute` before spawn
- cloud-shaped `devin-…` ids fail the local constructor until a canary says they are valid
- `New("devin")` works and an unknown type still fails
- `SupportedTypes` matches the latest CHECK whitelist

Daemon and execenv tests cover probe command names and `AGENTS.md` path only.

## Real Devin

Behind `agentintegration` and `MULTICA_RUN_REAL_AGENT_SMOKE=1`, same rule as other CLIs in CLAUDE.md.

```bash
cd server && MULTICA_RUN_REAL_AGENT_SMOKE=1 go test -tags=agentintegration ./pkg/agent -run 'TestDevin' -count=1 -v
```

That job may consume Devin quota. It must check the env flag before `LookPath("devin")`.

Add `devin` to `scripts/agent-cli-command-names.txt` in phase 5 so default Linux and macOS tests do not execute a user-installed Devin by accident.

## Surfaces

- CLI / daemon. `control-cli` if the plugin is present. Otherwise `go test` plus a manual `make daemon` on a machine with Devin.
- Web picker. No `control-ui` skill in this checkout. Say so in the PR if the picker was not clicked.
- Docs. Render the locale pages or grep the tables.

## Canaries before ship

1. AGENTS.md. Write a unique token into `{cwd}/AGENTS.md` and run `devin --print --prompt-file … --respect-workspace-trust false`. If the token comes back, keep Devin out of `providerNeedsInlineSystemPrompt`.
2. Skills. Write `{cwd}/.devin/skills/canary/SKILL.md` and run `devin skills list`. The row must appear.
3. Session id. After one `--print`, run `devin list --format json` and lock the constructor to the emitted id field.
4. Resume. Second `--print --resume <id>` must continue the first turn. Confirm `-c` is never on argv.
5. Stdin. Optional. Only switch off `--prompt-file` if `devin --print` reads stdin and closes cleanly.
6. MCP. Only if v1 wants the UI tab. A temp `--config` must attach a Multica-owned server for one run without writing the user profile.
