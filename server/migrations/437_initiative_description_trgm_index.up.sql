CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_initiative_description_trgm
    ON initiative USING gin (LOWER(COALESCE(description, '')) gin_trgm_ops);
