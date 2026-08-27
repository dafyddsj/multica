CREATE UNIQUE INDEX CONCURRENTLY entity_status_workspace_type_name_uidx
    ON entity_status (workspace_id, resource_type, lower(name))
    WHERE archived_at IS NULL;
