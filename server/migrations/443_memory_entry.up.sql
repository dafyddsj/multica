-- Multica-owned memory. One row is one pin on an existing object.
-- No FKs: owner existence and delete cleanup live in application code.
CREATE TABLE memory_entry (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID,
    scope TEXT NOT NULL
        CHECK (scope IN ('workspace', 'initiative', 'project', 'issue', 'squad', 'agent', 'user')),
    owner_id UUID NOT NULL,
    body TEXT NOT NULL
        CHECK (char_length(body) BETWEEN 1 AND 16000),
    kind TEXT NOT NULL DEFAULT 'fact'
        CHECK (kind IN ('fact', 'preference', 'procedure', 'observation')),
    provenance JSONB NOT NULL DEFAULT '{}'::jsonb
        CHECK (jsonb_typeof(provenance) = 'object'),
    created_by_type TEXT
        CHECK (created_by_type IS NULL OR created_by_type IN ('member', 'agent')),
    created_by_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT memory_entry_user_scope_workspace
        CHECK ((scope = 'user') = (workspace_id IS NULL))
);
