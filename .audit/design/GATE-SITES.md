# Sites that must treat paused like "no new work"

Every `ArchivedAt.Valid` / `archived_at` check that refuses new work.

## Server

- `server/internal/service/agent_ready.go` AgentReadiness
- `server/internal/service/reason_code_test.go` TestAgentReadinessVerdict
- `server/internal/service/task.go` every `ArchivedAt.Valid` enqueue guard
- `server/internal/handler/issue.go` validateAssigneePair
- `server/internal/handler/chat.go` archived chat create / send
- `server/internal/handler/comment.go` mention triggers
- `server/internal/handler/autopilot.go` assignee archived
- `server/internal/handler/mika_onboarding.go`
- `server/internal/handler/quick_action.go` if it checks archive
- `server/internal/service/issue_trigger.go`
- `server/internal/service/autopilot.go` if it checks archive besides readiness
- Claim fences in `server/pkg/db/queries/agent.sql` (`ClaimAgentTask`, `ListQueuedClaimCandidatesByRuntime`, `ListQueuedClaimCandidatesByRuntimes`, reclaim queries, and every "Keep this authorization fence" copy).
- `server/internal/handler/issue_child_done.go`, `source_context.go`
- `packages/views/issues/blocked-trigger-copy.ts` (`agent_paused`)
- `packages/views/editor/extensions/slash-command-suggestion.tsx`
- `packages/views/modals/quick-create-issue.tsx`
- `packages/views/chat` archived banner / composer twin for paused
- `server/cmd/multica/cmd_agent.go` pause/resume next to archive/restore
- `server/cmd/server/router.go` POST pause / resume
- `server/pkg/protocol/events.go` agent:paused / agent:resumed
- `server/internal/dispatch/reason.go` ReasonAgentPaused
- `server/internal/handler/admission.go` fallback message

## Client

- `packages/core/types/agent.ts` paused_at, paused_by
- `packages/core/types/events.ts`
- `packages/core/api/client.ts` pauseAgent / resumeAgent
- `packages/core/agents/work-admission.ts` (new) + test
- `packages/core/agents/derive-presence.ts` + test
- `packages/core/agents/types.ts` AgentAvailability + "paused"
- `packages/views/agents/presence.ts`
- `packages/views/agents/components/agent-row-actions.tsx`
- `packages/views/agents/components/agent-detail-page.tsx`
- `packages/views/issues/components/pickers/assignee-picker.tsx`
- `packages/views/editor/extensions/mention-suggestion.tsx`
- `packages/views/autopilots/components/pickers/agent-picker.tsx`
- `packages/views/locales/{en,zh-Hans,ja,ko}/agents.json`
- `apps/mobile/data/schemas.ts` paused_at default null
- `apps/mobile` pickers and chat filters: import `agentAcceptsNewWork`
- `apps/mobile/data/realtime/use-presence-realtime.ts` subscribe paused/resumed
- `apps/mobile/components/ui/presence-dot.tsx` if it special-cases archived

## Skills

- `server/internal/service/builtin_skills/multica-mentioning/SKILL.md` and source map
- `server/internal/service/builtin_skills/multica-creating-agents/SKILL.md` and source map
- `server/internal/service/builtin_skills/multica-autopilots` if it names archive as the only stop
