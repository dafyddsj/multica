-- Backing index for entity_status's primary key, attached in 449.
CREATE UNIQUE INDEX CONCURRENTLY entity_status_pkey_uidx
    ON entity_status (id);
