# Devin runtime

## Context

Multica already hosts coding-agent CLIs through `server/pkg/agent.Backend`. Devin is not in that set.

Devin is Cognition's agent. It is not cloud-only. Cognition ships a local CLI named `devin` and a separate Organization REST API for cloud sessions. This plan used Devin CLI 3000.6.2 (`devin 3000.6.2 (ce8ebcc1)`) downloaded from the official manifest at `https://static.devin.ai/cli/current/manifest.json`. Headless work is `devin --print`. Auth stays on Devin (`devin auth login`, credentials in `~/.local/share/devin/credentials.toml`).

Amp on `cursor/amp-runtime-plan-6a73` is the method precedent. This branch starts from `main`. Do not stack on the Amp PR.

## Is Devin a local-CLI runtime?

Yes. v1 is the local CLI on the daemon host.

The cloud API at `https://api.devin.ai/v3/organizations/*/sessions` creates a Cognition-hosted session, not a process on the Multica daemon. `devin cloud` manages Declarative Repo Setup, sandbox boxes, and builds. `devin ssh` opens those boxes. Those are a second product, the way Amp orbs were a second product. They do not force a remote execute path for v1.

Do not defer. The CLI installs, prints `--help`, and documents `--print` for scripts.

## Scope

Included:

- A protocol family `devin` in `SupportedTypes`, `New()`, and the `runtime_profile.protocol_family` CHECK.
- A `devinBackend` that launches `devin --print`, loads the prompt from `--prompt-file`, stores a parsed session id for resume, and fails closed on a bare `--continue` or a bare `--resume`.
- Daemon probe (`MULTICA_DEVIN_PATH`, `devin` on PATH), `defaultAgentCommandNames`, and `scripts/agent-cli-command-names.txt`.
- Runtime brief in `{workDir}/AGENTS.md`.
- ExtraArgs so `MULTICA_DEVIN_ARGS` is not plumbed then dropped.
- Frontend family list, logo, metrics allow-list, install docs, providers table, env docs, README / CLI_AND_DAEMON / SELF_HOSTING, creating-agents skill source map, and the landing tool count.
- MCP UI tab only after a canary proves a per-run isolated config. `devin mcp add` writes durable user config. That is not isolation.

Excluded:

- Organization API v3, Enterprise API, and legacy v1/v2 session create.
- `devin cloud`, `devin ssh`, Outposts `devin worker`, and Desktop handoff.
- `@cognition` SDKs as the execute path.
- `devin acp` as the Multica transport.
- Multica acting as a Devin identity provider.
- Treating Devin as a Claude fork or a `BuiltinRuntime`.
- Bare `devin --continue` / `-c`. That resumes the most recent session in the current directory on a shared daemon host.
- Bare `devin --resume` / `-r` with no id. That opens an interactive picker, which a daemon cannot answer.

## Constraints

- `Backend.Execute` stays the only production entry.
- Devin argv is not Claude argv. Do not send `--output-format stream-json`.
- Prompt text goes through `--prompt-file` on a 0600 temp file, then the file is removed. Official help also allows `-p "prompt"`. Do not put the Multica prompt on argv. Qwen and Cursor already hit Windows quote breakage that way.
- Official help does not document stdin as the `--print` prompt. Do not assume stdin until a canary shows `devin --print` reads it.
- Unattended runs must not wait on permission prompts. Pin `--permission-mode dangerous` and block user overrides. `--sandbox` plus `autonomous` is a different product. Leave it off v1 unless a host already has `bwrap` and wants that follow-up.
- `--print` cannot show the workspace trust prompt. Pin `--respect-workspace-trust false` for daemon runs.
- Session ids are parse-only. Official examples are `brisk-otter` and `abc12345`. The binary does not document a regex. Persist only a token that matches the constructor, and tighten the constructor after `devin list --format json` is captured from a real session.
- Never emit `-c` / `--continue`. Never emit `-r` / `--resume` without an id.
- No archive-on-execute flag appears in `--help`. Canary resume after `--print` before claiming follow-up chats work.
- No database foreign keys. CHECK widen is `NOT VALID`. Rebase and take the next free migration number after `458` and after Amp if Amp landed.
- Default CI must never resolve a user-installed `devin`. Fake path or missing path. Real smoke behind `agentintegration` and `MULTICA_RUN_REAL_AGENT_SMOKE=1`.

## Alternatives

**New `devin` protocol family on the local CLI.** Own argv and resume. Reuse `newAgentStreamScanner` only if `--print` actually emits JSONL. Otherwise parse printed text plus `--export` ATIF as a follow-up, and keep v1 on stdout text plus a parsed session id from `devin list --format json` or the export file. Same registration shape as Qwen.

**Cloud Organization API as Execute.** Rejected for v1. The daemon would become a Cognition API client. Work would run on a remote VM, not in the task workdir. Auth would be a `cog_` service user. That is a different product.

**`devin acp` as an ACP family next to Hermes.** Rejected for v1. `--print` is the documented script path. ACP is an editor host. Amp already refused a shared protocol extract for one sibling.

**Defer until Cognition publishes a daemon-shaped SDK.** Rejected. The local CLI is installable and documented.

Chosen shape is the first. Callers keep `ResolveBackend("devin")`. Complexity stays inside `devin.go`.

