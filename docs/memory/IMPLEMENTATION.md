# How memory v1 is implemented

This is the implementation record for the first ship. Product rules live in
`DESIGN.md`. This file is the map from those rules to code, plus how to turn
the feature on and how a later run plugs in Hindsight.

## Gates

Both must be on. Either off is a 404 `"memory is not enabled"` on every HTTP
verb, including recall. Claim-time injection is a no-op.

| Gate | Default | Where |
| --- | --- | --- |
| Feature flag `memory_v1` | off | `server/internal/featureflags/keys.go`, published on `/api/config` |
| Workspace Labs `memory_enabled` | off | `workspace.settings` JSON, key `memory_enabled` |

Local override for the flag:

```bash
FF_MEMORY_V1=true
```

Or a YAML rule at `MULTICA_FEATURE_FLAGS_FILE`:

```yaml
memory_v1:
  default: true
```

The Labs toggle is **Settings → Labs**. `PATCH /api/workspaces/{id}` is
owner/admin-only, so a member cannot flip the workspace on. Turning Labs off
stops reads, writes, and claim recall. Rows stay.

## Layout

```
docs/memory/                          this folder
server/migrations/443–445             table + concurrent indexes
server/pkg/db/queries/memory.sql      sqlc
server/internal/memory/               types, gates, MemoryEngine, NativeEngine
server/internal/handler/memory.go     HTTP + claim attach + owner delete
server/cmd/multica/cmd_memory.go      CLI
packages/core/types/memory.ts         shared types
packages/core/memory/                 query keys + mutations
packages/core/api/{client,schemas}.ts parseWithFallback
packages/views/memory/                MemoryPanel
packages/views/settings/.../labs-tab  Labs toggle
apps/docs/content/docs/memory.mdx     product docs
```

There is no new root route. UI hangs on Labs plus existing entity pages
(issue, project, initiative, squad, agent, workspace settings, account).

## Persistence

One table, `memory_entry`. Scope plus owner is the bank. The check
`(scope = 'user') = (workspace_id IS NULL)` makes the invalid mix
unrepresentable.

No foreign keys. Application code deletes the bank when the owner is deleted
or archived:

- issue / project / initiative delete
- squad / agent archive
- workspace teardown (`DeleteWorkspaceMemoryEntries` before leaf data)

User-scope rows survive workspace delete. Forget is `deleted_at`. Cap is 200
live rows per bank, enforced in the handler before insert.

Indexes are concurrent and live in their own migration files (444, 445).

## Engine seam

```go
type MemoryEngine interface {
    Search(ctx context.Context, q SearchQuery) ([]Hit, error)
}
```

`Handler.MemoryEngine` is optional. Nil uses `NativeEngine`, which runs
`ListMemoryEntriesForRecall` (`ILIKE` on `body`, 8 hits, 400 runes).

A later Hindsight (or similar) adapter implements the same interface and is
wired onto `Handler.MemoryEngine` at process start. Do not change `/api/memory`
or `multica memory` when that happens. Agents never talk to the engine.

Recommended Hindsight mapping (not built in this PR):

- one team bank per workspace, tags for scope + owner
- one bank per agent
- one bank per user

Do not make one Hindsight bank per Multica tier. Hindsight cannot query across
banks. Recall ancestry is a Multica concern.

## HTTP and CLI

```
GET    /api/memory?scope=&owner_id=&q=&limit=
POST   /api/memory
GET    /api/memory/{id}
PATCH  /api/memory/{id}
DELETE /api/memory/{id}          # forget
GET    /api/memory/recall?...
```

`GET /api/memory/recall` is registered before `/{id}`.

CLI: `multica memory {add,list,search,get,recall,forget}`.

Auth is in `authorizeMemory`. Workspace write is owner/admin. Initiative /
project / issue write is any member. Squad write is leader, leader-owner, or
owner/admin. Agent is owner, admin, invoker, or the agent. User is that user.

## Claim and prompt

`buildClaimedTaskResponse` calls `attachClaimMemory` after workspace context
is loaded. Ancestry:

- always workspace + this agent
- issue / project / initiative when the claim already resolved them
- squad on a leader task, or when the issue is assigned to that squad
- user = runtime owner

Hits are copied onto `AgentTaskResponse.MemoryHits` and the daemon's
`Task.MemoryHits`. `buildMemoryBlock` appends `## Memory` to the **per-turn**
prompt, not the cached brief (MUL-5377).

CLI / HTTP `recall` is not claim recall. `multica memory recall --issue-id`
adds that issue bank on top of workspace + the caller's user bank. It does
not infer agent, project, initiative, or squad. Pass those flags explicitly.

User-scope notes on claim are the **runtime owner's** bank — the same person
as `## Requesting User`. Anyone whose task runs on that runtime sees those
notes. That is the v1 rule; a later provider run can revisit it.

When the gates are off, `MemoryEnabled` stays false and existing briefs stay
byte-identical.

## Frontend

Both gates hide the UI:

- `useFeatureEnabled(MEMORY_V1_FLAG)`
- `isWorkspaceMemoryEnabled(workspace.settings)` (`=== true` only)

`MemoryPanel` lists, searches, pins, and forgets one bank. Labs shows an empty
state when the flag is off, and an owner/admin switch when it is on.

API JSON stays snake_case through `parseWithFallback` and the zod schemas in
`packages/core/api/schemas.ts`. Malformed payloads fall back; see
`schemas.test.ts`.

## What this PR does not do

- No embeddings, no Hindsight client, no provider picker
- No Hermes `MEMORY.md` sync
- No promotion up the tree
- No mobile UI
- No new global route
