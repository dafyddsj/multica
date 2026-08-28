# Phase 8. Docs and skills

Back to [overview](overview.md).

## Goal

Install docs, the providers table, env docs, and the creating-agents skill name Devin with the flags this plan verified. Chinese copy keeps the Latin product name Devin.

## Changes

- `apps/docs/content/docs/providers.mdx` and `.zh.mdx`, `.ko.mdx`, `.ja.mdx`. Add a Devin row. Detected command `devin`. Session resumption yes. Multica-managed MCP no until the canary. Skill path `.devin/skills/`.
- Install and env docs in en/zh/ko/ja. Auth is `devin auth login`. Probe env is `MULTICA_DEVIN_PATH` and `MULTICA_DEVIN_ARGS`.
- README, CLI_AND_DAEMON, SELF_HOSTING if those pages list families.
- `server/internal/service/builtin_skills/multica-creating-agents/SKILL.md` and `references/creating-agents-source-map.md` if create-time argv rules exist.
- Landing i18n. Recount the tool list at implement time. Do not assume a number.

Chinese voice. Read `apps/docs/content/docs/developers/conventions.zh.mdx` first. Devin stays Latin. Put a single space between Devin and surrounding Chinese. 运行时, 智能体, 守护进程, and 任务 follow the glossary. `skill` stays lowercase English.

Say plainly that v1 is the local CLI, not Devin Cloud.

## Data structures

None.

## Verification

**Static.** Grep the four locales for `Devin` and `devin`. Confirm no translated product name.

**Runtime.** Render the docs pages or grep the tables. No `control-ui` skill in this checkout.
