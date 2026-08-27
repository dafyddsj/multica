ALTER TABLE agent
    DROP COLUMN IF EXISTS lightweight_model,
    DROP COLUMN IF EXISTS lightweight_thinking_level,
    DROP COLUMN IF EXISTS start_lightweight,
    DROP COLUMN IF EXISTS failover_runtime_id,
    DROP COLUMN IF EXISTS failover_model,
    DROP COLUMN IF EXISTS failover_thinking_level,
    DROP COLUMN IF EXISTS failover_service_tier;
