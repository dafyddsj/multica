# Phase 1. Goose argv

Back to [overview](overview.md).

## Goal

`buildGooseArgs` emits the 1.48.0 `goose run` contract. Prompt text is not on argv. Custom args cannot break stdin, stream-json, or resume.

## Changes

- New `server/pkg/agent/goose.go` with `gooseBlockedArgs` and `buildGooseArgs`.
- Tests in `goose_test.go` for fresh argv, resume argv, blocked flags, and ExtraArgs order.

Do not spawn a process yet.

## Data structures

`gooseBlockedArgs map[string]blockedArgMode` owns `-i`, `--instructions`, `-t`, `--text`, `--output-format`, `--resume`, `--session-id`, `--no-session`, `-s`, `--interactive`, `--recipe`, `--acp`, and `serve`.

## Verification

**Static.** `cd server && go test ./pkg/agent -run 'TestBuildGoose|TestFilterGoose' -count=1`

**Runtime.** None. Argv only.
