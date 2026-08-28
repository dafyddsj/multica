# Phase 8. Docs and skills

Back to [overview](overview.md).

## Goal

Install docs, the providers table, env docs, and the creating-agents skill name Goose with the flags this plan verified. Chinese copy keeps the Latin product name Goose.

## Changes

- `apps/docs/content/docs/providers.mdx` and `.zh.mdx`, `.ko.mdx`, `.ja.mdx`. Add a Goose row. Detected command `goose`. Session resumption yes. Multica-managed MCP no until the canary. Skill path `.agents/skills/`.
- Install and env docs in en/zh/ko/ja. Auth is `goose configure` plus `GOOSE_PROVIDER` / `GOOSE_MODEL` / the vendor key. Probe env is `MULTICA_GOOSE_PATH` and `MULTICA_GOOSE_ARGS`.
- README, CLI_AND_DAEMON, SELF_HOSTING if those pages list families.
- `server/internal/service/builtin_skills/multica-creating-agents/SKILL.md` and `references/creating-agents-source-map.md` if create-time argv rules exist.
- Landing i18n in `apps/web/features/landing/i18n/{en,zh,ko,ja}.ts`. Recount the tool list at implement time. Do not assume 24. Amp may have landed. ZeroClaw is already missing from the current 23. Do not silently "fix" that count in this family PR unless the list is already being edited.

Chinese voice. Read `apps/docs/content/docs/developers/conventions.zh.mdx` first. Goose stays Latin. Put a single space between Goose and surrounding Chinese. 运行时, 智能体, 守护进程, and 任务 follow the glossary. `skill` stays lowercase English.

## Data structures

None.

## Verification

**Static.** Grep the four locales for `Goose` and `goose`. Confirm no translated product name.

**Runtime.** Render the docs pages or grep the tables. No `control-ui` skill in this checkout.
