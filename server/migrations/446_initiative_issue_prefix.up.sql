-- Optional per-initiative issue prefix. When set, issues whose project
-- belongs to this initiative render and resolve as PREFIX-number instead of
-- the workspace prefix. Number remains unique per workspace; the prefix is
-- display + lookup, not a second counter.
ALTER TABLE initiative ADD COLUMN IF NOT EXISTS issue_prefix TEXT;
