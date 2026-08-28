# Phase 6. Execenv

Back to [overview](overview.md).

## Goal

A task workdir for provider `goose` receives the Multica runtime brief in `AGENTS.md`. Skills land in the directory Goose 1.48.0 actually scans.

## Changes

- `server/internal/daemon/execenv/runtime_config.go`. Add `"goose"` to the `AGENTS.md` case list in `runtimeConfigPath`.
- `runtime_config_test.go`. One row `{"goose", "AGENTS.md"}`.
- `skillsDirPath`. `"goose"` returns `{workDir}/.agents/skills`.
- `sidecar_manifest_test.go` `allFileBasedProviders`. Add `"goose"`.
- Leave `providerNeedsInlineSystemPrompt` unchanged until the AGENTS.md canary.

Do not install Multica skills into `~/.agents/skills/` or `~/.config/goose/skills/`.

Do not add Goose to `loadRuntimeMcpServerConfigs`. Hold `MCP_SUPPORTED_PROVIDERS` until the isolated-extension canary.

## Data structures

No new types. `skillsDirPath(workDir, "goose")` returns `{workDir}/.agents/skills`.

## Verification

**Static.** `cd server && go test ./internal/daemon/execenv -run 'RuntimeConfig|WriteContextFilesGoose|Sidecar' -count=1`

**Runtime.** Skills canary in [testing.md](testing.md).
