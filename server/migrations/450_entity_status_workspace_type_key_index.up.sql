CREATE UNIQUE INDEX CONCURRENTLY entity_status_workspace_type_key_uidx
    ON entity_status (workspace_id, resource_type, key);
