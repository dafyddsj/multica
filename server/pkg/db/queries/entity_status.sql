-- Initiative and Project status catalog. Each workspace holds the 5 built-in
-- statuses per resource_type plus any custom ones. A category's value IS its
-- canonical built-in key.

-- name: SeedEntityStatusEntries :exec
-- Idempotent seed of the 5 built-ins for both resource types. Safe to call
-- concurrently: the unique (workspace_id, resource_type, key) index makes a
-- losing racer a no-op.
INSERT INTO entity_status (workspace_id, resource_type, key, name, description, category, color, is_system, position)
VALUES
    (sqlc.arg('workspace_id')::uuid, 'initiative', 'planned', 'Planned', 'Not started yet.', 'planned', '#6b7280', TRUE, 0),
    (sqlc.arg('workspace_id')::uuid, 'initiative', 'in_progress', 'In Progress', 'Actively being worked on.', 'in_progress', '#f59e0b', TRUE, 0),
    (sqlc.arg('workspace_id')::uuid, 'initiative', 'paused', 'Paused', 'Temporarily on hold.', 'paused', '#6b7280', TRUE, 0),
    (sqlc.arg('workspace_id')::uuid, 'initiative', 'completed', 'Completed', 'Finished.', 'completed', '#3b82f6', TRUE, 0),
    (sqlc.arg('workspace_id')::uuid, 'initiative', 'cancelled', 'Cancelled', 'Decided not to continue.', 'cancelled', '#ef4444', TRUE, 0),
    (sqlc.arg('workspace_id')::uuid, 'project', 'planned', 'Planned', 'Not started yet.', 'planned', '#6b7280', TRUE, 0),
    (sqlc.arg('workspace_id')::uuid, 'project', 'in_progress', 'In Progress', 'Actively being worked on.', 'in_progress', '#f59e0b', TRUE, 0),
    (sqlc.arg('workspace_id')::uuid, 'project', 'paused', 'Paused', 'Temporarily on hold.', 'paused', '#6b7280', TRUE, 0),
    (sqlc.arg('workspace_id')::uuid, 'project', 'completed', 'Completed', 'Finished.', 'completed', '#3b82f6', TRUE, 0),
    (sqlc.arg('workspace_id')::uuid, 'project', 'cancelled', 'Cancelled', 'Decided not to continue.', 'cancelled', '#ef4444', TRUE, 0)
ON CONFLICT DO NOTHING;

-- name: ListEntityStatusEntries :many
SELECT * FROM entity_status
WHERE workspace_id = sqlc.arg('workspace_id')::uuid
  AND resource_type = sqlc.arg('resource_type')::text
  AND (sqlc.arg('include_archived')::bool OR archived_at IS NULL)
ORDER BY
    CASE category
        WHEN 'planned' THEN 0
        WHEN 'in_progress' THEN 1
        WHEN 'paused' THEN 2
        WHEN 'completed' THEN 3
        WHEN 'cancelled' THEN 4
        ELSE 5
    END,
    position,
    key;

-- name: GetEntityStatusEntryByKey :one
SELECT * FROM entity_status
WHERE workspace_id = sqlc.arg('workspace_id')::uuid
  AND resource_type = sqlc.arg('resource_type')::text
  AND key = sqlc.arg('key')::text;

-- name: GetEntityStatusEntryByID :one
SELECT * FROM entity_status
WHERE id = sqlc.arg('id')::uuid
  AND workspace_id = sqlc.arg('workspace_id')::uuid;

-- name: CreateEntityStatusEntry :one
INSERT INTO entity_status (workspace_id, resource_type, key, name, description, category, color, position)
VALUES (
    sqlc.arg('workspace_id')::uuid,
    sqlc.arg('resource_type')::text,
    sqlc.arg('key')::text,
    sqlc.arg('name')::text,
    sqlc.arg('description')::text,
    sqlc.arg('category')::text,
    sqlc.arg('color')::text,
    COALESCE(
        (SELECT MAX(position) + 1 FROM entity_status
         WHERE workspace_id = sqlc.arg('workspace_id')::uuid
           AND resource_type = sqlc.arg('resource_type')::text
           AND category = sqlc.arg('category')::text),
        0
    )
)
RETURNING *;

-- name: UpdateEntityStatusEntry :one
-- key, category and resource_type are immutable. Built-ins may have name,
-- description and color edited — unlike issue statuses, these catalogs are
-- meant to be customised from settings.
UPDATE entity_status SET
    name = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    color = COALESCE(sqlc.narg('color'), color),
    position = COALESCE(sqlc.narg('position'), position),
    updated_at = now()
WHERE id = sqlc.arg('id')::uuid
  AND workspace_id = sqlc.arg('workspace_id')::uuid
  AND archived_at IS NULL
RETURNING *;

-- name: ArchiveEntityStatusEntry :one
UPDATE entity_status SET
    archived_at = now(),
    updated_at = now()
WHERE id = sqlc.arg('id')::uuid
  AND workspace_id = sqlc.arg('workspace_id')::uuid
  AND is_system = FALSE
  AND archived_at IS NULL
RETURNING *;

-- name: DeleteEntityStatusEntriesForWorkspace :exec
DELETE FROM entity_status WHERE workspace_id = sqlc.arg('workspace_id')::uuid;

-- name: LockEntityStatusCatalog :exec
SELECT pg_advisory_xact_lock(hashtextextended(sqlc.arg('workspace_id')::uuid::text || ':entity_status:' || sqlc.arg('resource_type')::text, 0));

-- name: LockEntityStatusCatalogShared :exec
SELECT pg_advisory_xact_lock_shared(hashtextextended(sqlc.arg('workspace_id')::uuid::text || ':entity_status:' || sqlc.arg('resource_type')::text, 0));

-- name: ListActiveCustomEntityStatusEntries :many
SELECT * FROM entity_status
WHERE workspace_id = sqlc.arg('workspace_id')::uuid
  AND resource_type = sqlc.arg('resource_type')::text
  AND category = sqlc.arg('category')::text
  AND is_system = FALSE
  AND archived_at IS NULL
ORDER BY position, key;

-- name: ReorderEntityStatusEntries :execrows
UPDATE entity_status s
SET position = v.ordinality::int,
    updated_at = now()
FROM unnest(sqlc.arg('ids')::uuid[]) WITH ORDINALITY AS v(id, ordinality)
WHERE s.id = v.id
  AND s.workspace_id = sqlc.arg('workspace_id')::uuid
  AND s.resource_type = sqlc.arg('resource_type')::text
  AND s.is_system = FALSE
  AND s.archived_at IS NULL;
