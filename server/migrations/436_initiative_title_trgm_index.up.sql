CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_initiative_title_trgm
    ON initiative USING gin (LOWER(title) gin_trgm_ops);
