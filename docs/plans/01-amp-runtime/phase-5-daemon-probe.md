# Phase 5. Daemon probe

Back to [overview](overview.md).

## Goal

A machine with `amp` on PATH, or `MULTICA_AMP_PATH` set, registers an Amp runtime. `MULTICA_AMP_ARGS` reaches `ExecOptions.ExtraArgs` and then argv.

## Changes

- `server/internal/daemon/agents_probe.go`: `probe("MULTICA_AMP_PATH", "amp", "MULTICA_AMP_MODEL")`.
- `server/internal/daemon/config.go`: add `"amp"` to `defaultAgentCommandNames`.
- `scripts/agent-cli-command-names.txt`: add `amp` in sorted position.
- `server/pkg/agent/version.go`: add a `MinVersions["amp"]` only after a real `--version` string and a known floor. If the floor is unknown, omit the entry and register any parseable version.
- ExtraArgs wiring test on `ampBackend`, same assertion as `TestQwenpawExtraArgsReachTheCommandLine`.
- `MULTICA_*_ARGS` is not generic. Add a `shellArgsFromEnv("MULTICA_AMP_ARGS")` case next to `MULTICA_QWEN_ARGS` in `config.go`, and the daemon forward into `ExecOptions.ExtraArgs`. Do not add the env var without both sides.
- `server/internal/metrics/labels.go` `knownRuntimeProviders`: add `"amp"`. A missing key collapses to `"other"`. `qwenpaw` already has that hole. Do not copy it.

## Data structures

`AgentEntry{Path, Command: "amp", Model}` from the existing probe helper.

## Verification

**Static.** `cd server && go test ./internal/daemon -run 'CommandNames|Probe' -count=1` and `go test ./pkg/agent -run 'TestAmpExtraArgs' -count=1`

**Runtime.** Use `control-cli` if present. Otherwise run `command -v amp` on a machine that has Amp and confirm the daemon log line that registered provider `amp`. Default CI stays on the fake path.
