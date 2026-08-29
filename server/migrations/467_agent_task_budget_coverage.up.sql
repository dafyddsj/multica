ALTER TABLE agent_task_queue
    ADD COLUMN IF NOT EXISTS budget_project_id UUID,
    ADD COLUMN IF NOT EXISTS budget_initiative_id UUID,
    ADD COLUMN IF NOT EXISTS budget_origin_squad_id UUID;
