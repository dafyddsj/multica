CREATE UNIQUE INDEX CONCURRENTLY budget_workspace_scope_owner_uidx ON budget (workspace_id, scope, owner_id);
