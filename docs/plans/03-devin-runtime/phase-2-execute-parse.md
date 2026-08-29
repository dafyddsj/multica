# Phase 2. Execute and parse

Back to [overview](overview.md).

## Goal

`devinBackend.Execute` writes the prompt to a 0600 temp file, starts `devin --print`, and turns stdout into `Message` and `Result`.

## Changes

- `Execute` on `devinBackend` in `devin.go`. Reuse `startOwnedProcessTree`, `filterCustomArgs`, and `finalizeStreamResult` if the capture is JSONL. If `--print` is plain text, wrap it as a single assistant message. Do not invent a stream schema.
- Temp prompt file helper with cleanup after `cmd.Wait`.
- Fake executable tests that check the prompt file contents and that `-p "…prompt…"` is absent from argv.

Do not persist a raw session token that failed the parser.

## Data structures

Private result helpers next to the backend. Do not share Claude wire types.

## Verification

**Static.** `cd server && go test ./pkg/agent -run 'TestDevin' -count=1`

**Runtime.** Fake executable only in default CI.
