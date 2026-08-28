ALTER TABLE agent_task_queue
    DROP COLUMN IF EXISTS budget_origin_squad_id,
    DROP COLUMN IF EXISTS budget_initiative_id,
    DROP COLUMN IF EXISTS budget_project_id;
