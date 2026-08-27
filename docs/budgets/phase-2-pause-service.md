# Phase 2. Pause service

Back to [overview.md](overview.md).

## Goal

One function writes `paused_at`. The HTTP handler and a later budget enforcer call the same path.

## Changes

- Move the write, COALESCE semantics, system-agent refusal, and `agent:paused` / `agent:resumed` publish from `Handler.PauseAgent` / `ResumeAgent` into `server/internal/service/agent_pause.go`.
- Handler keeps `canManageAgent`, archive checks, and status codes.
- Add `paused_by_budget_id` on `agent` (nullable UUID, no FK) and a `PauseActor` with constructors `PausedByUser` and `PausedByBudget`. Exactly one side is set.
- A second pause does not overwrite the first actor (today's COALESCE).
- `ResumeBudgetPaused` clears only rows with that `paused_by_budget_id`. Do not call it from the HTTP resume. HTTP resume still clears any pause the user is allowed to clear.

## Data structures

`PauseActor` is a sum type encoded as two UUID fields plus constructors. Do not add a second paused flag.

## Verification

**Static.** Move `TestPauseAgentDoesNotCancelTasks` and the idempotent pause tests onto the service. Handler tests still cover 403/409.

**Runtime.** Pause and resume from the agent menu. Presence shows paused. Queued work sits. Running work finishes. Same as today.
