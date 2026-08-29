# Phase 7. Frontend lockstep

Back to [overview](overview.md).

## Goal

The create-agent and runtime-picker pages treat `amp` as a first-class family. The MCP tab stays hidden until phase 6's file-path canary passes.

## Changes

- `packages/core/agents/mcp-support.ts` and `mcp-support.test.ts`: add `"amp"` only after the backend actually forwards `--mcp-config` and a real CLI accepts that form.
- `packages/views/runtimes/components/provider-logo.tsx` and its test: an Amp mark. Reuse a simple letter mark if no official SVG is licensed for the repo. Do not hotlink ampcode.com.
- `packages/core/runtimes/display.ts`: skip unless product copy wants a name other than `Amp` (first-letter capital of `amp` already matches).

No `packages/core` `process.env`. No stores in `packages/views`.

## Data structures

`RUNTIME_PROFILE_PROTOCOL_FAMILIES` already gained `"amp"` in phase 4.

## Verification

**Static.** `pnpm --filter @multica/core test -- mcp-support` and `pnpm --filter @multica/views test -- provider-logo`

**Runtime.** This checkout has no `control-ui` skill. Open the create-agent runtime picker on web and confirm Amp appears once a daemon with Amp is registered. Note the gap in the PR if you cannot run the app.
