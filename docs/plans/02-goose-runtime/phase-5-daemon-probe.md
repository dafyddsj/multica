# Phase 5. Daemon probe

Back to [overview](overview.md).

## Goal

A machine with `goose` on PATH, or `MULTICA_GOOSE_PATH` set, registers a Goose runtime. `MULTICA_GOOSE_ARGS` reaches `ExecOptions.ExtraArgs` and then argv.

## Changes

- `server/internal/daemon/agents_probe.go`. `probe("MULTICA_GOOSE_PATH", "goose", "")`. Do not add `MULTICA_GOOSE_MODEL` until `ListModels` is real.
- `server/internal/daemon/config.go`. Add `"goose"` to `defaultAgentCommandNames`. Add `shellArgsFromEnv("MULTICA_GOOSE_ARGS")` next to the existing `MULTICA_*_ARGS` cases, and forward into ExtraArgs.
- `scripts/agent-cli-command-names.txt`. Add `goose` in sorted position.
- `server/internal/metrics/labels.go` `knownRuntimeProviders`. Add `"goose"`.
- ExtraArgs wiring test on `gooseBackend`.

## Data structures

`AgentEntry{Path, Command: "goose"}` from the existing probe helper.

## Verification

**Static.** `cd server && go test ./internal/daemon -run 'CommandNames|Probe' -count=1` and `go test ./pkg/agent -run 'TestGooseExtraArgs' -count=1`

**Runtime.** `command -v goose` on a machine that has Goose, then confirm the daemon log line that registered provider `goose`. Default CI stays on the fake path.
