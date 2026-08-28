# Phase 3. Resume

Back to [overview](overview.md).

## Goal

A later `Execute` with `ResumeSessionID` continues Amp's thread. A refused continue sets `Result.ResumeRejected`.

## Changes

- `resolveAmpResume` runs before spawn. Empty `ResumeSessionID` is a fresh turn. A non-empty value that is not `T-<uuid>` fails `Execute`. Do not launch `threads continue` without a parsed id. Bare continue follows the latest thread on the Amp account and races on a shared daemon host.
- `buildAmpArgs` emits `threads continue <id>` plus the execute flags only when `resume` is a parsed `ampThreadID`.
- Parse `session_id` from init and result events. Persist it on `Result.SessionID` only after `parseAmpThreadID` succeeds. If Amp emits a non-`T-` token, leave `SessionID` empty rather than storing garbage the next claim would refuse or, worse, continue as latest.
- Reuse `resumeWasRejected` if Amp's wording matches. Add phrases only from a captured real-CLI stderr fixture, with a provenance comment like Qwen's. Do not add `"amp"` to `resumeRejectionUndetectable` unless you cannot detect refusal.
- Tests for continue argv, session id round-trip, malformed resume failing `Execute`, and a refused-thread fixture.

## Data structures

`Result.SessionID` stays a string. The value is a parsed `T-<uuid>` or empty. Do not strip the `T-` prefix. Do not persist the raw stream field on parse failure.

## Verification

**Static.** `cd server && go test ./pkg/agent -run 'TestAmp' -count=1`

**Runtime.** Fake binary that prints a different `session_id` than the requested one, and a fake that exits with a "thread not found" stderr. No live Amp.
