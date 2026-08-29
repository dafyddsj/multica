# Phase 6. API and core

Back to [overview.md](overview.md).

## Goal

A workspace can create, list, update, and delete budgets. The response includes the current period after a create-time backfill.

## Changes

- `BudgetService` CRUD. Create backfills opening debits for the current UTC month from `task_usage` joined through the task snapshot, priced with `costrate`. Squad without origin stamps returns `state=unattributed` and spent 0.
- Waiver CRUD on the same service. `CreateWaiver` rejects `scope` other than project or initiative, inverted windows, and an overlapping live waiver on the same owner. A waiver may exist with no budget on that owner (carve-out from a parent initiative).
- `handler/budget.go` plus routes under the workspace member group. Budget ACL from [overview.md](overview.md). Waiver writes are owner or admin only. Owner must exist in the workspace.
- `packages/core/api` methods, zod schemas, `parseWithFallback`, malformed-response tests for budgets and waivers.
- `packages/core/budgets` hooks and keys `["budgets", wsId]` plus `["budgets", wsId, "waivers"]`. No Zustand store.
- `budget:updated` on mutate. Realtime invalidates both keys.

## Data structures

Wire `Budget` and `BudgetWaiver` in [design.md](design.md). Form unions stay in views in the next phase.

## Verification

**Static.** Handler tests via `testutil.Call`. Core schema tests for missing `current_period`, unknown `over_limit`, non-positive limit, extra fields, waiver on `agent`, inverted window, member 403 on waiver POST.

**Runtime.** `curl` create and list against a local API. Confirm ticks and state. Confirm a second POST for the same project updates, it does not insert. Confirm an owner can POST a project waiver and a member cannot.
