-- A project belongs to at most one initiative. No FK: delete detaches
-- (NULL) in the initiative-delete transaction rather than cascading.
ALTER TABLE project ADD COLUMN initiative_id UUID;
