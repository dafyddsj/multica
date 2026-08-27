# Amp runtime

## Context

Multica already hosts coding-agent CLIs through `server/pkg/agent.Backend`. The daemon probes PATH, registers an `AgentRuntime`, and spawns the CLI for a claimed task. Amp is not in that set. Users who already run [Amp](https://ampcode.com) cannot assign Multica issues to it.

Amp's headless contract is documented. `amp --execute --stream-json` emits Claude-compatible JSONL. Thread ids look like `T-<uuid>`. The next turn continues with `amp threads continue`. Auth stays on Amp (`AMP_API_KEY` or a completed CLI login). The Multica daemon is already the host process. Amp orbs and `amp --no-tui` runners are a second product, not a prerequisite for calling Amp a Multica runtime.

## Scope

Included:

- A protocol family `amp` in `SupportedTypes`, `New()`, and the `runtime_profile.protocol_family` CHECK.
- An `ampBackend` that launches the `amp` CLI, parses stream-json into existing `Message` / `Result` types, and stores Amp's `session_id` for resume.
- Daemon probe (`MULTICA_AMP_PATH`, `amp` on PATH), `defaultAgentCommandNames`, and `scripts/agent-cli-command-names.txt`.
- Runtime brief in `{workDir}/AGENTS.md` through `runtimeConfigPath`.
- ExtraArgs so `MULTICA_AMP_ARGS` is not plumbed then dropped.
- Frontend family list, logo, metrics allow-list, install docs, providers table, env docs, README / CLI_AND_DAEMON / SELF_HOSTING, and the creating-agents skill source map when Amp has create-time argv rules.
- `--mcp-config` only after a real `amp` accepts a file path. The Amp manual documents inline JSON. Hold the MCP UI tab until that canary. If the CLI rejects paths, either pass hardened inline JSON or leave MCP off v1.

Excluded:

- Amp orbs (`executor: 'orb'`).
- Long-lived `amp --no-tui` runners and ampcode.com remote thread creation.
- `@ampcode/sdk` as the daemon transport.
- Multica acting as an Amp identity provider.
- Extracting a shared stream-json protocol family from Claude, CodeBuddy, and Qwen.
- Treating Amp as a `BuiltinRuntime` on the `claude` family.

## Constraints

- `Backend.Execute` stays the only production entry. Do not add a parallel agent API.
- Amp is not a Claude Code fork. Claude argv (`-p --output-format stream-json --resume`) must not be sent to `amp`.
- Prompt text goes on stdin, not argv. Qwen and Cursor already hit Windows quote breakage with prompts on the command line (`qwen.go`, #6082, #5649). Amp accepts `echo prompt | amp --execute --stream-json`.
- Unattended runs must not wait on Amp permission prompts. Pin `--dangerously-allow-all` (or the current Amp equivalent from `amp --help`) in argv and block user overrides.
- Do not add Amp to `providerNeedsInlineSystemPrompt` until a canary in `AGENTS.md` fails. Amp's manual says it loads that file.
- No database foreign keys. The CHECK widen is a new pair of migration files, `NOT VALID`, following `server/migrations/403_runtime_profile_add_zeroclaw.up.sql`.
- `resumeRejectionUndetectable` is opt-in. Leave Amp off that map unless resume rejection cannot be detected.
- Amp does not promise protocol stability. Fixture tests against captured JSONL, not live Amp, in default CI.

## Alternatives

**New `amp` protocol family.** Own argv and resume verb. Reuse `newAgentStreamScanner` and shared result helpers. Same registration shape as Qwen. Smallest change that does not lie about the CLI.

**BuiltinRuntime on `claude`.** Rejected. `NewRuntime` would launch Claude flags against `amp`. Only `piBackend` implements `applyBuiltinRuntimeOverrides` today.

**Extract a stream-json family, then hang Claude, CodeBuddy, Qwen, and Amp on descriptors.** Rejected for v1. The three existing backends already diverge on argv, stdin, resume, and permission flags. Amp adds a fourth. An extract would touch every stream-json backend to add one CLI. Revisit only after a second Amp-like CLI appears.

**Shell out to `@ampcode/sdk`.** Rejected. The SDK wraps the same CLI. The daemon would grow a Node dependency for no new protocol.

Chosen shape is the first. Callers keep `ResolveBackend("amp")`. Complexity stays inside `amp.go`. Arena confirmed it. Do not extract a shared stream-json runner to add Amp.

## Applicable skills

- **how** over `server/pkg/agent` and `server/internal/daemon/execenv` before editing them.
- **architect** if implementation friction repeats (Phase E signal). Do not re-arena the settled family choice.
- **unslop** on docs, skill copy, commit messages, and PR text.
- **technical-writing** on `apps/docs` and this plan.
- Cursor **create-skill** if a phase edits `SKILL.md`.
- `/deslop` before each commit. `/no-comments` before review.
- **show-me-your-work** if the implementer keeps a decision trail. Seed log lives at `/tmp/amp-runtime-grounding/decisions/amp-runtime.tsv` for this planning run.
- `control-cli` for daemon/CLI verification. There is no `control-ui` skill in this checkout. Flag that in each UI phase.

Caller usage is in [USAGE.md](USAGE.md).

## Phases

1. [Amp argv](phase-1-amp-argv.md)
2. [Execute and parse](phase-2-execute-parse.md)
3. [Resume](phase-3-resume.md)
4. [Factory and whitelist](phase-4-factory-whitelist.md)
5. [Daemon probe](phase-5-daemon-probe.md)
6. [Execenv brief](phase-6-execenv-brief.md)
7. [Frontend lockstep](phase-7-frontend.md)
8. [Docs and skills](phase-8-docs-skills.md)

[Testing](testing.md) is the shared verification recipe.

## Verification

Project-level:

```bash
cd server && go test ./pkg/agent ./internal/daemon ./internal/daemon/execenv -count=1
pnpm --filter @multica/core test -- runtimes display mcp-support
pnpm --filter @multica/views test -- provider-logo
```

A real Amp binary is optional and must stay behind `MULTICA_RUN_REAL_AGENT_SMOKE=1` plus the `agentintegration` tag. Default tests use a fake executable.

## Implementation guidance

- Run the **how** skill on `server/pkg/agent` and execenv before changing them.
- Do not re-run **architect** unless the sketch below starts needing the same workaround twice.
- Run **interrogate** only if someone reopens orbs, SDK transport, or a stream-json extract.
- `/deslop` each diff before commit. **unslop** every doc, commit message, and PR body.
- Keep a decision trail with **show-me-your-work** if the work spans more than one PR.
- After the PR opens, use Cursor's built-in **babysit** skill.

## Shape

Callers still call `ResolveBackend("amp")` and get a `Backend`.

```
ampBackend { cfg Config }
Execute(ctx, prompt, opts) (*Session, error)
resolveAmpResume(id) (ampThreadID, error)
buildAmpArgs(resume ampThreadID, opts) []string
writeAmpInput(w, prompt) error   // plaintext, then close
```

`ampThreadID` has parse-only constructors. The zero value is a fresh run. A non-empty `ResumeSessionID` that is not `T-<uuid>` fails `Execute` before spawn. `Result.SessionID` is only a parsed `T-<uuid>` or empty. Never persist a raw unparsed token. Never emit `amp threads continue` without that id. Bare `threads continue` follows the latest thread on the Amp account, which races on a shared daemon host.

Argv sketch (confirm against `amp --help` in phase 1):

```
amp --execute --stream-json --stream-json-thinking --dangerously-allow-all
```

Resume prefix:

```
amp threads continue <T-uuid> --execute --stream-json --stream-json-thinking --dangerously-allow-all
```

`--dangerously-allow-all` is unconditional and sits in `ampBlockedArgs`. Prompt is plaintext on stdin, then stdin closes. That is Amp's documented `echo prompt | amp -x` path. Do not keep stdin open for Claude `control_request`. Do not enable `--stream-json-input` in v1. `--stream-json-thinking` is on so thinking and `redacted_thinking` blocks parse. Dropping it is one argv line if a capture shows it breaks a release.

`ExecOptions.Model` and `ThinkingLevel` are ignored. `ModelSelectionSupported("amp")` is false. `ListModels("amp")` returns an empty catalog. Do not probe `MULTICA_AMP_MODEL`. Amp's product dial is mode and effort, documented as SDK extras, not a verified CLI `--model`. ExtraArgs can carry those later. Do not advertise a dead knob.

Windows copies Qwen's `.cmd` to `.ps1` shim so npm's `amp.cmd` does not re-tokenize argv.

## Synthesis decision

Arena scored three structurally different sketches. Candidate 1 (standalone `ampBackend` after `qwen.go`) is the base. Candidate 2 extracted a private `streamJSONBackend` and rewrote Claude, CodeBuddy, and Qwen to pay for Amp. Candidate 3 used an `ampKind` fresh/continue enum but persisted raw session ids on parse failure and wired `--model` / `MULTICA_AMP_MODEL` that the CLI contract does not list.

The extract is still rejected. It deletes the cheap wire structs and couples Amp's unstable frames to three production backends. The hard contracts (`finalizeStreamResult`, `resolveFallback`, `resumeWasRejected`) are already shared.

Two grafts:

- From candidate 2, filter `LaunchPrefix` as sequences. Stripping `threads` or `continue` as standalone tokens can leave a stray `T-` positional. Remove the whole `threads continue <id>` run, or nothing.
- From candidate 3, block `--no-tui`, `--executor`, `-p`, and `--output-format` so an operator pasting Claude or orb flags cannot change the launch.

Leave `providerNeedsInlineSystemPrompt` and `resumeRejectionUndetectable` untouched unless a canary or captured stderr says otherwise.

## Throughput checkpoint

`throughput checkpoint: n/a, read-only investigation`
