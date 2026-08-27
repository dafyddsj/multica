# Phase 6. Execenv brief

Back to [overview](overview.md).

## Goal

A task workdir for provider `amp` receives the Multica runtime brief in `AGENTS.md`. The brief is not also inlined on the prompt.

## Changes

- `server/internal/daemon/execenv/runtime_config.go`: add `"amp"` to the `AGENTS.md` case list in `runtimeConfigPath`.
- `runtime_config_test.go`: one row `{"amp", "AGENTS.md"}` next to the existing table.
- Leave `providerNeedsInlineSystemPrompt` unchanged. Record a follow-up to canary-probe a real Amp the way MUL-5392 did for Claude.

Skills directory. Amp reads `AGENTS.md` and its own plugin/skills settings. If Amp documents a project skills dir, seed it in execenv in this phase. If it does not, skip rather than invent `.amp/skills`.

## Data structures

No new types. `runtimeConfigPath(workDir, "amp")` returns `{workDir}/AGENTS.md`.

## Verification

**Static.** `cd server && go test ./internal/daemon/execenv -run 'RuntimeConfig' -count=1`

**Runtime.** None in default CI. The canary probe is a manual `amp --execute` in a temp dir with a unique token in `AGENTS.md`.
