# Phase 2. Execute and parse

Back to [overview](overview.md).

## Goal

`ampBackend.Execute` starts a fake `amp`, writes the prompt on stdin, reads Claude-compatible JSONL, and returns `Message` events plus one `Result`.

## Changes

- Fill `Execute` on `ampBackend` in `server/pkg/agent/amp.go`. Reuse `newAgentStreamScanner`, `runContext`, `startOwnedProcessTree`, `hasManagedMcpConfig`, and the shared stream-json result helpers already used by Claude and Qwen.
- Add a captured Amp JSONL fixture under `server/pkg/agent/testdata/` copied from the Amp appendix example (system init, assistant text, success result, `session_id` like `T-…`).
- Extend `amp_test.go` with a fake executable, the same pattern as `qwen_test.go` / `claude_test.go`.

Do not teach Claude to launch Amp. Do not share `buildClaudeArgs`.

## Data structures

`ampBackend struct { cfg Config }` implements `Backend`. Wire types stay private to `amp.go`.

## Verification

**Static.** `cd server && go test ./pkg/agent -run 'TestAmp' -count=1`

**Runtime.** Fake binary only. A live `amp` run belongs in [testing](testing.md) behind `MULTICA_RUN_REAL_AGENT_SMOKE=1`.