## Applicable skills

- **how** over `server/pkg/agent` and `server/internal/daemon/execenv` before editing them.
- **architect** if implementation friction repeats. Do not re-arena the settled family choice or the cloud-API reject.
- **unslop** on docs, skill copy, commit messages, and PR text.
- **technical-writing** on `apps/docs` and this plan.
- Cursor **create-skill** if a phase edits `SKILL.md`.
- `/deslop` before each commit. `/no-comments` before review.
- **show-me-your-work** if the implementer keeps a decision trail. Planning log is `.audit/goose-devin-runtime-plan.tsv`.

Caller usage is in [USAGE.md](USAGE.md). Captured `--help` is in [help/](help/). Contract notes are in [EVIDENCE.md](EVIDENCE.md).

## Phases

1. [Devin argv](phase-1-argv.md)
2. [Execute and parse](phase-2-execute-parse.md)
3. [Resume](phase-3-resume.md)
4. [Factory and whitelist](phase-4-factory-whitelist.md)
5. [Daemon probe](phase-5-daemon-probe.md)
6. [Execenv](phase-6-execenv.md)
7. [Frontend lockstep](phase-7-frontend.md)
8. [Docs and skills](phase-8-docs-skills.md)

[Testing](testing.md) is the shared verification recipe. [Lockstep](lockstep.md) is the file inventory.

Goose is a separate family. See [02-goose-runtime](../02-goose-runtime/overview.md). Do not share a backend.

## Verification

Project-level:

```bash
cd server && go test ./pkg/agent ./internal/daemon ./internal/daemon/execenv -count=1
pnpm --filter @multica/core test -- runtimes display mcp-support
pnpm --filter @multica/views test -- provider-logo
```

A real Devin binary is optional and must stay behind `MULTICA_RUN_REAL_AGENT_SMOKE=1` plus the `agentintegration` tag. Default tests use a fake executable.

## Implementation guidance

- Run the **how** skill on `server/pkg/agent` and execenv before changing them.
- Do not re-run **architect** unless the sketch below starts needing the same workaround twice.
- Run **interrogate** only if someone reopens the cloud API, ACP transport, or a Claude-family reuse.
- `/deslop` each diff before commit. **unslop** every doc, commit message, and PR body.
- Keep a decision trail with **show-me-your-work** if the work spans more than one PR.
- After the PR opens, use Cursor's built-in **babysit** skill.

There is no `control-ui` skill in this checkout. Flag that in the frontend phase.

## Shape

Callers still call `ResolveBackend("devin")` and get a `Backend`.

```
devinBackend { cfg Config }
Execute(ctx, prompt, opts) (*Session, error)
resolveDevinResume(id) (devinSessionID, error)
buildDevinArgs(resume devinSessionID, opts, promptFile string) []string
writeDevinPromptFile(prompt) (path string, cleanup func(), err error)
```

`devinSessionID` has parse-only constructors. The zero value is a fresh run. A non-empty `ResumeSessionID` that does not parse fails `Execute` before spawn.

Until `devin list --format json` is captured, the constructor accepts a non-empty token of `[A-Za-z0-9][A-Za-z0-9_-]{1,64}` that does not start with `-`. That covers `brisk-otter` and `abc12345` from the official docs. Fail closed on spaces, flags, paths, and URLs. Tighten after the JSON list canary. Never persist a raw unparsed string.

Argv sketch, confirmed against `devin --help` in 3000.6.2:

```
devin --print --prompt-file <tmp> --permission-mode dangerous --respect-workspace-trust false
```

Resume:

```
devin --print --resume <parsed-id> --prompt-file <tmp> --permission-mode dangerous --respect-workspace-trust false
```

`--model` is a real flag and reads `DEVIN_MODEL`. v1 may pass `ExecOptions.Model` when set. `ListModels("devin")` can call `devin models list --format json` later. Keep the catalog empty and `ModelSelectionSupported` false until that canary. The CLI requires a logged-in account to list models.

`--export` writes ATIF after each turn. Use it only if `--print` stdout is not enough to recover a session id. Do not make ATIF the execute transport.

Windows. Official install is a zip plus `devin.exe`. Confirm argv tokenization before assuming the Linux binary's flag order.

## Synthesis decision

Arena scored four structurally different sketches.

- Candidate 1. Standalone `devinBackend` on `--print`. This is the base.
- Candidate 2. Organization API v3 sessions. Rejected for v1. Remote VM, service-user auth, not the task workdir.
- Candidate 3. `devin acp` reused as an ACP family. Rejected for v1. `--print` is the script contract.
- Candidate 4. Defer. Rejected. The local CLI is real.

Two grafts:

- From candidate 2, keep cloud IDs (`devin-abc123…` from `devin cloud drs`) out of the local session constructor. A cloud box id is not a CLI `--resume` id unless a canary proves they are the same token.
- From candidate 3, block `acp`, `cloud`, `ssh`, `desktop`, `--continue`, `-c`, and a value-less `--resume` in custom args and launch prefix.

Leave `providerNeedsInlineSystemPrompt` and `resumeRejectionUndetectable` untouched unless a canary says otherwise.

## Throughput checkpoint

`throughput checkpoint: n/a, plan only`
