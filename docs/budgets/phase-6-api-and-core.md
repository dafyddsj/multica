# Phase 6. API and core

Back to [overview.md](overview.md).

## Goal

A workspace can create, list, update, and delete budgets. The response includes the current period after a create-time backfill.

## Changes

- `BudgetService` CRUD. Create backfills opening debits for the current UTC month from `task_usage` joined through the task snapshot, priced with `costrate`. Squad without origin stamps returns `state=unattributed` and spent 0.
- `handler/budget.go` plus routes under the workspace member group. ACL from [overview.md](overview.md). Owner must exist in the workspace.
- `packages/core/api` methods, zod schemas, `parseWithFallback`, malformed-response tests.
- `packages/core/budgets` hooks and `["budgets", wsId]` keys. No Zustand store.
- `budget:updated` on mutate. Realtime invalidates that key only.

## Data structures

Wire `Budget` in [design.md](design.md). Form union stays in views in the next phase.

## Verification

**Static.** Handler tests via `testutil.Call`. Core schema tests for missing `current_period`, unknown `over_limit`, non-positive limit, extra fields.

**Runtime.** `curl` create and list against a local API. Confirm ticks and state. Confirm a second POST for the same project updates, it does not insert.
