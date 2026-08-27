# Phase 2. Execute and parse

Back to [overview](overview.md).

## Goal

`ampBackend.Execute` starts a fake `amp`, writes the prompt on stdin, reads Claude-compatible JSONL, and returns `Message` events plus one `Result`.

## Changes

- Fill `Execute` on `ampBackend` in `server/pkg/agent/amp.go`. Reuse `newAgentStreamScanner`, `finalizeStreamResult`, `runContext`, `startOwnedProcessTree`, and `hasManagedMcpConfig`.
- Add a captured Amp JSONL fixture under `server/pkg/agent/testdata/` from the Amp appendix (system init, assistant text, success result, `session_id` like `T-…`).
- Extend `amp_test.go` with a fake executable, the same pattern as `qwen_test.go`.
- If that fixture unmarshals with Claude's event envelope, add Amp to `TestStreamJSONBackendsFinalOutputBoundaries`. If thinking lives in a `thinking` field instead of `text`, fork the parser the way Qwen did. Do not assume `handleAssistant` is shared.
- Write the prompt as plaintext on stdin, then close stdin. That is Amp's documented pipe path (`echo prompt | amp --execute --stream-json`). Do not keep stdin open for Claude `control_request`. Do not enable `--stream-json-input` in v1.
- Always pass `--stream-json-thinking`. Amp emits `thinking` and `redacted_thinking` on that flag. Treat `redacted_thinking` as understood with no text so it does not clear the #6006 fallback. Drop the flag in a follow-up if a capture shows it breaks a release.
- Copy Qwen's Windows `.cmd` to `.ps1` invocation shim (`amp_invocation*.go`). npm ships `amp.cmd`, and cmd re-tokenization mangles argv.

Do not teach Claude to launch Amp. Do not share `buildClaudeArgs`, `handleControlRequest`, `claudeRootSudoPreflight`, or the `CLAUDECODE*` env denylist. Ignore `opts.Model` and `opts.ThinkingLevel`. Log at debug when they are set.

## Data structures

`ampBackend struct { cfg Config }` implements `Backend`. Wire types stay private to `amp.go`.

## Verification

**Static.** `cd server && go test ./pkg/agent -run 'TestAmp' -count=1`

**Runtime.** Fake binary only. A live `amp` run belongs in [testing](testing.md) behind `MULTICA_RUN_REAL_AGENT_SMOKE=1`.
