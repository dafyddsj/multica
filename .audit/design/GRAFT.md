# Grafted contract deltas

Parent synthesis in `SYNTHESIS.md` plus these overrides after all four
arena packages landed.

## Reclaim is fenced

Candidate 3 said reclaim is in-flight recovery. Candidate 4 showed the
row is still `dispatched` with `started_at IS NULL`, so reclaim is a
new start. Implementer already added `a.paused_at IS NULL` on every
query that carries `Keep this authorization fence in sync with
ClaimAgentTask`, including both reclaim queries. Keep that.

Do not extract `agent_claim_binding_ok(...)` in this PR.

## Presence

`deriveAgentPresenceDetail`:

1. `archived_at` set → availability `archived`, workload idle, counts 0.
2. `paused_at` set → availability `paused`, real workload and counts.
3. else runtime + tasks as today.

`paused` is a lifecycle override on the existing availability union so
`availabilityConfig` and mobile `PresenceDot` keep one switch. It is
not a runtime color and it is not in `availabilityOrder`.

## Picker

Show paused. Disable them. Tooltip uses a paused string, not archived
and not "needs a runtime".

## Pause SQL

`PauseAgent` should use `COALESCE(paused_at, now())` so a second pause
does not reset "paused 2 hours ago". Review if the implementer shipped
`paused_at = now()`.

## Helper

`packages/core/agents/work-admission.ts` is the only client read of
`paused_at` for admission. Mobile imports that function.
