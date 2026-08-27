-- name: CreateMemoryEntry :one
INSERT INTO memory_entry (
    workspace_id, scope, owner_id, body, kind, provenance,
    created_by_type, created_by_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetMemoryEntry :one
SELECT * FROM memory_entry
WHERE id = $1
  AND deleted_at IS NULL
  AND (
    workspace_id = sqlc.arg('workspace_id')
    OR (scope = 'user' AND workspace_id IS NULL)
  );

-- name: ListMemoryEntries :many
SELECT * FROM memory_entry
WHERE deleted_at IS NULL
  AND scope = $1
  AND owner_id = $2
  AND (
    (scope <> 'user' AND workspace_id = sqlc.arg('workspace_id'))
    OR (scope = 'user' AND workspace_id IS NULL)
  )
  AND (sqlc.narg('query')::text IS NULL OR body ILIKE '%' || sqlc.narg('query') || '%')
ORDER BY created_at DESC
LIMIT $3;

-- name: CountMemoryEntriesInBank :one
SELECT count(*)::bigint
FROM memory_entry
WHERE deleted_at IS NULL
  AND scope = $1
  AND owner_id = $2
  AND (
    (scope <> 'user' AND workspace_id = sqlc.arg('workspace_id'))
    OR (scope = 'user' AND workspace_id IS NULL)
  );

-- name: UpdateMemoryEntry :one
UPDATE memory_entry SET
    body = COALESCE(sqlc.narg('body'), body),
    kind = COALESCE(sqlc.narg('kind'), kind),
    provenance = COALESCE(sqlc.narg('provenance'), provenance),
    updated_at = now()
WHERE id = $1
  AND deleted_at IS NULL
  AND (
    workspace_id = sqlc.arg('workspace_id')
    OR (scope = 'user' AND workspace_id IS NULL)
  )
RETURNING *;

-- name: SoftDeleteMemoryEntry :one
UPDATE memory_entry SET
    deleted_at = now(),
    updated_at = now()
WHERE id = $1
  AND deleted_at IS NULL
  AND (
    workspace_id = sqlc.arg('workspace_id')
    OR (scope = 'user' AND workspace_id IS NULL)
  )
RETURNING *;

-- name: DeleteMemoryEntriesByOwner :exec
DELETE FROM memory_entry
WHERE scope = $1
  AND owner_id = $2
  AND workspace_id = $3;

-- name: DeleteWorkspaceMemoryEntries :exec
DELETE FROM memory_entry
WHERE workspace_id = $1;

-- name: ListMemoryEntriesForRecall :many
SELECT * FROM memory_entry
WHERE deleted_at IS NULL
  AND (
    (scope = 'workspace' AND workspace_id = sqlc.arg('workspace_id') AND owner_id = sqlc.arg('workspace_id'))
    OR (scope = 'initiative' AND workspace_id = sqlc.arg('workspace_id') AND owner_id = sqlc.narg('initiative_id'))
    OR (scope = 'project' AND workspace_id = sqlc.arg('workspace_id') AND owner_id = sqlc.narg('project_id'))
    OR (scope = 'issue' AND workspace_id = sqlc.arg('workspace_id') AND owner_id = sqlc.narg('issue_id'))
    OR (scope = 'squad' AND workspace_id = sqlc.arg('workspace_id') AND owner_id = sqlc.narg('squad_id'))
    OR (scope = 'agent' AND workspace_id = sqlc.arg('workspace_id') AND owner_id = sqlc.narg('agent_id'))
    OR (scope = 'user' AND workspace_id IS NULL AND owner_id = sqlc.narg('user_id'))
  )
  AND (sqlc.narg('query')::text IS NULL OR body ILIKE '%' || sqlc.narg('query') || '%')
ORDER BY
  CASE scope
    WHEN 'issue' THEN 1
    WHEN 'project' THEN 2
    WHEN 'initiative' THEN 3
    WHEN 'squad' THEN 4
    WHEN 'agent' THEN 5
    WHEN 'workspace' THEN 6
    WHEN 'user' THEN 7
    ELSE 8
  END,
  created_at DESC
LIMIT $1;
