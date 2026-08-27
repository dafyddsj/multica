ALTER TABLE agent_task_queue
    DROP COLUMN IF EXISTS execution_lane,
    DROP COLUMN IF EXISTS model_override;
