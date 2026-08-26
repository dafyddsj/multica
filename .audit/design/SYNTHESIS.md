# Agent pause/resume design

Parent synthesis. Arena runners: claude-fable-5-thinking-xhigh, gpt-5.6-sol-xhigh, cursor-grok-4.6-high-fast, claude-opus-5-thinking-high. This file is the contract until a candidate forces a scrap.

## Problem

Archive is the only stop today. It hides the agent and cancels every task. Users need a temporary hold that keeps the agent visible, leaves in-flight work alone, and refuses new work with a paused error.

Concurrency 0 is illegal. `agent.status` is activity. Autopilot `paused` is a different row.

## Usage

```
POST /api/agents/:id/pause
POST /api/agents/:id/resume
multica agent pause <id>
multica agent resume <id>
```

Both are idempotent. Permission matches archive: `canManageAgent`. System agents refuse.

Call sites that already reject archive grow a paused branch with its own wording:

- `validateAssigneePair` → `cannot assign to paused agent`
- chat create → `agent is paused`
- `AgentReadiness` → blocked, `reason_code: agent_paused`
- mention / comment / autopilot admission use that readiness verdict
- Claim SQL adds `a.paused_at IS NULL` on every fence that already joins `agent`

Clients:

```
agentAcceptsNewWork(agent) // !archived_at && !paused_at
deriveAgentPresenceDetail  // archived wins, then paused, then runtime
```

Pickers filter with `agentAcceptsNewWork`. Presence shows Paused. Workload still reflects real running/queued counts so a paused agent that is finishing a run reads as Paused + Working.

## Shape

Lifecycle is two optional timestamps, same as archive.

```
agent.paused_at timestamptz null
agent.paused_by uuid null
```

Migration number is 439 (`439_agent_paused`). Numbers 432-438 already exist. No foreign key. No `paused_until`. Archive wins in derivation if both are set.

Wire fields on `Agent`: `paused_at`, `paused_by`, default null for old backends.

New dispatch code: `agent_paused`. Fallback English: `this agent is paused`. Clients switch with a default.

Events: `agent:paused` and `agent:resumed`, payload `{ agent }`. The existing `agent:` prefix already invalidates web lists. Mobile adds the two names next to archived/restored.

One Go helper beside archive checks:

```
func agentRefusesNewWork(a db.Agent) (blocked bool, archived bool, paused bool)
```

Handlers pick the error string from that. `AgentReadiness` uses it first, archived before paused.

Frontend pure helper in `packages/core/agents/work-admission.ts`:

```
export function agentAcceptsNewWork(agent: Pick<Agent, "archived_at" | "paused_at">): boolean
```

Mobile imports that function. Do not re-derive.

Public surface stays two POST verbs. Do not add `paused` to PUT /agents. That would mix a lifecycle command into a settings patch.

## Tradeoffs

- We accept a new column pair instead of reusing concurrency, so resume keeps the saved cap.
- We accept queued rows sitting until resume, instead of cancelling them. Cancel-all-tasks already exists.
- We accept a new reason code instead of collapsing into `target_unavailable`, so the UI can say paused.
- We accept no duration timer. "For a period of time" means temporary versus archive, not a clock.

## Alternatives rejected

- `max_concurrent_tasks = 0`. Illegal today, loses the cap, no distinct error.
- `agent.status = "paused"`. Collides with activity.
- Client-only badge. Claim would keep dispatching queued work.
- Timer / `paused_until`. Extra sweeper, not needed for the predicate.
- PATCH boolean. Archive already taught the product two verbs.

## First implementation step

Add the columns and a failing `AgentReadiness` case for a paused agent with a runtime.
