CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_memory_entry_user_scope
    ON memory_entry (owner_id, created_at DESC)
    WHERE deleted_at IS NULL AND scope = 'user';
