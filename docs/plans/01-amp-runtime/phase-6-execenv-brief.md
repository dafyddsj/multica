# Phase 6. Execenv brief

Back to [overview](overview.md).

## Goal

A task workdir for provider `amp` receives the Multica runtime brief in `AGENTS.md`. The brief is not also inlined on the prompt.

## Changes

- `server/internal/daemon/execenv/runtime_config.go`: add `"amp"` to the `AGENTS.md` case list in `runtimeConfigPath`.
- `runtime_config_test.go`: one row `{"amp", "AGENTS.md"}` next to the existing table.
- Leave `providerNeedsInlineSystemPrompt` unchanged. Record a follow-up to canary-probe a real Amp the way MUL-5392 did for Claude.

Skills directory. Amp reads `AGENTS.md` and its own plugin settings. If Amp documents a project skills dir, add it to `skillsDirPath` and `localSkillRootsForProvider`. If it does not, leave the dim/zeroclaw fallback (`.agent_context/skills`, no user-level import) and say so in providers.mdx. Do not invent `.amp/skills`.

- `sidecar_manifest_test.go` `allFileBasedProviders`: add `"amp"`. The comment on that list says a new workdir writer must appear there.
- Do not add Amp to `loadRuntimeMcpServerConfigs`. Claude/Codex merge user MCP files. CodeBuddy does not.
- MCP forwarding waits on a canary. Amp's manual documents `--mcp-config` as inline JSON. If a real `amp` accepts a file path, reuse `writeMcpConfigToTemp` (0600, browser-MCP hardening) and then add the UI tab. If it rejects paths, either pass the hardened JSON inline (accepting argv visibility to same-host users) or leave MCP off v1. Do not add `"amp"` to `MCP_SUPPORTED_PROVIDERS` until that canary.

## Data structures

No new types. `runtimeConfigPath(workDir, "amp")` returns `{workDir}/AGENTS.md`.

## Verification

**Static.** `cd server && go test ./internal/daemon/execenv -run 'RuntimeConfig' -count=1`

**Runtime.** None in default CI. The canary probe is a manual `amp --execute` in a temp dir with a unique token in `AGENTS.md`.
