# Phase 1. Amp argv

Back to [overview](overview.md).

## Goal

A test-pinned argv builder for Amp that cannot put the prompt on the command line and cannot let custom args override the stream or permission flags.

## Changes

- Add `server/pkg/agent/amp.go` with `ampThreadID`, `parseAmpThreadID`, `resolveAmpResume`, `buildAmpArgs`, and `ampBlockedArgs`. No `Execute` yet.
- `buildAmpArgs` takes `resume ampThreadID`, not a raw string. Zero resume is a fresh `--execute` turn. A valid resume prepends `threads continue <id>`. The function cannot emit `--resume`.
- Add `server/pkg/agent/amp_test.go` that asserts the flag set, the blocks, and that a malformed resume never reaches argv.

Confirm the live flag names with `amp --help` and `amp threads continue --help` if a binary is available. If not, pin the documented names and add a test comment that the first real-binary smoke must re-read help.

`--dangerously-allow-all` is always present. Do not add `--model`. Do not add `--effort`.

Filter `LaunchPrefix` as sequences, not as standalone tokens. Removing `threads` or `continue` one at a time can leave a stray `T-` positional on the command line. Drop the whole `threads continue <id>` run, or keep it.

## Data structures

`ampThreadID` is a validated `T-<uuid>`. The only constructors are `parseAmpThreadID` and `resolveAmpResume`.

`ampBlockedArgs` owns `--execute`, `-x`, `--stream-json`, `--stream-json-input`, `--stream-json-thinking`, `--dangerously-allow-all`, `--no-archive-after-execute`, `--unarchive`, `--mcp-config`, `--resume`, `--continue`, the `threads` subcommand tokens, plus `--no-tui`, `--executor`, `-p`, and `--output-format` so pasted Claude or orb flags cannot change the launch.

## Verification

**Static.** `cd server && go test ./pkg/agent -run 'TestBuildAmpArgs|TestAmpBlockedArgs' -count=1`

**Runtime.** No CLI process yet. `control-cli` is not applicable.
