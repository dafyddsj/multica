-- Workspace spend cap. One row per (workspace, scope, owner).
-- No FKs. Primary key and unique owner index are built CONCURRENTLY.
CREATE TABLE budget (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL,
    scope TEXT NOT NULL CHECK (scope IN ('agent', 'squad', 'project', 'initiative')),
    owner_id UUID NOT NULL,
    period TEXT NOT NULL DEFAULT 'monthly' CHECK (period IN ('monthly')),
    limit_usd_ticks BIGINT NOT NULL CHECK (limit_usd_ticks > 0),
    soften_at_percent SMALLINT CHECK (soften_at_percent IS NULL OR (soften_at_percent BETWEEN 1 AND 100)),
    over_limit TEXT NOT NULL CHECK (over_limit IN ('pause', 'allow')),
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
