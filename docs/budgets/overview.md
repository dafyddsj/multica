# Budgets

Implementation plan. Do not start coding from this folder until the design checkpoint in the parent chat is accepted.

The synthesized type sketch lives in [design.md](design.md). Read that before any phase file.

## Context

Users can already see spend on Analytics (`/{slug}/usage`). They cannot cap it. Pause is shipped with no flag. Lightweight lanes exist behind `agent_execution_lanes` (default off). The missing piece is a budget on a squad, agent, project, or initiative that can degrade the next task to the cheap lane and, optionally, stop new covered work when the limit is gone.

Spend today is written once at the end of a task (`ReportTaskUsage`), rolled up hourly, and priced in the browser (`estimateCost`). A hard dollar gate cannot live in the browser. The next Multica-controlled model session is the next task, not the next provider call inside a CLI.

## Scope

**In**

- One budget per `(workspace, scope, owner)` for `agent`, `squad`, `project`, `initiative`
- Monthly USD limit, stored as `limit_usd_ticks` (1e-10 USD, same unit as `task_usage.cost_usd_ticks`)
- Optional soften threshold (default 80 percent). NULL means the cheap-lane behavior is off
- Required over-limit choice, `pause` or `allow`
- Analytics Usage tab CRUD and progress
- Server-priced ledger used for bars and for gates
- Claim-time admission (`proceed` / `soften` / `hold`)
- Reuse of `PauseAgent` and `executionlane`, after a pause-service extract
- Removal of `agent_execution_lanes` before any budget promises soften
- Squad attribution as the root squad invocation, stamped onto worker tasks
- Time-bounded waiver. Workspace owner or admin can bypass resource-scope teeth (soften and hold) for one project or one initiative

**Out of v1**

- Mobile UI
- CLI budget commands
- Workspace-wide budget
- Weekly or custom windows
- Mid-run cancel or mid-session model switch
- Inbox or email alerts
- Debit audit UI
- Autopilot soften (holds still apply)
- Chat-session `project_id` as project spend (chat stays in the null-project bucket)
- Waiver of agent or squad teeth. A paused agent stays paused. A project waiver does not unpause anyone.

## Constraints

- No database foreign keys. Each concurrent index is its own migration file.
- Parse API JSON with `parseWithFallback` and a zod schema. Add a malformed-response test.
- Workspace-scoped query keys include `wsId`.
- Zustand does not own budgets. React Query does.
- Views must not import `next/*` or `react-router-dom`.
- `packages/core` must not import `packages/views`. Inject the price function if a shared helper needs it.
- Display dollars and gate dollars share one generated rate catalog. Custom browser rates stay display-only.
- Pause does not cancel running tasks. Budgets inherit that.
- Fail open on `Admit` store errors, the same way autopilot quota fails open on a missing entitlement.
- Waiver writes are workspace owner or admin only. `Decide` sees `Account.Waived`. Callers do not special-case a waiver.

## Alternatives

Three whole shapes were sketched in parallel. The cross-judge scored them against a six-point rubric.

| Shape | Verdict |
| --- | --- |
| **Ledger** (base) | Period account plus idempotent debits. Claim reads a balance. Attribution is snapshotted at task create. |
| **Sealed ledger** | Same idea with heavier types, enqueue+claim locks, and project budgets that pause the whole agent. Grafts taken. Base rejected. |
| **Observe overlay** | CRUD and client-priced bars in v1. `Admit` always proceeds. Honest about holes. Rejected as the v1 product because every configured action would be inert, and enforcement would add tables later instead of filling bodies. |

The ledger already contains an observe budget. Soften off plus `allow` draws a bar and never holds work.

**Why not query `task_usage_hourly` at claim.** The rollup lags about five minutes, has no squad dimension, and re-attributes when a project moves. Claim would become an analytic query on the hot path.

**Why not pause every agent on a project budget.** A project cap would stop unrelated work. Project and initiative `pause` holds covered tasks at claim. Agent and squad `pause` writes `paused_at`, because those scopes really cover everything the agent runs.

**Why a waiver table, not a flag on the budget row.** Teeth need a start and an end that are not the budget month. A waiver can cover a project that has no budget of its own so its tasks escape a parent initiative cap. Agent and squad accounts stay in `Decide`. A boolean on `budget` cannot express that.

**Why not waive agent teeth for that project's tasks.** Pause is agent-wide (`paused_at`). Letting waived-project work through would mean either unpausing the agent (unrelated work starts) or a claim exception that punches the pause SQL. v1 keeps pause as the hard stop. Owner or admin who needs that agent running resumes it.

## Applicable skills

