# Phase 7. Frontend lockstep

Back to [overview](overview.md).

## Goal

The family picker, logo, and MCP tab stay honest. Devin appears as a protocol family. The MCP tab stays hidden until the isolated-config canary.

## Changes

- `packages/core/types/agent.ts` already gained `"devin"` in phase 4. Confirm the derived union compiles.
- `packages/views/runtimes/components/provider-logo.tsx` and `provider-logo.test.tsx`. Add a Devin mark.
- `packages/core/agents/mcp-support.ts`. Do not add `"devin"` yet.

There is no `control-ui` skill in this checkout. Say so in the implementer PR if the picker was not clicked.

## Data structures

No new types. `RuntimeProtocolFamily` stays derived from the tuple.

## Verification

**Static.** `pnpm --filter @multica/core test -- runtimes display mcp-support` and `pnpm --filter @multica/views test -- provider-logo`

**Runtime.** Flag missing UI drive in the PR.
