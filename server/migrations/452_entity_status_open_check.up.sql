-- Open project.status and initiative.status to catalog keys.
--
-- Membership validation moves to the application layer (entitystatus.Resolve).
-- A format constraint stays as defense in depth. Added NOT VALID so it takes
-- no table scan under lock; migration 453 validates it.
ALTER TABLE project DROP CONSTRAINT project_status_check;

ALTER TABLE project
    ADD CONSTRAINT project_status_format_check
    CHECK (status ~ '^[a-z0-9][a-z0-9_]{0,31}$')
    NOT VALID;

ALTER TABLE initiative DROP CONSTRAINT initiative_status_check;

ALTER TABLE initiative
    ADD CONSTRAINT initiative_status_format_check
    CHECK (status ~ '^[a-z0-9][a-z0-9_]{0,31}$')
    NOT VALID;
