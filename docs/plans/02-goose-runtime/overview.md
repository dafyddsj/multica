# Goose runtime

## Context

Multica already hosts coding-agent CLIs through `server/pkg/agent.Backend`. The daemon probes PATH, registers an `AgentRuntime`, and spawns the CLI for a claimed task. Goose is not in that set. Users who already run [Goose](https://github.com/block/goose) cannot assign Multica issues to it.

Goose is Block's open-source agent. The current GitHub home is [aaif-goose/goose](https://github.com/aaif-goose/goose). The product name stays Goose. This plan used Goose CLI 1.48.0 (`goose --version`) downloaded from that repo's GitHub release. Headless work is `goose run`. Interactive work is `goose session`. Auth stays on Goose (`goose configure`, `GOOSE_PROVIDER`, `GOOSE_MODEL`, and the vendor API key for the chosen provider).

Amp on `cursor/amp-runtime-plan-6a73` is the method precedent, not a sibling to stack on. This branch starts from `main`. Treat Amp as a live lesson list, not as code you import.

## Scope

Included:

- A protocol family `goose` in `SupportedTypes`, `New()`, and the `runtime_profile.protocol_family` CHECK.
- A `gooseBackend` that launches `goose run`, writes the prompt on stdin through `-i -`, parses `--output-format stream-json` into existing `Message` / `Result` types, and stores a parsed Goose session id for resume.
- Daemon probe (`MULTICA_GOOSE_PATH`, `goose` on PATH), `defaultAgentCommandNames`, and `scripts/agent-cli-command-names.txt`.
- Runtime brief in `{workDir}/AGENTS.md` through `runtimeConfigPath`.
- ExtraArgs so `MULTICA_GOOSE_ARGS` is not plumbed then dropped.
- Frontend family list, logo, metrics allow-list, install docs, providers table, env docs, README / CLI_AND_DAEMON / SELF_HOSTING, creating-agents skill source map, and the landing tool count.
- Isolated MCP only after a canary proves `--no-profile --with-extension` accepts a Multica-owned stdio server. Hold the MCP UI tab until that canary.

Excluded:

- Treating Goose as a Claude fork or a `BuiltinRuntime` on `claude`.
- Extracting a shared `streamJSONBackend` because Goose also has `--output-format stream-json`.
- `goose acp` or `goose serve` as the execute path.
- Recipes, schedules, gateways, local-models, and the Desktop app as the Multica transport.
- Multica acting as a Goose identity provider.
- Writing Multica skills into the operator's `~/.agents/skills/` or `~/.config/goose/skills/`.
- Bare `goose run --resume` with no `--session-id`. That continues the most recently used session on a shared daemon host.

## Constraints

- `Backend.Execute` stays the only production entry.
- Goose argv is not Claude argv. Do not send `-p --output-format stream-json --resume`.
- Prompt text goes on stdin through `-i -`. Do not put the Multica prompt on `-t`. Qwen and Cursor already hit Windows quote breakage with prompts on the command line.
- Close stdin after the prompt. Goose reads instructions to EOF.
- Unattended runs must not wait on Goose permission prompts. `goose run` is already non-interactive. Pin `GOOSE_MODE=auto` in the backend env unless a canary shows `run` already skips prompts without it.
- Session ids are parse-only. Persist only a value that matches the constructor. Never persist a raw `session_id` string and later emit `--resume` without proving the CLI owns that id.
- Never emit `--resume` without `--session-id`. The CLI help says a bare `--resume` continues the most recently used session.
- `--no-session` discards history. Do not use it for Multica chats.
- No database foreign keys. The CHECK widen is a new pair of migration files, `NOT VALID`, following `server/migrations/403_runtime_profile_add_zeroclaw.up.sql`. Main's latest numbered migration is `458`. Rebase and take the next free number. If Amp lands first, skip past its number.
- `resumeRejectionUndetectable` is opt-in. Leave Goose off that map unless resume rejection cannot be detected.
- Default CI must never resolve a user-installed `goose`. Fake path or missing path. Real smoke behind `agentintegration` and `MULTICA_RUN_REAL_AGENT_SMOKE=1`.

## Alternatives

**New `goose` protocol family.** Own argv and resume verb. Reuse `newAgentStreamScanner`, `finalizeStreamResult`, `assistantTurn` / `resolveFallback`, `resumeWasRejected`, `startOwnedProcessTree`, and `filterCustomArgs`. Same registration shape as Qwen. Smallest change that does not lie about the CLI.

**BuiltinRuntime on `claude`.** Rejected. Goose `run` flags are not Claude flags. `NewRuntime` would launch Claude argv against `goose`. Only `piBackend` implements `applyBuiltinRuntimeOverrides` today.

**Extract a stream-json family, then hang Claude, CodeBuddy, Qwen, Amp, and Goose on descriptors.** Rejected for v1. The existing stream-json backends already diverge on argv, stdin, resume, and permission flags. Goose adds another dialect (`goose run -i - --output-format stream-json --resume --session-id`). An extract would touch every stream-json backend to add one CLI.

**`goose acp` as an ACP family next to Hermes and Grok.** Rejected for v1. `goose run` is the documented automation path. ACP is a second product for editor hosts. Amp already refused a shared protocol extract for one sibling.

Chosen shape is the first. Callers keep `ResolveBackend("goose")`. Complexity stays inside `goose.go`.

## Applicable skills

- **how** over `server/pkg/agent` and `server/internal/daemon/execenv` before editing them.
- **architect** if implementation friction repeats. Do not re-arena the settled family choice.
- **unslop** on docs, skill copy, commit messages, and PR text.
- **technical-writing** on `apps/docs` and this plan.
- Cursor **create-skill** if a phase edits `SKILL.md`.
- `/deslop` before each commit. `/no-comments` before review.
- **show-me-your-work** if the implementer keeps a decision trail. Planning log is `.audit/goose-devin-runtime-plan.tsv`.

Caller usage is in [USAGE.md](USAGE.md). Captured `--help` is in [help/](help/). Contract notes are in [EVIDENCE.md](EVIDENCE.md).

## Phases

1. [Goose argv](phase-1-argv.md)
2. [Execute and parse](phase-2-execute-parse.md)
3. [Resume](phase-3-resume.md)
4. [Factory and whitelist](phase-4-factory-whitelist.md)
5. [Daemon probe](phase-5-daemon-probe.md)
6. [Execenv](phase-6-execenv.md)
7. [Frontend lockstep](phase-7-frontend.md)
8. [Docs and skills](phase-8-docs-skills.md)

[Testing](testing.md) is the shared verification recipe. [Lockstep](lockstep.md) is the file inventory.

Devin is a separate family. See [03-devin-runtime](../03-devin-runtime/overview.md). Do not share a backend.

## Verification

Project-level:

```bash
cd server && go test ./pkg/agent ./internal/daemon ./internal/daemon/execenv -count=1
pnpm --filter @multica/core test -- runtimes display mcp-support
pnpm --filter @multica/views test -- provider-logo
```

A real Goose binary is optional and must stay behind `MULTICA_RUN_REAL_AGENT_SMOKE=1` plus the `agentintegration` tag. Default tests use a fake executable.

## Implementation guidance

- Run the **how** skill on `server/pkg/agent` and execenv before changing them.
- Do not re-run **architect** unless the sketch below starts needing the same workaround twice.
- Run **interrogate** only if someone reopens ACP transport, a Claude-family reuse, or a stream-json extract.
- `/deslop` each diff before commit. **unslop** every doc, commit message, and PR body.
- Keep a decision trail with **show-me-your-work** if the work spans more than one PR.
- After the PR opens, use Cursor's built-in **babysit** skill.

There is no `control-ui` skill in this checkout. Flag that in the frontend phase. Daemon and CLI checks use `go test` plus `control-cli` if the plugin is present.

## Shape

Callers still call `ResolveBackend("goose")` and get a `Backend`.

```
gooseBackend { cfg Config }
Execute(ctx, prompt, opts) (*Session, error)
resolveGooseResume(id) (gooseSessionID, error)
buildGooseArgs(resume gooseSessionID, opts) []string
writeGooseInput(w, prompt) error   // plaintext, then close
```

`gooseSessionID` has parse-only constructors. The zero value is a fresh run. A non-empty `ResumeSessionID` that does not parse fails `Execute` before spawn. `Result.SessionID` is only a parsed id or empty.

Documented id shapes from Goose 1.48.0 help and docs:

- `20250325_200615` from the `--path` help example
- `20251108_1` from the CLI commands guide

Do not invent a third shape. Persist the id the CLI emits in stream-json or `goose session list --format json`. The constructor accepts `^[0-9]{8}_[A-Za-z0-9]+$` until a capture shows a different official form. Fail closed on anything else.

Argv sketch, confirmed against `goose run --help` in 1.48.0:

```
goose run -i - --output-format stream-json
```

Resume:

```
goose run -i - --output-format stream-json --resume --session-id <parsed-id>
```

`--session-id` requires `--resume`. Never emit one without the other. Never emit `--resume` alone.

`--quiet` suppresses non-response output. Leave it off until a capture shows stream-json still arrives under `-q`. ExtraArgs can add it later.

`--provider` and `--model` are real flags. v1 may pass `ExecOptions.Model` as `--model` when set. `ListModels("goose")` stays an empty catalog until a canary lists models without a signed-in provider. `ModelSelectionSupported("goose")` is false until that canary. The UI then shows "Managed by runtime" instead of a picker that drops values.

Windows copies Qwen's `.cmd` to `.ps1` shim if `goose` is installed through npm. The official 1.48.0 Linux binary is a single executable. Confirm the Windows install shape before adding a shim.

## Synthesis decision

Arena scored three structurally different sketches.

- Candidate 1. Standalone `gooseBackend` after `qwen.go`. This is the base.
- Candidate 2. Reuse `claude.go` because Goose speaks stream-json. Rejected. Resume, stdin, and permission flags do not match.
- Candidate 3. `goose acp` as an ACP family. Rejected for v1. `goose run` is the automation contract. ACP is an editor host.

Two grafts:

- From candidate 2, keep the shared scanner and result helpers. Do not extract a runner type.
- From candidate 3, block `--acp`, `serve`, `--interactive`, `-s`, `--no-session`, and `-t` / `--text` in `gooseBlockedArgs` so an operator cannot change the launch.

Leave `providerNeedsInlineSystemPrompt` and `resumeRejectionUndetectable` untouched unless a canary says otherwise.

## Throughput checkpoint

`throughput checkpoint: n/a, plan only`
