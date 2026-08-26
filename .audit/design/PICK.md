# Arena pick

Models: claude-fable-5-thinking-xhigh (1), gpt-5.6-sol-xhigh (2),
cursor-grok-4.6-high-fast (3), claude-opus-5-thinking-high (4).

Base: candidate 3. Pause is an admission fence, not a second archive.
Lifecycle stays off `AgentAvailability`'s runtime colors in spirit, even
though the wire still hangs `paused` next to `archived` so every existing
dot/label switch keeps working.

Grafts from candidate 4:

- Fence reclaim. A stale dispatched row has `started_at IS NULL`.
  Redelivering it starts work. The claim-family comment already lists
  those queries as copies of `ClaimAgentTask`.
- One client helper (`agentLifecycle` / `agentAcceptsNewWork`). Callers
  do not re-read timestamps.
- Assignee picker shows paused agents and drops them from
  `runnableAgentIds`, same as a runtime-unbound agent.
- Pause keeps real running/queued counts. Archive still zeroes them.

Rejected from candidate 4: `agent_claim_binding_ok(...)` SQL function.
The repo already has a sync comment. A function is a later cleanup, not
this PR.

Rejected from candidate 1: stuffing pause into availability without
keeping workload.

Rejected from candidate 2: a lock-backed claim fence and an
`AgentNewWorkBlock` type that still leaves each caller inventing English.
