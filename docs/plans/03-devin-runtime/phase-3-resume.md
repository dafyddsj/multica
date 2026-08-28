# Phase 3. Resume

Back to [overview](overview.md).

## Goal

A follow-up chat continues the same Devin CLI session. A bad id fails before spawn. `-c` and bare `--resume` are never emitted.

## Changes

- `devinSessionID` parse-only constructors in `devin.go`.
- `resolveDevinResume` used by `Execute`.
- Tests for parse success, parse failure, cloud-shaped `devin-` rejection, and argv that includes `--resume <id>` only.

No archive-unarchive dance unless a canary shows `--print` archives the session.

## Data structures

`devinSessionID` is a branded string. Zero value means fresh. Construction is the only way to get a non-zero value.

## Verification

**Static.** `cd server && go test ./pkg/agent -run 'TestDevinResume|TestParseDevin' -count=1`

**Runtime.** Real-binary canary behind `MULTICA_RUN_REAL_AGENT_SMOKE=1` only.
