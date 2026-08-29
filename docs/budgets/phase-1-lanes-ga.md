# Phase 1. Lanes GA

Back to [overview.md](overview.md).

## Goal

Lightweight and failover settings work for every workspace. Soften cannot hide behind a default-off flag.

## Changes

- Delete `agent_execution_lanes` from `server/internal/featureflags/keys.go` and `packages/core/feature-flags/keys.ts`.
- Drop the `enabled` argument from `executionlane.Initial`, `NextOnFailure`, and `applyClaimLaneSelection`.
- Show `ExecutionLanesFields` on create and inspector with no `useFeatureEnabled` gate.
- Persist lane fields on create and update even when they were previously ignored.
- Leave `CreateAutopilotTask` unstamped. `Decide` later downgrades budget soften for autopilot. Do not change autopilot start behavior in this phase.
- Audit rows that already have `lightweight_model` set. `start_lightweight` defaults true, so issue and chat tasks for those agents will start cheap after this ships. Put that in the release note.

## Data structures

No new types. `executionlane.AgentLanes` and `Selection` stay the organizing structure.

## Verification

**Static.** `go test ./internal/executionlane ./internal/handler ./internal/service -count=1` and the existing lane flag tests rewritten as always-on. `pnpm exec vitest run` on agent create/inspector tests that mocked the flag.

**Runtime.** Open agent create. The lightweight fields render without a flag file. Create an agent with a lightweight model and `start_lightweight` on. The next issue task claims that model.
