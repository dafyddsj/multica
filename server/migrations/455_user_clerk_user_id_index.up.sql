CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_user_clerk_user_id
    ON "user" (clerk_user_id)
    WHERE clerk_user_id IS NOT NULL;
