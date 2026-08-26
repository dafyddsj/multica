-- Optional extra Co-authored-by email for this agent's commits.
-- NULL means no extra trailer. The workspace GitHub toggle still
-- controls the shared Multica trailer.
ALTER TABLE agent ADD COLUMN co_authored_by_email TEXT;
