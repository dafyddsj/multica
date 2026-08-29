CREATE INDEX CONCURRENTLY agent_paused_by_budget_id_idx
    ON agent (paused_by_budget_id)
    WHERE paused_by_budget_id IS NOT NULL;
