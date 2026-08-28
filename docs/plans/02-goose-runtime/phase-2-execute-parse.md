# Phase 2. Execute and parse

Back to [overview](overview.md).

## Goal

`gooseBackend.Execute` starts `goose`, writes the prompt on stdin, closes stdin, and parses stream-json into `Message` and `Result`.

## Changes

- `Execute` on `gooseBackend` in `goose.go`. Reuse `newAgentStreamScanner`, `finalizeStreamResult`, `assistantTurn` / `resolveFallback`, `startOwnedProcessTree`, and `filterCustomArgs`.
- Fixture file under `server/pkg/agent/testdata/` captured from a real `goose run --output-format stream-json` once a signed-in canary exists. Until then, parse tests use a hand-built fixture that matches documented event names only after those names appear in a capture. Do not invent event types.
- Fake executable tests that check stdin contents and that stdin is closed.

Do not persist a raw `session_id` that failed the parser.

## Data structures

Private Goose stream event structs next to the backend. Do not share Claude or Qwen wire types.

## Verification

**Static.** `cd server && go test ./pkg/agent -run 'TestGoose' -count=1`

**Runtime.** Fake executable only in default CI.
