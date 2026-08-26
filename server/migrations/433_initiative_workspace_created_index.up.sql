CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_initiative_workspace_created
    ON initiative (workspace_id, created_at DESC);
