-- Single statement: CREATE INDEX CONCURRENTLY cannot run inside a transaction
-- or share a multi-command migration file.
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_agentmail_purge_workspace
    ON agentmail_purge (workspace_id);
