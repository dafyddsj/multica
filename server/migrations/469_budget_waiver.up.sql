-- Time-bounded resource-scope carve-out. Overlap is refused in the service.
CREATE TABLE budget_waiver (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    scope TEXT NOT NULL CHECK (scope IN ('project', 'initiative')),
    owner_id UUID NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ NOT NULL,
    created_by UUID NOT NULL,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (ends_at > starts_at)
);
