# Phase 9. Reconcile and docs

Back to [overview.md](overview.md).

## Goal

Crash windows and month rollover converge. Humans can read how budgets work.

## Changes

- Scheduler job calls `BudgetService.Reconcile` on the rollup cadence. Resume agents whose `paused_by_budget_id` account is no longer exhausted. Re-apply pauses lost between commit and `enforce`.
- Product docs under `apps/docs/content/docs/` for budgets. Link from Analytics.
- Builtin skill touch if agents are told they can set or respect a budget. Update `SKILL.md` and the matching `references/*-source-map.md` in the same change.
- Workspace delete walks `budget`, `budget_period`, `budget_debit`, and `budget_waiver` in the same explicit cleanup that already deletes usage.
- Initiative, project, squad archive/delete, and agent archive delete the matching budget and any waiver on that owner in the same application transaction.
- Reconcile does not invent waivers and does not end them early. An expired window drops out of `Admit` because now is outside `[starts_at, ends_at)`.

## Data structures

None. Reconcile is idempotent over existing rows.

## Verification

**Static.** Reconcile tests. Month boundary with a frozen clock. Human pause left untouched. Delete-cleanup tests for each owner.

**Runtime.** Pause via budget, raise the limit, wait for the job (or invoke it). The agent resumes. Pause via the menu, raise a covering limit. The agent stays paused.
