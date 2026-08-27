# Memory v1 design

## Data shape

One table. Scope plus owner is the bank. Invalid mixes are unrepresentable.

```
MemoryScope = workspace | initiative | project | issue | squad | agent | user
MemoryKind  = fact | preference | procedure | observation

MemoryEntry
  id
  workspace_id     // null iff scope = user
  scope
  owner_id
  body             // 1..4000 runes
  kind
  provenance       // JSON object, optional issue/task/author hints
  created_by_type  // member | agent
  created_by_id
  created_at
  updated_at
  deleted_at       // forget is soft delete
```

Check: `(scope = 'user') = (workspace_id IS NULL)`.

Cap: 200 live entries per `(scope, owner_id)` bank.

## Public surface

```
GET    /api/memory?scope=&owner_id=&q=&limit=
POST   /api/memory
GET    /api/memory/{id}
PATCH  /api/memory/{id}
DELETE /api/memory/{id}
GET    /api/memory/recall?q=&issue_id=&project_id=&initiative_id=&squad_id=&agent_id=
```

CLI mirrors those verbs. `forget` is DELETE.

## Auth

| Scope | Read | Write |
| --- | --- | --- |
| workspace | member | owner / admin |
| initiative, project, issue | member | member |
| squad | member | squad leader, owner, admin |
| agent | owner, admin, invoker, the agent | same |
| user | that user | that user |

Recall ancestry for a run:

- always workspace + this agent
- issue when the task has one
- project / initiative when claim already resolved them
- squad when the run is a leader task or the issue is assigned to that squad
- user = runtime owner, same person as `## Requesting User`

Hits go in the per-turn prompt, not the cached brief.

## Delete

Bank dies with its owner. No promotion up the tree.

- Issue / project / initiative delete: `DELETE` live rows for that `(scope, owner_id)`
- Squad / agent **archive**: same hard delete, after the archive update succeeds. Restore does not bring the bank back.
- Workspace delete: all rows with that `workspace_id`
- User rows survive workspace delete

Manual `GET /api/memory/recall` always includes the current workspace bank and the caller's user bank. Optional banks are included only when their query param is set. Claim-time recall is different: it auto-resolves ancestry (workspace, this agent, issue/project/initiative when known, squad when relevant, runtime-owner user).

## Engine seam

```go
type MemoryEngine interface {
    Search(ctx context.Context, q SearchQuery) ([]Hit, error)
}
```

v1 uses `NativeEngine` (`ILIKE` on `body`). A later Hindsight engine implements the same interface. Agents never talk to the engine. They talk to Multica.

Recommended Hindsight mapping when that run happens:

- one team bank per workspace, tags for scope + owner
- one bank per agent
- one bank per user

Do not make one Hindsight bank per Multica tier. Hindsight cannot query across banks.

## Hermes

Keep the overlay store. Do not sync `MEMORY.md` into `memory_entry`. An agent that wants a note on the platform runs `multica memory add`.
