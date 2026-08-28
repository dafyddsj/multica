# Lockstep inventory

Every new protocol family touches this set. Amp's live run confirmed it. Do not ship a backend that skips a row.

| Surface | File or action | Goose v1 |
|---|---|---|
| Family whitelist | `server/pkg/agent/agent.go` `SupportedTypes` and `New` | add `"goose"` |
| Launch header | `launchHeaders` | `"goose run (stream-json)"` |
| Factory test | `agent_supported_types_test.go` want map | add `"goose"` |
| CHECK constraint | new migration pair, `NOT VALID`, no FKs | next free number after rebase |
| Frontend family tuple | `packages/core/types/agent.ts` `RUNTIME_PROFILE_PROTOCOL_FAMILIES` | add `"goose"` |
| Models | `ListModels`, `ModelSelectionSupported` | empty catalog, unsupported until canary |
| Daemon probe | `server/internal/daemon/agents_probe.go` | `probe("MULTICA_GOOSE_PATH", "goose", "")` until model canary |
| ExtraArgs env | `config.go` `shellArgsFromEnv("MULTICA_GOOSE_ARGS")` and ExtraArgs forward | both sides or neither |
| Command names | `defaultAgentCommandNames` and `scripts/agent-cli-command-names.txt` | add `goose` sorted |
| Min version | `server/pkg/agent/version.go` | omit until a real `--version` floor is known. 1.48.0 is the planning binary, not a floor. |
| Metrics | `server/internal/metrics/labels.go` `knownRuntimeProviders` | add `"goose"` |
| Execenv brief | `runtimeConfigPath` | `AGENTS.md` |
| Skills dir | `skillsDirPath` | `{workDir}/.agents/skills` |
| Sidecar manifest | `sidecar_manifest_test.go` `allFileBasedProviders` | add `"goose"` |
| MCP tab | `packages/core/agents/mcp-support.ts` | omit until isolated `--no-profile --with-extension` canary |
| User MCP merge | `loadRuntimeMcpServerConfigs` | omit. Claude and Codex merge user files. Goose v1 does not. |
| Logo | `packages/views/runtimes/components/provider-logo.tsx` and test | letter mark or official mark |
| Docs en/zh/ko/ja | providers, install, env, self-host | Goose stays Latin in Chinese |
| Creating-agents skill | `server/internal/service/builtin_skills/multica-creating-agents/` plus source-map | argv rules if create-time flags exist |
| Landing count | `apps/web/features/landing/i18n/{en,zh,ko,ja}.ts` | recount the list at implement time. Main today says 23. |

Concurrent indexes only, one statement per file, if any index is added. This family should not need a new table.
