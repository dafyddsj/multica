# Phase 8. Docs and skills

Back to [overview](overview.md).

## Goal

A human can install Amp, sign in, and see it in the supported-tools tables. The creating-agents skill names Amp's argv rules.

## Changes

- `apps/docs/content/docs/install-agent-runtime.mdx` and locale siblings: a row for Amp, command `amp`, link `https://ampcode.com`.
- `apps/docs/content/docs/providers.mdx` and locale siblings: session resume yes, Multica-managed MCP yes, skill path as documented (likely `AGENTS.md` / Amp plugins, not a invented `.amp/skills` unless phase 6 found one).
- `apps/docs/content/docs/environment-variables.mdx`: `MULTICA_AMP_PATH`, `MULTICA_AMP_MODEL`, `MULTICA_AMP_ARGS`.
- `server/internal/service/builtin_skills/multica-creating-agents/SKILL.md` and `references/creating-agents-source-map.md`. Use Cursor's **create-skill** skill when editing `SKILL.md`.

Do not ship Amp in env docs while omitting the install table. Zeroclaw already has that hole.

## Data structures

None.

## Verification

**Static.** Grep the four locale install tables and four providers tables for `| Amp |`. All must mention `amp`.

**Runtime.** Open the docs page `/install-agent-runtime` and `/providers` if the docs app is running. This is a docs change. A render check is enough.
