# Memory v1

Multica-owned, user-visible memory. Agents and humans pin, list, search, and forget notes on existing objects. Hermes `MEMORY.md` stays a separate runtime scratch pad.

This folder is the implementation record. Read `IMPLEMENTATION.md` for the
code map and how to enable the gates. Product copy lives in
`apps/docs/content/docs/memory.mdx`.

## What shipped

- Postgres `memory_entry` rows, one type, many owners (migrations 443–445)
- HTTP API under `/api/memory`
- CLI `multica memory {add,list,get,search,recall,forget}`
- Feature flag `memory_v1` (off by default)
- Labs toggle `memory_enabled` on the workspace (off by default)
- Per-turn recall on claim when both gates are on
- A `MemoryEngine` interface so a later run can plug in Hindsight (or similar) without changing the API

## Gates

Both must be on or every write/read/recall is 404:

1. Deployment flag `memory_v1`
2. Workspace setting `memory_enabled` (Labs)

User-scope rows store `workspace_id` as null. They are still only reachable through a workspace that has Labs on, because the API is workspace-membered.

## Scopes

| Scope | Owner | `workspace_id` |
| --- | --- | --- |
| `workspace` | workspace id | required |
| `initiative` | initiative id | required |
| `project` | project id | required |
| `issue` | issue id | required |
| `squad` | squad id | required |
| `agent` | agent id | required |
| `user` | user id | null |

See `DESIGN.md` for the record, auth, delete, and engine seam.

## What this is not

- Not Hermes `MEMORY.md` / `USER.md`
- Not Codex native auto-memory
- Not embeddings or Hindsight in this PR
- Not a new root route. UI lives on Labs plus entity pages
