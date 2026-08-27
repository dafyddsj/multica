# Phase 4. Factory and whitelist

Back to [overview](overview.md).

## Goal

`New("amp")` constructs `ampBackend`. The protocol-family lockstep accepts `amp` and rejects unknown keys the same way as today.

## Changes

This phase is one invariant, so it touches more than two files on purpose.

- `server/pkg/agent/agent.go`: add `"amp"` to `SupportedTypes`, the `New` switch, and the launch-header map (`"amp (stream-json)"`).
- `server/pkg/agent/agent_supported_types_test.go`: expected whitelist includes `amp`.
- New migration pair after the current head (main has `458` from the clerk series; rebase and take the next free number) widening `runtime_profile_protocol_family_check` with `NOT VALID`. Follow the body of `server/migrations/403_runtime_profile_add_zeroclaw.up.sql`.
- `packages/core/types/agent.ts`: add `"amp"` to `RUNTIME_PROFILE_PROTOCOL_FAMILIES`.

`ListModels` needs an `"amp"` case. The switch `default` errors. Return an empty catalog. `ModelSelectionSupported("amp")` is false. Amp's product default is a mode dial, documented as an SDK extra, not a verified CLI `--model`. The UI then shows "Managed by runtime" instead of a picker that drops values.

`TestLaunchHeaderCoversAllSupportedBackends` fails if `launchHeaders` omits `amp`.

## Data structures

`SupportedTypes` remains the single family whitelist. `amp` is a family, not a `BuiltinRuntime`.

## Verification

**Static.** `cd server && go test ./pkg/agent -run 'TestSupportedTypes' -count=1`

**Runtime.** None. Migration is schema only.
