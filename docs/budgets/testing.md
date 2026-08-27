# Budgets testing

Back to [overview.md](overview.md).

Every phase lists its own static and runtime check. This file is the shared bar.

## Static

- Go tests go through `testutil.Call` and `dbfx` fixtures. Do not open-code `httptest` recorders or raw `INSERT ... RETURNING` in new tests.
- `budgetpolicy` tests are a `.go` table next to `Decide`. They are the canonical composition matrix. Handler tests do not repeat that matrix in HTTP.
- TypeScript suites that need no DOM start with `// @vitest-environment node`.
- Views tests mock `@multica/core` with the callable Zustand shape only when they already mock a store. Budgets are React Query.
- New API methods get a malformed-response test.

## Runtime

The matching surface is the Analytics Usage tab on web.

Use the `verify-multica` skill in this repo when the cloud box is up. Otherwise use `control-ui` if the plugin is present. If neither is available, say so in the PR and keep the `curl` plus handler proof.

Walk this path before you call enforcement done:

1. Sign in. Open `/{slug}/usage`.
2. Create an agent budget with soften 80 and allow.
3. Run work until the bar crosses 80 percent. Confirm the next task used the lightweight model.
4. Switch the same budget to pause and cross 100 percent. Confirm the agent is paused in the list and chat is blocked.
5. Resume from the menu. Confirm work runs (softened) until the next report re-pauses.
6. Create a project budget with pause. Confirm a task on that project holds and a task on another project for the same agent starts.
7. Confirm a squad without origin stamps shows unattributed, not $0.

## What this does not prove

- Mid-run spend. There is none in the DB until `ReportTaskUsage`.
- Mobile. Out of v1.
- Desktop pixel parity. Shared `DashboardPage` is enough unless you change platform chrome.
