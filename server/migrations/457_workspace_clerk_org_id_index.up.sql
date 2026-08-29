CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_workspace_clerk_org_id
    ON workspace (clerk_org_id)
    WHERE clerk_org_id IS NOT NULL;
