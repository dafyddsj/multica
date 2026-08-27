CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_memory_entry_workspace_scope_owner
    ON memory_entry (workspace_id, scope, owner_id, created_at DESC)
    WHERE deleted_at IS NULL AND workspace_id IS NOT NULL;
