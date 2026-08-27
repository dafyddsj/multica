# Testing

Back to [overview](overview.md).

## Default CI

No real Amp binary. Tests install a fake executable that prints fixture JSONL and exits 0.

Required coverage, next to `amp.go`:

- argv for a fresh turn (no prompt on argv)
- argv for resume (`threads continue <id>`)
- blocked custom args cannot break stream-json or permissions
- ExtraArgs precede CustomArgs
- stdin payload is one JSONL user message
- fixture parse of init, assistant text, tool_use, success result
- `Result.SessionID` keeps the `T-` prefix
- refused continue sets `ResumeRejected`
- `New("amp")` works and an unknown type still fails
- `SupportedTypes` matches the latest CHECK whitelist

Daemon and execenv tests cover probe command names and `AGENTS.md` path only.

## Real Amp

Behind `agentintegration` and `MULTICA_RUN_REAL_AGENT_SMOKE=1`, same rule as other CLIs in CLAUDE.md.

```bash
cd server && MULTICA_RUN_REAL_AGENT_SMOKE=1 go test -tags=agentintegration ./pkg/agent -run 'TestAmp' -count=1 -v
```

That job may consume Amp quota. It must check the env flag before `LookPath("amp")`.

Add `amp` to `scripts/agent-cli-command-names.txt` in phase 5 so default Linux/macOS tests do not execute a user-installed Amp by accident.

## Surfaces

- CLI / daemon: `control-cli` if the plugin is present. Otherwise `go test` plus a manual `make daemon` on a machine with Amp.
- Web picker: no `control-ui` skill in this checkout. Say so in the PR if the picker was not clicked.
- Docs: render the two pages or grep the tables.

## Canary before inline prompt

After a real Amp is available, write a unique token into `{cwd}/AGENTS.md` and run `amp --execute "repeat the token" --stream-json`. If the token comes back, keep Amp out of `providerNeedsInlineSystemPrompt`. If it does not, add it in a follow-up with a captured fixture.
