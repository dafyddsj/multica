# Phase 7. Analytics UI

Back to [overview.md](overview.md).

## Goal

The Usage tab of `/{slug}/usage` can create and edit a budget and show a server-priced bar.

## Changes

- `packages/views/dashboard/components/budgets-card.tsx` and `budget-form-dialog.tsx`.
- Mount on `dashboard-page.tsx` Usage tab. Do not add a third tab.
- Scope picker reuses the initiative, project, agent, and squad lists already loaded for filters.
- Soften control is a discriminated union in form state, default `{ kind: "at", percent: 80 }`.
- Bar reads `current_period`. `unattributed` and `pricing_incomplete` have explicit copy. Do not draw a $0 bar for those.
- Copy states that the cap applies to new tasks.
- Owner and admin see Bypass teeth on project and initiative rows. Window defaults to now through end of the current UTC month. Optional reason. The chip reads teeth waived until {date}. Agent and squad rows have no waiver control.
- i18n keys in `packages/views/locales/*/usage.json` (and the other locale files). Read conventions before Chinese copy.
- Web and desktop already share `DashboardPage`. No new routes.

## Data structures

`SoftenChoice = { kind: "off" } | { kind: "at"; percent: number }`. Collapse on submit.

`WaiverChoice = { kind: "inactive" } | { kind: "active"; starts_at: string; ends_at: string; reason: string | null }`.

## Verification

**Static.** Views tests for the form variants, default 80, unattributed empty state, exhausted chip, allow-over copy, waived chip, hidden waiver control for a member and for an agent row. Mock `@multica/core` with the Zustand-callable shape only where a store is already mocked. Do not remount the helper matrix.

**Runtime.** In the browser, create a $50 project budget with soften at 80 and pause. The card shows the backfilled bar. Edit the limit. As owner, waive the project for the rest of the month. Delete the budget. Check desktop only if you have it. Mobile is out.
