# Phase 4. Factory and whitelist

Back to [overview](overview.md).

## Goal

`New("devin")` constructs `devinBackend`. The protocol-family lockstep accepts `devin` and rejects unknown keys the same way as today.

## Changes

This phase is one invariant, so it touches more than two files on purpose.

- `server/pkg/agent/agent.go`. Add `"devin"` to `SupportedTypes`, the `New` switch, and `launchHeaders`.
- `server/pkg/agent/agent_supported_types_test.go`. Expected whitelist includes `devin`.
- New migration pair after the current head, widening `runtime_profile_protocol_family_check` with `NOT VALID`. Follow `server/migrations/403_runtime_profile_add_zeroclaw.up.sql`.
- `packages/core/types/agent.ts`. Add `"devin"` to `RUNTIME_PROFILE_PROTOCOL_FAMILIES`.

`ListModels` needs a `"devin"` case. Return an empty catalog. `ModelSelectionSupported("devin")` is false until `devin models list --format json` is captured from a logged-in CLI.

`TestLaunchHeaderCoversAllSupportedBackends` fails if `launchHeaders` omits `devin`.

## Data structures

`SupportedTypes` remains the single family whitelist. `devin` is a family, not a `BuiltinRuntime`.

## Verification

**Static.** `cd server && go test ./pkg/agent -run 'TestSupportedTypes' -count=1`

**Runtime.** None. Migration is schema only.
