# Phase 5. Daemon probe

Back to [overview](overview.md).

## Goal

A machine with `devin` on PATH, or `MULTICA_DEVIN_PATH` set, registers a Devin runtime. `MULTICA_DEVIN_ARGS` reaches `ExecOptions.ExtraArgs` and then argv.

## Changes

- `server/internal/daemon/agents_probe.go`. `probe("MULTICA_DEVIN_PATH", "devin", "")`. Do not add `MULTICA_DEVIN_MODEL` until `ListModels` is real.
- `server/internal/daemon/config.go`. Add `"devin"` to `defaultAgentCommandNames`. Add `shellArgsFromEnv("MULTICA_DEVIN_ARGS")` and forward into ExtraArgs.
- `scripts/agent-cli-command-names.txt`. Add `devin` in sorted position.
- `server/internal/metrics/labels.go` `knownRuntimeProviders`. Add `"devin"`.
- ExtraArgs wiring test on `devinBackend`.

## Data structures

`AgentEntry{Path, Command: "devin"}` from the existing probe helper.

## Verification

**Static.** `cd server && go test ./internal/daemon -run 'CommandNames|Probe' -count=1` and `go test ./pkg/agent -run 'TestDevinExtraArgs' -count=1`

**Runtime.** `command -v devin` on a machine that has Devin, then confirm the daemon log line that registered provider `devin`. Default CI stays on the fake path.
