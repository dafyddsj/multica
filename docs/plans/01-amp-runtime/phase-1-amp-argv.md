# Phase 1. Amp argv

Back to [overview](overview.md).

## Goal

A test-pinned argv builder for Amp that cannot put the prompt on the command line and cannot let custom args override the stream or permission flags.

## Changes

- Add `server/pkg/agent/amp.go` with `buildAmpArgs` and `ampBlockedArgs` only. No `Execute` yet.
- Add `server/pkg/agent/amp_test.go` that asserts the flag set and the blocks.

Confirm the live flag names with `amp --help` and `amp threads continue --help` if a binary is available. If not, pin the documented names and add a test comment that the first real-binary smoke must re-read help.

## Data structures

`ampBlockedArgs map[string]blockedArgMode` owns `--execute`, `-x`, `--stream-json`, `--stream-json-input`, `--stream-json-thinking`, `--dangerously-allow-all`, `--mcp-config`, `--resume`, `--continue`, and the `threads` subcommand tokens.

## Verification

**Static.** `cd server && go test ./pkg/agent -run 'TestBuildAmpArgs|TestAmpBlockedArgs' -count=1`

**Runtime.** No CLI process yet. `control-cli` is not applicable.