- `how` over Analytics, `executionlane`, pause, and `task_usage` before you edit them
- `principle-model-the-domain` (`budgetpolicy.Decide` is the structure, not a chain of ifs in `task.go`)
- `principle-type-system-discipline` (unions for watch and exceed)
- `principle-boundary-discipline` (validate at HTTP, trust `Policy` inside)
- `principle-make-operations-idempotent` (debit upsert by `(budget_id, task_id, provider, model)`)
- `principle-migrate-callers-then-delete-legacy-apis` (delete the lanes flag in the same wave as its callers)
- `principle-sequence-verifiable-units` (one phase, one check)
- `interrogate` before the enforcement PR merges, if the pause split is still contested
- `unslop` on every prose surface, `/deslop` before each commit
- `control-ui` for Analytics verification (or `verify-multica` in this repo)

## Phases

1. [Lanes GA](phase-1-lanes-ga.md). Remove `agent_execution_lanes`.
2. [Pause service](phase-2-pause-service.md). One writer for `paused_at`.
3. [Rate catalog](phase-3-rate-catalog.md). One JSON source for Go and TypeScript.
4. [Attribution](phase-4-attribution.md). Snapshot project, initiative, and origin squad on the task.
5. [Domain and schema](phase-5-domain-and-schema.md). `budgetpolicy` plus tables.
6. [API and core](phase-6-api-and-core.md). HTTP, zod, React Query.
7. [Analytics UI](phase-7-analytics-ui.md). Budgets card on the Usage tab.
8. [Enforcement](phase-8-enforcement.md). `Admit`, `PostUsage`, `Soften`.
9. [Reconcile and docs](phase-9-reconcile-and-docs.md). Sweep, skills, product docs.

[testing.md](testing.md) lists the checks that every phase must run, plus the end-to-end box.

## Verification

Project-level commands after each phase that touches that layer:

```bash
# Go, from server/
go test ./internal/budgetpolicy ./internal/service ./internal/handler ./internal/executionlane -count=1

# TypeScript
pnpm exec vitest run packages/core/budgets packages/views/dashboard packages/views/runtimes
pnpm typecheck
```

Runtime. Drive `/{slug}/usage` with `verify-multica` or `control-ui`. Create a project budget, watch the bar, trip soften, trip pause, then waive that project and confirm the next task on it claims. Unit tests do not replace that.

## Implementation guidance

- Run `how` on each unfamiliar subsystem before you change it.
- Name the data shape first. `Decide` is a max over a small verdict order. Do not grow `if paused && percent && allow` in three files.
- Extract pause before budgets write `paused_at`. Do not copy the handler.
- Delete the lanes flag and its `enabled` parameters in one wave. Do not leave a dual path.
- `/deslop` each diff before commit. `unslop` every README, PR body, and commit message.
- Keep a decision trail in this folder if a phase changes the sketch. Edit [design.md](design.md), do not leave a second unofficial shape.
- After the first implementation PR opens, run Cursor babysit on that PR, not on this plan PR.

## Throughput checkpoint

**Blocking first steps.** Phases 1 through 4. Soften is a lie while the flag is off. Pause must have one service before budgets call it. Debits need a server price. Squad and initiative ledgers need a snapshot on the task.

**Independent workstreams.** After phase 4, work serializes on the budget types. Do not fan out two writers onto `budget` / `Admit` / the Analytics card. Prerequisite PRs 1 through 4 can stack. They do not share tables.

**Shared mutable state.** Only `AgentPauseService` writes pause columns. Only `BudgetService` writes budget tables and waivers. The rate JSON has one owner (the generator). `ReportTaskUsage` stays the usage writer. `PostUsage` runs after the upsert and must not become a second usage store.

**Smallest safe decomposition.** One owner from phase 5 onward. The schema, the API, the card, and admission share one type sketch. Splitting those across agents produces two money types.

## Open decisions for the human

These are product calls. The sketch picked a default so implementation can start. Change the default here before phase 5 if you disagree.

1. **Write ACL.** Default from existing manage rules. Project and initiative, any member. Agent, `canManageAgent`, no system agents. Squad, creator or admin. List, any member.
2. **UTC calendar month.** v1 does not follow the Analytics viewing timezone.
3. **Manual resume of a budget-paused agent.** Honored until the next `PostUsage` that still finds the account exhausted. The sweep never clears a human pause (`paused_by` set, `paused_by_budget_id` null).
4. **Unpriced lines on a `pause` budget.** Fail closed (refuse new covered work) and show a `pricing_incomplete` state. `allow` budgets keep running and show the unpriced count.
5. **Squad pause.** Pause the leader and every member agent on the squad. The covered set is the invocation roster, not "anyone who was ever a member."
6. **Waiver window.** Default `starts_at` is now. Default `ends_at` is the end of the current UTC month. Owner or admin may pick any future `ends_at`.
7. **Waiver depth.** A project waiver skips that project account and the parent initiative account for tasks stamped with that project. An initiative waiver skips the initiative account and every child project account for tasks stamped with that initiative. Agent and squad accounts still evaluate.
