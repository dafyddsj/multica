# How agent lifecycle works today

Explainer: parent Cursor Grok 4.6, grounded by four cursor-grok-4.6-high-fast explorers plus direct reads.

## Overview

An agent is a workspace teammate that owns issues, chats, and runs. Two lifecycle facts exist today: archive (`archived_at`) and runtime binding. Activity `status` (`idle|working|blocked|error|offline`) is not a lifecycle switch. Concurrency (`max_concurrent_tasks`, 1–50) is a scheduler cap, not an on/off switch.

Archive is the only way to stop an agent. It hides the row, cancels every active task, and fails new assignment, chat, and enqueue with archived errors. That is why pause is a request.

## Key concepts

- `agent.archived_at` / `archived_by`. Soft retire. List endpoints skip these rows unless `include_archived`.
- `AgentReadiness` in `server/internal/service/agent_ready.go`. Shared verdict for new work: available, waitable (offline runtime), or blocked (archived, no runtime, unusable CLI).
- `validateAssigneePair` in `server/internal/handler/issue.go`. Assignment rejects archived agents with `cannot assign to archived agent`.
- Chat create in `server/internal/handler/chat.go`. Rejects archived agents.
- Enqueue in `server/internal/service/task.go`. About ten `ArchivedAt.Valid` early returns.
- Claim SQL in `server/pkg/db/queries/agent.sql`. Does not check `archived_at`. Archive works because it cancels the queue first.
- `AgentAvailability` on the client. Runtime health plus an `archived` override in `deriveAgentPresenceDetail`.
- `max_concurrent_tasks`. Min 1 on both sides. Zero is illegal.

## How it works

A user archives from the list menu or the detail page. `POST /api/agents/:id/archive` sets timestamps, cancels queued/dispatched/running tasks, and publishes `agent:archived`. Restore clears the timestamps and publishes `agent:restored`.

New work has two layers. Interactive paths (assign, chat, mention) check `archived_at` and return a 400. Admission paths call `AgentReadiness` and treat archive as blocked. The daemon claim path only looks at capacity and the runtime fence. A paused agent that still has queued rows would keep running unless claim learns a new fence.

Presence is derived on the client. Archive short-circuits to `availability: archived` and idle workload, even if the runtime is still online.

## Where things live

- SQL: `server/pkg/db/queries/agent.sql`
- Handlers: `server/internal/handler/agent.go`, `issue.go`, `chat.go`, `comment.go`, `autopilot.go`
- Readiness: `server/internal/service/agent_ready.go`
- Enqueue/claim: `server/internal/service/task.go`
- Types: `packages/core/types/agent.ts`
- Presence: `packages/core/agents/derive-presence.ts`, `packages/views/agents/presence.ts`
- Actions: `packages/views/agents/components/agent-row-actions.tsx`, `agent-detail-page.tsx`
- Pickers: `packages/views/issues/components/pickers/assignee-picker.tsx`, mobile twins under `apps/mobile/components/issue/pickers/`
- CLI: `server/cmd/multica/cmd_agent.go`

## Gotchas

- Setting concurrency to 0 cannot pause an agent. Validation forbids it.
- Claim does not look at archive. Pause cannot copy archive blindly.
- Autopilot `paused` is a different entity. Do not reuse that column.
- System agents cannot be archived. Pause should match.
- Mobile has no archive action, but its pickers filter `archived_at`. Product semantics require the same for pause.
- `AgentReadiness` is the existing consolidation point. Direct `ArchivedAt` checks still exist at assignment and chat so the error string stays specific.
