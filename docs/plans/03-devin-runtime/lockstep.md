# Lockstep inventory

Every new protocol family touches this set. Amp's live run confirmed it. Do not ship a backend that skips a row.

| Surface | File or action | Devin v1 |
|---|---|---|
| Family whitelist | `server/pkg/agent/agent.go` `SupportedTypes` and `New` | add `"devin"` |
| Launch header | `launchHeaders` | `"devin --print"` |
| Factory test | `agent_supported_types_test.go` want map | add `"devin"` |
| CHECK constraint | new migration pair, `NOT VALID`, no FKs | next free number after rebase |
| Frontend family tuple | `packages/core/types/agent.ts` `RUNTIME_PROFILE_PROTOCOL_FAMILIES` | add `"devin"` |
| Models | `ListModels`, `ModelSelectionSupported` | empty catalog, unsupported until `devin models list --format json` canary |
| Daemon probe | `server/internal/daemon/agents_probe.go` | `probe("MULTICA_DEVIN_PATH", "devin", "")` until model canary |
| ExtraArgs env | `config.go` `shellArgsFromEnv("MULTICA_DEVIN_ARGS")` and ExtraArgs forward | both sides or neither |
| Command names | `defaultAgentCommandNames` and `scripts/agent-cli-command-names.txt` | add `devin` sorted |
| Min version | `server/pkg/agent/version.go` | omit until a real floor is known. 3000.6.2 is the planning binary, not a floor. |
| Metrics | `server/internal/metrics/labels.go` `knownRuntimeProviders` | add `"devin"` |
| Execenv brief | `runtimeConfigPath` | `AGENTS.md` |
| Skills dir | `skillsDirPath` | `{workDir}/.devin/skills` |
| Sidecar manifest | `sidecar_manifest_test.go` `allFileBasedProviders` | add `"devin"` |
| MCP tab | `packages/core/agents/mcp-support.ts` | omit until isolated `--config` canary |
| User MCP merge | `loadRuntimeMcpServerConfigs` | omit |
| Logo | `packages/views/runtimes/components/provider-logo.tsx` and test | letter mark or official mark |
| Docs en/zh/ko/ja | providers, install, env, self-host | Devin stays Latin in Chinese |
| Creating-agents skill | `multica-creating-agents` plus source-map | argv rules if create-time flags exist |
| Landing count | `apps/web/features/landing/i18n/{en,zh,ko,ja}.ts` | recount the list at implement time. Main today says 23. |

Concurrent indexes only, one statement per file, if any index is added. This family should not need a new table.
