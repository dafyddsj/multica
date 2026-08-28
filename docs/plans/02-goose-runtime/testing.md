# Testing

Back to [overview](overview.md).

## Default CI

No real Goose binary. Tests install a fake executable that prints fixture JSONL and exits 0.

Required coverage, next to `goose.go`:

- argv for a fresh turn (`run -i - --output-format stream-json`, no prompt on argv)
- argv for resume (`--resume --session-id <id>`, never `--resume` alone)
- blocked custom args cannot break stream-json, stdin, or resume
- ExtraArgs precede CustomArgs
- stdin payload is plaintext, not a JSONL user frame, and is not on argv
- stdin is closed after the prompt
- fixture parse of init, assistant text, tool use, and success result once a capture exists
- `Result.SessionID` is a parsed Goose id or empty
- a non-matching `ResumeSessionID` fails `Execute` before spawn
- refused resume sets `ResumeRejected` once stderr or stream-json shows a detectable phrase
- `New("goose")` works and an unknown type still fails
- `SupportedTypes` matches the latest CHECK whitelist

Daemon and execenv tests cover probe command names and `AGENTS.md` path only.

## Real Goose

Behind `agentintegration` and `MULTICA_RUN_REAL_AGENT_SMOKE=1`, same rule as other CLIs in CLAUDE.md.

```bash
cd server && MULTICA_RUN_REAL_AGENT_SMOKE=1 go test -tags=agentintegration ./pkg/agent -run 'TestGoose' -count=1 -v
```

That job may consume Goose provider quota. It must check the env flag before `LookPath("goose")`.

Add `goose` to `scripts/agent-cli-command-names.txt` in phase 5 so default Linux and macOS tests do not execute a user-installed Goose by accident.

## Surfaces

- CLI / daemon. `control-cli` if the plugin is present. Otherwise `go test` plus a manual `make daemon` on a machine with Goose.
- Web picker. No `control-ui` skill in this checkout. Say so in the PR if the picker was not clicked.
- Docs. Render the locale pages or grep the tables.

## Canaries before ship

1. AGENTS.md. Write a unique token into `{cwd}/AGENTS.md` and run `goose run -i - --output-format stream-json` with a prompt that asks the model to repeat the token. If the token comes back, keep Goose out of `providerNeedsInlineSystemPrompt`.
2. Skills. Write `{cwd}/.agents/skills/canary/SKILL.md` and run `goose skills list`. The row must appear. If it does not, try `.goose/skills/` only as a second measurement, then change `skillsDirPath` to the path the CLI actually listed.
3. Resume. Run once, parse the session id, run again with `--resume --session-id`. Confirm the second turn sees the first. Confirm a bare `--resume` is never on argv.
4. MCP. Only if v1 wants the UI tab. `goose run --no-profile --with-extension ...` must start the Multica-owned server and ignore the user profile.
