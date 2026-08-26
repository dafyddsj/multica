CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_project_workspace_initiative
    ON project (workspace_id, initiative_id)
    WHERE initiative_id IS NOT NULL;
