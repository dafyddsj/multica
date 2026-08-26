# Agent pause/resume: Phase A frame

Parent model: Cursor Grok 4.6
How explorers: cursor-grok-4.6-high-fast

## Predicate (done when all are true)

A workspace member who can manage an agent can pause it and resume it without archiving it.

1. `POST /api/agents/:id/pause` sets `paused_at` on a non-archived user agent. A second pause is idempotent (200, still paused).
2. `POST /api/agents/:id/resume` clears `paused_at`. A second resume is idempotent (200, still active).
3. A paused agent stays on `GET /api/agents`. It is not in the archived list.
4. Assignment, new chat, mention enqueue, and autopilot assignee checks reject a paused agent with a distinct paused error, not an archived error.
5. `ClaimAgentTask` and sibling claim queries skip paused agents. Queued rows stay queued.
6. Pause does not cancel running or queued tasks. Archive still cancels them.
7. System agents cannot be paused (same product rule as archive).
8. The agents list and agent detail show Pause and Resume. Presence reads as Paused, not Archived or Offline.
9. The assignee picker does not offer a paused agent as a live target.
10. Targeted Go and Vitest suites pass. A browser walkthrough records pause, a rejected assign, resume, and a successful assign.

## Scope

Cross-cutting, one product behavior.

- Backend: agent columns, pause/resume SQL and handlers, enqueue/claim/assign/chat/autopilot gates, CLI, tests.
- Shared client: Agent type, zod schema, API client, events, presence union.
- Views: row actions, detail, presence, picker, i18n (en plus other locales).
- Skills that tell agents archive is the only way to stop work.

Rough size: one migration, about a dozen Go gate sites that already check `archived_at`, matching frontend presence/picker/actions, CLI twins of archive/restore.

## Rigor

High. The data shape is a one-way door. The existing suggestion (set `max_concurrent_tasks` to 0) is already illegal (`min = 1` in `server/internal/agentconfig/concurrency.go` and `packages/core/agents/constants.ts`). Archive is the wrong verb: it hides the agent and cancels every task.

## Grounded facts

- Agent activity `status` is `idle | working | blocked | error | offline`. Not a lifecycle flag.
- Archive is `archived_at` / `archived_by`. Archive cancels tasks, then broadcasts `agent:archived`.
- Claim SQL does not check `archived_at`. Archive works because it empties the queue first.
- Presence already has a lifecycle override: `archived` wins over runtime health in `deriveAgentPresenceDetail`.
- Assignment already rejects archived agents in `validateAssigneePair`.
- Chat create already rejects archived agents.
- Autopilot already has its own `paused` status. That is a different entity.

## Product choices locked for this run

These are reversible in review. They are not questions to wait on.

- Pause is a first-class timestamp (`paused_at`, `paused_by`), not concurrency 0 and not a new `agent.status` value.
- Pause is indefinite. Resume is explicit. No `paused_until` sweeper in this PR.
- Pause does not cancel in-flight work. Cancel-all-tasks already exists.
- Pause keeps the agent visible and editable.
- Archive still wins if both timestamps are set.

## Open only if implementation friction appears

- Whether a paused agent may stay assigned to existing issues (yes: the assignee stays; new runs refuse).
- Whether mobile needs a pause control in this PR (yes for types/semantics if mobile shows agents; UI only if an existing mobile archive action exists).
