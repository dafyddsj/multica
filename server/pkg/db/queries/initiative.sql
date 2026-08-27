-- name: ListInitiatives :many
SELECT * FROM initiative
WHERE workspace_id = $1
  AND (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
  AND (sqlc.narg('priority')::text IS NULL OR priority = sqlc.narg('priority'))
ORDER BY created_at DESC;

-- name: GetInitiativeInWorkspace :one
SELECT * FROM initiative
WHERE id = $1 AND workspace_id = $2;

-- name: LockInitiativeForDelete :one
SELECT id FROM initiative
WHERE id = $1 AND workspace_id = $2
FOR UPDATE;

-- name: CreateInitiative :one
INSERT INTO initiative (
    workspace_id, title, description, icon, status,
    lead_type, lead_id, priority, start_date, due_date, issue_prefix
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: UpdateInitiative :one
UPDATE initiative SET
    title = COALESCE(sqlc.narg('title'), title),
    description = sqlc.narg('description'),
    icon = sqlc.narg('icon'),
    status = COALESCE(sqlc.narg('status'), status),
    priority = COALESCE(sqlc.narg('priority'), priority),
    lead_type = sqlc.narg('lead_type'),
    lead_id = sqlc.narg('lead_id'),
    start_date = sqlc.narg('start_date'),
    due_date = sqlc.narg('due_date'),
    issue_prefix = sqlc.narg('issue_prefix'),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteInitiative :exec
DELETE FROM initiative WHERE id = $1 AND workspace_id = $2;

-- name: GetInitiativeProjectCounts :many
SELECT initiative_id,
       count(*)::bigint AS project_count
FROM project
WHERE initiative_id = ANY(sqlc.arg('initiative_ids')::uuid[])
GROUP BY initiative_id;

-- name: GetInitiativeIssueStats :many
SELECT p.initiative_id,
       count(*)::bigint AS total_count,
       count(*) FILTER (WHERE issue_effective_status(i.workspace_id, i.status) IN ('done', 'cancelled'))::bigint AS done_count
FROM issue i
JOIN project p ON p.id = i.project_id
WHERE p.initiative_id = ANY(sqlc.arg('initiative_ids')::uuid[])
GROUP BY p.initiative_id;

-- name: ListProjectIDsByInitiative :many
SELECT id FROM project
WHERE workspace_id = $1 AND initiative_id = $2;

-- name: ClearProjectInitiativeByInitiative :exec
UPDATE project
SET initiative_id = NULL, updated_at = now()
WHERE initiative_id = $1 AND workspace_id = $2;

-- name: ListProjectInitiativeIssuePrefixes :many
-- Project → initiative prefix map for one workspace. Powers issue identifier
-- rendering so a list can apply overrides without an N+1.
SELECT p.id, i.issue_prefix
FROM project p
JOIN initiative i ON i.id = p.initiative_id
WHERE p.workspace_id = $1
  AND i.issue_prefix IS NOT NULL
  AND i.issue_prefix <> '';

-- name: ListInitiativeIssuePrefixes :many
-- Every non-empty initiative prefix in the workspace. Identifier lookup
-- accepts these in addition to the workspace prefix.
SELECT issue_prefix FROM initiative
WHERE workspace_id = $1
  AND issue_prefix IS NOT NULL
  AND issue_prefix <> '';
