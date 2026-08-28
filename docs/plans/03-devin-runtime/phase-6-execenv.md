# Phase 6. Execenv

Back to [overview](overview.md).

## Goal

A task workdir for provider `devin` receives the Multica runtime brief in `AGENTS.md`. Skills land in the directory `devin skills paths` names for project skills.

## Changes

- `server/internal/daemon/execenv/runtime_config.go`. Add `"devin"` to the `AGENTS.md` case list.
- `runtime_config_test.go`. One row `{"devin", "AGENTS.md"}`.
- `skillsDirPath`. `"devin"` returns `{workDir}/.devin/skills`.
- `sidecar_manifest_test.go` `allFileBasedProviders`. Add `"devin"`.
- Leave `providerNeedsInlineSystemPrompt` unchanged until the AGENTS.md canary.

Do not write Multica skills into `.agents/skills/` for Devin. Goose and Amp already claim that path.

Do not add Devin to `loadRuntimeMcpServerConfigs`. Hold `MCP_SUPPORTED_PROVIDERS` until the isolated `--config` canary.

## Data structures

No new types. `skillsDirPath(workDir, "devin")` returns `{workDir}/.devin/skills`.

## Verification

**Static.** `cd server && go test ./internal/daemon/execenv -run 'RuntimeConfig|WriteContextFilesDevin|Sidecar' -count=1`

**Runtime.** Skills canary in [testing.md](testing.md).
