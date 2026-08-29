# Phase 8. Enforcement

Back to [overview.md](overview.md).

## Goal

The next claim honors the budget. A finished task posts a debit and may pause principal scopes.

## Changes

- `BudgetService.Admit` and `PostUsage` as in [design.md](design.md).
- Claim path in `handler/daemon.go` next to `applyClaimLaneSelection`. Hold skips the candidate. Soften calls `executionlane.Soften`.
- `ReportTaskUsage` calls `PostUsage` after the upsert loop and ignores budget errors.
- `executionlane.Soften`.
- `enforce` on agent and squad scopes through `AgentPauseService`. Resource waivers do not call resume.
- Create, update, and delete also call `enforce` so a lowered limit pauses without waiting for the next report.

## Data structures

No new public types. `Admission.Verdict` is the only value the claim path switches on.

## Verification

**Static.** Service tests for the composition example, delta adjust on re-report, fail-open Admit error, PostUsage swallow, principal pause, resource hold, Soften fallback when no lightweight model, project waiver punching through project and parent initiative, initiative waiver punching through child projects, waiver ignored on an agent account.

**Runtime.**

1. Agent budget, soften 80, allow. Drive spend past 80 percent. The next task claims the lightweight model.
2. Same budget, switch to pause, spend past the limit. The agent shows paused. New chat is refused. A running task finishes.
3. Project budget, pause. The agent's other project still claims. A task on the exhausted project stays queued.
4. Waive that project. The next task on it claims. A task on another exhausted project still holds. The bar still shows over.
5. Re-POST the same usage payload. Period spend does not double.
