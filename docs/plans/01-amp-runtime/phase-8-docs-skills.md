# Phase 8. Docs and skills

Back to [overview](overview.md).

## Goal

A human can install Amp, sign in, and see it in the supported-tools tables. The creating-agents skill names Amp's argv rules.

## Changes

- `apps/docs/content/docs/install-agent-runtime.mdx` and locale siblings: a row for Amp, command `amp`, link `https://ampcode.com`.
- `apps/docs/content/docs/providers.mdx` and locale siblings: session resume yes, Multica-managed MCP only if the phase 6 canary passed, skill path as documented (likely `AGENTS.md` / Amp plugins, not an invented `.amp/skills` unless phase 6 found one).
- `apps/docs/content/docs/environment-variables.mdx` and the zh/ko/ja siblings: `MULTICA_AMP_PATH`, `MULTICA_AMP_ARGS`, and `AMP_API_KEY` passthrough. Do not document `MULTICA_AMP_MODEL`. Locale env pages already drift (EN names Dim, zh/ko/ja do not). Add Amp to all four.
- `README.md`, `CLI_AND_DAEMON.md` (detected-command table and env table), and `SELF_HOSTING.md`. Zeroclaw landed in factory and env docs and missed these three. Amp must not.
- `server/internal/service/builtin_skills/multica-creating-agents/SKILL.md` and `references/creating-agents-source-map.md` only if Amp has create-time argv rules (blocked flags, ExtraArgs). Use Cursor's **create-skill** skill when editing `SKILL.md`.

Do not ship Amp in env docs while omitting the install table or the README list.

## Data structures

None.

## Verification

**Static.** Grep the four locale install tables and four providers tables for `| Amp |`. All must mention `amp`.

**Runtime.** Open the docs page `/install-agent-runtime` and `/providers` if the docs app is running. This is a docs change. A render check is enough.
