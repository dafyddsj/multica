-- Initiatives are durable workspace containers above time-boxed projects
-- (a product, a service, a program). No FKs: relationships and delete
-- detach are enforced in application transactions.
CREATE TABLE initiative (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    icon TEXT,
    status TEXT NOT NULL DEFAULT 'planned'
        CHECK (status IN ('planned', 'in_progress', 'paused', 'completed', 'cancelled')),
    priority TEXT NOT NULL DEFAULT 'none'
        CHECK (priority IN ('urgent', 'high', 'medium', 'low', 'none')),
    lead_type TEXT CHECK (lead_type IN ('member', 'agent')),
    lead_id UUID,
    start_date DATE,
    due_date DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
