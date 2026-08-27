# Phase 3. Resume

Back to [overview](overview.md).

## Goal

A later `Execute` with `ResumeSessionID` continues Amp's thread. A refused continue sets `Result.ResumeRejected`.

## Changes

- `buildAmpArgs` emits `threads continue <id>` plus the execute flags when `opts.ResumeSessionID` is set.
- Parse `session_id` from init and result events into `Result.SessionID`.
- Reuse `resumeWasRejected` if Amp's wording matches. Otherwise add Amp phrases next to the existing list in `claude.go`, or a private Amp list if the strings are Amp-only. Do not add `"amp"` to `resumeRejectionUndetectable` unless you cannot detect refusal.
- Tests for continue argv, session id round-trip, and a refused-thread fixture.

## Data structures

`Result.SessionID` stays a string. Amp's `T-<uuid>` is a legal value. Do not strip the `T-` prefix.

## Verification

**Static.** `cd server && go test ./pkg/agent -run 'TestAmp' -count=1`

**Runtime.** Fake binary that prints a different `session_id` than the requested one, and a fake that exits with a "thread not found" stderr. No live Amp.
