# Phase 3. Resume

Back to [overview](overview.md).

## Goal

A follow-up chat continues the same Goose session. A bad id fails before spawn. Bare `--resume` is never emitted.

## Changes

- `gooseSessionID` parse-only constructors in `goose.go`.
- `resolveGooseResume` used by `Execute`.
- Tests for parse success, parse failure, and argv that includes both `--resume` and `--session-id`.

No archive-unarchive dance unless a canary shows Goose archives on execute. 1.48.0 help has no such flag.

## Data structures

`gooseSessionID` is a branded string. Zero value means fresh. Construction is the only way to get a non-zero value.

## Verification

**Static.** `cd server && go test ./pkg/agent -run 'TestGooseResume|TestParseGoose' -count=1`

**Runtime.** Real-binary canary behind `MULTICA_RUN_REAL_AGENT_SMOKE=1` only.
