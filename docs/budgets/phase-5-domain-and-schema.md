# Phase 5. Domain and schema

Back to [overview.md](overview.md). Sketch in [design.md](design.md).

## Goal

`Decide` is table-tested and the tables exist. No HTTP yet.

## Changes

- Add `server/internal/budgetpolicy` with `Decide`, `StateOf`, and `MonthWindow`.
- Cover the composition example and its pause/allow flip. Cover lattice associativity. Cover unpriced+pause, unpriced+allow, autopilot soften downgrade, principal versus resource pause.
- Migrations for `budget`, `budget_period`, `budget_debit`, and `agent.paused_by_budget_id`. Each concurrent index in its own file.
- sqlc queries named in the sketch. Do not wire handlers.

## Data structures

`Account`, `Verdict`, `Admission`. Period rows lazily created later. This phase only needs the types and empty tables.

## Verification

**Static.** `go test ./internal/budgetpolicy -count=1`. `make sqlc` succeeds. A dry migrate on a worktree database applies every concurrent index.

**Runtime.** Not yet. No user surface.
