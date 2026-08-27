-- Restoring the enum constraints FAILS if any row sits on a custom status.
-- Migrate those rows back to built-ins first.
ALTER TABLE project DROP CONSTRAINT IF EXISTS project_status_format_check;

ALTER TABLE project
    ADD CONSTRAINT project_status_check
    CHECK (status IN ('planned', 'in_progress', 'paused', 'completed', 'cancelled'));

ALTER TABLE initiative DROP CONSTRAINT IF EXISTS initiative_status_format_check;

ALTER TABLE initiative
    ADD CONSTRAINT initiative_status_check
    CHECK (status IN ('planned', 'in_progress', 'paused', 'completed', 'cancelled'));
