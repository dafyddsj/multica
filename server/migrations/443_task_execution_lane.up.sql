ALTER TABLE agent_task_queue
    ADD COLUMN IF NOT EXISTS execution_lane TEXT NOT NULL DEFAULT 'primary',
    ADD COLUMN IF NOT EXISTS model_override TEXT;
