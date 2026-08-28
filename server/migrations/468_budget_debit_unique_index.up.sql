CREATE UNIQUE INDEX CONCURRENTLY budget_debit_budget_task_provider_model_uidx ON budget_debit (budget_id, task_id, provider, model);
