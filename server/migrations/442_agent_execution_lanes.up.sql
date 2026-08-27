ALTER TABLE agent
    ADD COLUMN IF NOT EXISTS lightweight_model TEXT,
    ADD COLUMN IF NOT EXISTS lightweight_thinking_level TEXT,
    ADD COLUMN IF NOT EXISTS start_lightweight BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS failover_runtime_id UUID,
    ADD COLUMN IF NOT EXISTS failover_model TEXT,
    ADD COLUMN IF NOT EXISTS failover_thinking_level TEXT,
    ADD COLUMN IF NOT EXISTS failover_service_tier TEXT;
