# Designed workflow

Rigor: high. One-way door is the pause data shape. Gates and artifacts, not hope.

## Phase list

1. How. Four explorers plus one explainer. Artifact: subsystem map.
2. Architect arena. Four isolated design packages. Artifact: synthesized sketch in `.audit/design/`.
3. Throughput checkpoint. Blocking types first. Then serialized implementation.
4. Harness. Failing tests for pause gates and presence before feature code.
5. Backend unit. Migration, SQL, handlers, enqueue/claim/assign/chat gates. Verify with Go tests.
6. Core unit. Types, schema defaults, API client, events, presence derivation. Verify with Vitest.
7. Views unit. Row actions, detail, picker, i18n. Verify with Vitest.
8. CLI and mobile picker filters. Verify with existing command/test style.
9. Browser demo on the real web app. Artifact: video.
10. Interrogate if the design stays contested.
11. Opening a PR. Ready, not draft.

## Throughput checkpoint

- Blocking first steps. Data shape, migration, Agent type, one `agentAcceptsNewWork` helper. Nothing fans out until those exist.
- Independent workstreams. After the helper exists, Go gate tests and presence/picker tests can proceed on the same branch in series, not parallel writers.
- Shared mutable state. The `agent` row, `Agent` type, `AgentAvailability` union, locale files. One writer.
- Smallest safe decomposition. One owner after architect. Parallel workers would fight the same row and the same type.

## Sequence

Failing tests first, then the code that makes them pass. Each unit ends in a check before the next starts.

## Models

- How explorers: cursor-grok-4.6-high-fast
- How explainer: claude-fable-5-thinking-xhigh
- Arena runners: claude-fable-5-thinking-xhigh, gpt-5.6-sol-xhigh, cursor-grok-4.6-high-fast, claude-opus-5-thinking-high
- Implementer: cursor-grok-4.6-high-fast
- Trail review: gpt-5.6-sol-xhigh
