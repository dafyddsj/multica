# Budgets design

Synthesized from the architect arena. Base is candidate 1 (ledger). Grafts are named in [Synthesis decision](#synthesis-decision).

This file is the contract. Implementation fills `not implemented` bodies. If a phase needs a parameter this sketch does not have, stop and edit this file first.

## Problem

A budget has to turn delayed task usage into a decision at claim. The hourly rollup lags, has no squad key, and is priced in the browser. Pause and lanes already implement the two effects. Overlapping scopes are normal. The public surface must hide coverage, pricing, and composition.

## Usage (caller's view)

### Analytics

```tsx
import { useBudgets, useCreateBudget } from "@multica/core/budgets";

const { data } = useBudgets(wsId);
const create = useCreateBudget(wsId);

await create.mutateAsync({
  scope: "project",
  owner_id: projectId,
  limit_usd_ticks: 2_000_000_000_000, // $200
  soften_at_percent: 80,              // null turns soften off
  over_limit: "pause",
});
```

The card reads `current_period.spent_usd_ticks` and `current_period.state`. It does not call `estimateCost` for the bar.

### Waiver (owner or admin)

```tsx
import { useCreateBudgetWaiver } from "@multica/core/budgets";

const waive = useCreateBudgetWaiver(wsId);

await waive.mutateAsync({
  scope: "project",
  owner_id: projectId,
  starts_at: new Date().toISOString(), // default now
  ends_at: monthEndUtc.toISOString(),  // default end of current UTC month
  reason: "launch week",               // optional
});
```

The bar still shows spent versus limit. The card shows teeth waived until `ends_at`. Spend keeps posting. Soften and hold from the waived resource scopes do not apply until the window ends.

### Claim

```go
adm, err := h.Budgets.Admit(ctx, task)
if err != nil {
    adm = budgetpolicy.Admission{Verdict: budgetpolicy.VerdictProceed} // fail open
}
switch adm.Verdict {
case budgetpolicy.VerdictHold:
    // leave queued, try the next candidate
case budgetpolicy.VerdictSoften:
    claimSel = executionlane.Soften(lanes, claimSel)
}
```

### Usage report

```go
h.Budgets.PostUsage(ctx, task, lines) // after UpsertTaskUsage; never fails the report
```

### Composition

**Strictest covering verdict wins.** Order is `proceed < soften < hold`.

Example. Agent A, project P, initiative I.

| Budget | Spent | Limit | Soften | Over | Local |
| --- | --- | --- | --- | --- | --- |
| Agent A | $50 | $100 | 80 | allow | proceed |
| Project P | $170 | $200 | 80 | pause | soften |
| Initiative I | $1010 | $1000 | none | allow | proceed |

Composed result is `soften`. Flip I to `pause` and the result is `hold` held by I.

Add an active project waiver on P. Resource accounts that the waiver covers (P, and I for tasks stamped with P) contribute `proceed`. Agent A is unchanged. Composed result is `proceed`. Flip A's over-limit to `pause` with A exhausted and the result is still a principal pause. The waiver does not clear `paused_at`.

Agent and squad `pause` writes `paused_at` at debit time. Project and initiative `pause` holds the task at claim. Both mean no new covered work starts. A waiver only turns off the resource-scope teeth.

## Shape

Organizing structure. A ledger plus a pure verdict function.

### Tables

```
budget
  id, workspace_id, scope, owner_id,
  period ('monthly'),
  limit_usd_ticks CHECK > 0,
  soften_at_percent NULL or 1..100,
  over_limit ('pause' | 'allow'),
  created_by, created_at, updated_at
  UNIQUE (workspace_id, scope, owner_id)

budget_period
  id, budget_id, workspace_id,
  period_start, period_end,
  spent_usd_ticks >= 0,
  unpriced_line_count >= 0
  UNIQUE (budget_id, period_start)

budget_debit
  id, workspace_id, budget_id, budget_period_id,
  task_id, provider, model,
  amount_usd_ticks >= 0,
  priced_by ('provider' | 'rate_table' | 'unpriced')
  UNIQUE (budget_id, task_id, provider, model)

budget_waiver
  id, workspace_id,
  scope CHECK IN ('project', 'initiative'),
  owner_id,
  starts_at, ends_at CHECK (ends_at > starts_at),
  created_by, reason NULL, created_at

agent
  + paused_by_budget_id   -- budget pause. Human pause keeps paused_by.
```

No foreign keys. Each unique or lookup index is its own `CREATE [UNIQUE] INDEX CONCURRENTLY` file.

### Illegal states

| Illegal | Closed by |
| --- | --- |
| Threshold with no cheap-lane action | `soften_at_percent` is the action. NULL is off |
| Pause and allow both set | one required enum |
| Limit <= 0 | CHECK + zod `.positive()` |
| Two budgets on one entity | unique `(workspace, scope, owner)` |
| Double-count a re-report | unique debit key, adjust by delta |
| Sweep resumes a human pause | resume only where `paused_by_budget_id` is set |
| Squad bar that looks like $0 | `state` includes `unattributed` and `pricing_incomplete` |
| Waiver on agent or squad | `scope` CHECK allows only `project` and `initiative` |
| Empty or inverted window | `ends_at > starts_at`, constructors reject the rest |
| Two live waivers on one owner | service refuses overlap. History rows may sit in the past |
| Member-created waiver | handler, owner or admin only |

### Pure core (`server/internal/budgetpolicy`)

```go
type Verdict int // proceed < soften < hold

type Account struct {
    Budget     BudgetRef
    LimitTicks int64
    SpentTicks int64
    Unpriced   int
    SoftenAt   *int16
    OverLimit  OverLimit
    Waived     bool // resource-scope teeth off. Principal accounts never set this.
}

func Decide(accounts []Account, forAutopilot bool) Admission
func StateOf(a Account) AccountState // ok | softened | exhausted | pricing_incomplete | unattributed | waived
func WaiverCovers(w Waiver, account BudgetRef, task TaskRef) bool
```

`Decide` uses max over local results. A waived account contributes `proceed` and nothing else. An account with `Unpriced > 0` and `over_limit=pause` contributes `hold`. The same account with `allow` contributes `soften` if the threshold is crossed on known spend, else `proceed`. Autopilot downgrades `soften` to `proceed`.

`WaiverCovers` is the depth rule. A project waiver matches that project account, and the parent initiative account when the task's project stamp is the waived project. An initiative waiver matches that initiative account, and every project account whose task initiative stamp is the waived initiative. Agent and squad refs never match.

### Service (`BudgetService`)

```go
Admit(ctx, task) (Admission, error)      // read periods, active waivers, Decide
PostUsage(ctx, task, lines)              // price, debit, enforce principals
Create/Update/Delete/List...
CreateWaiver/DeleteWaiver/ListWaivers...
Reconcile(ctx) error                     // resume budget pauses that no longer apply
```

`Admit` resolves coverage from the task snapshot, loads waivers whose window contains now, sets `Account.Waived` through `WaiverCovers`, then calls `Decide`. Missing period means spent 0.

`PostUsage` still posts debits on waived scopes. The bar stays honest. `enforce` on agent and squad ignores resource waivers. A waived project does not resume a budget-paused agent.

`PostUsage` prices each line (authoritative ticks if > 0, else the catalog, else unpriced). It upserts the debit, moves the period by the delta, then `enforce()` on agent and squad scopes.

### Pause

```go
type PauseActor struct { UserID, BudgetID pgtype.UUID } // constructors set exactly one

Pause(ctx, agent, by PauseActor) error
Resume(ctx, agent) error
ResumeBudgetPaused(ctx, budgetID) error
```

Handler keeps authz. Service owns the write and the event.

### Lanes

```go
func Soften(lanes AgentLanes, stamped Selection) Selection
```

Returns the lightweight selection when `LightweightModel != ""`. Otherwise returns `stamped`. Budget code never builds a `Selection` by hand.

### HTTP

```
GET    /api/budgets
POST   /api/budgets
PATCH  /api/budgets/{id}
DELETE /api/budgets/{id}

GET    /api/budgets/waivers
POST   /api/budgets/waivers
DELETE /api/budgets/waivers/{id}
```

Budget writes follow the ACL in [overview.md](overview.md). Waiver writes are workspace owner or admin only. Members may list waivers so the card can render the chip. `budget:updated` invalidates `["budgets", wsId]` and `["budgets", wsId, "waivers"]`.

### TypeScript

```ts
type BudgetScope = "agent" | "squad" | "project" | "initiative";
type BudgetOverLimit = "pause" | "allow";
type AccountState =
  | "ok"
  | "softened"
  | "exhausted"
  | "pricing_incomplete"
  | "unattributed"
  | "waived";

type WaiverScope = "project" | "initiative";

interface BudgetWaiver {
  id: string;
  scope: WaiverScope;
  owner_id: string;
  starts_at: string;
  ends_at: string;
  created_by: string;
  reason: string | null;
}

interface Budget {
  id: string;
  scope: BudgetScope;
  owner_id: string;
  limit_usd_ticks: number;
  soften_at_percent: number | null;
  over_limit: BudgetOverLimit;
  current_period: {
    period_start: string;
    period_end: string;
    spent_usd_ticks: number;
    unpriced_line_count: number;
    state: string; // include a default branch in every switch
  } | null;
}
```

Form state uses `{ kind: "off" } | { kind: "at"; percent: number }` and collapses to the nullable wire field. Waiver form is a window, not a boolean. `{ kind: "inactive" } | { kind: "active"; starts_at: string; ends_at: string; reason: string | null }`.

## Modules

| Module | Owns |
| --- | --- |
| `server/internal/budgetpolicy` | `Decide`, `StateOf`, `WaiverCovers`, month window |
| `server/internal/service/budget.go` | ledger, CRUD, `Admit`, `PostUsage`, `Reconcile` |
| `server/internal/service/agent_pause.go` | the only `paused_at` writer |
| `server/internal/executionlane` | `Soften` |
| `server/internal/costrate` | generated `PriceTicks` |
| `server/internal/handler/budget.go` | HTTP, ACL, owner exists |
| `packages/core/budgets` | schemas, hooks |
| `packages/views/dashboard` | card and dialog |

`TaskService` stamps attribution. It does not price or compose.

## Synthesis decision

Base is candidate 1.

**Kept from 1.** Three public calls (`Admit`, `PostUsage`, CRUD). Claim-time hold for project and initiative. Agent and squad pause through the extracted service. Soften as one lane helper. Fail open on `Admit` errors. Flag removal before soften ships. UTC month. Ticks as JSON numbers.

**Grafted from 2.** Closed `Priced | Unpriced` on each debit. `unpriced_line_count` on the period. Fail closed when a `pause` account has unpriced lines. Origin-squad attribution, membership does not count. Snapshot project and initiative at task create. Lattice tests on `Decide`.

**Grafted from 3.** `unattributed` state so a squad without a stamp cannot render a $0 bar. Per-scope write ACL. Price function stays out of `packages/core` for any client helper that still needs `estimateCost` (explorer charts, not budget bars).

**Rejected from 2.** Sealed interface ceremony and branded bigint on the wire. Global agent pause for a project budget. `DecideTask` inserting and locking inside enqueue. Rewriting `ReportTaskUsage` into a new `taskusage` service in the same wave. Fail-closed `Admit` on a store error.

**Rejected from 3.** Observe-only v1 as the product. Keeping `agent_execution_lanes`. Client-priced bars as the enforcement number. A module `Phase` constant that makes `Admit` a no-op.

## Tradeoffs accepted

- We accept a second book of spend (ledger beside `task_usage`) so claim is a handful of point reads and attribution stays stable.
- We accept that the explorer chart (live attribution, current rates) and the budget bar (frozen at post) can differ by a few dollars.
- We accept overshoot by in-flight tasks. Copy must say the cap applies to new tasks.
- We accept that a squad budget pauses member agents' other work.
- We accept UTC months in v1.
- We accept that a project waiver does not unpause an agent. Pause stays the workspace-wide stop. Owner or admin resume the agent if that project's work must run on a budget-paused agent.
- We accept overlapping waiver history as many rows, with the service refusing a second live window on the same owner.

## Next implementation step

Phase 1. Delete `agent_execution_lanes` and drop every `enabled` parameter on the lane helpers.
