-- Per-workspace Initiative and Project status catalog.
--
-- Same model as issue_status (MUL-6243): the 5 built-in keys are also the
-- 5 categories. A custom status names a category and inherits that
-- category's meaning (active vs done) wholesale. `project.status` and
-- `initiative.status` stay TEXT keys — no status_id, no backfill.
--
-- resource_type is 'initiative' or 'project'. Issues keep their own catalog.
--
-- No foreign keys (workspace_id is an application-layer relation). id is
-- NOT an inline PRIMARY KEY: the backing unique index is built CONCURRENTLY
-- in 448 and attached in 449.
CREATE TABLE entity_status (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    resource_type TEXT NOT NULL CHECK (resource_type IN ('initiative', 'project')),

    key TEXT NOT NULL CHECK (key ~ '^[a-z0-9][a-z0-9_]{0,31}$'),
    name TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 AND 64),
    description TEXT NOT NULL DEFAULT '' CHECK (char_length(description) <= 256),

    category TEXT NOT NULL CHECK (
        category IN ('planned', 'in_progress', 'paused', 'completed', 'cancelled')
    ),

    color TEXT NOT NULL CHECK (color ~ '^#[0-9a-f]{6}$'),

    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    position DOUBLE PRECISION NOT NULL DEFAULT 0,
    archived_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT entity_status_system_is_canonical
        CHECK (NOT is_system OR key = category),

    CONSTRAINT entity_status_system_not_archivable
        CHECK (NOT is_system OR archived_at IS NULL)
);
