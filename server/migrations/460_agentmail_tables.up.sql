-- AgentMail integration: workspace connection + per-agent inbox, plus a
-- remote-cleanup ledger. Two product rows, not booleans on workspace/agent.
-- NO foreign keys and NO cascades: DeleteWorkspace / DeleteAgent sweep the
-- product rows in application transactions (SweepWorkspace / SweepAgent).
-- The purge ledger is unfinished remote cleanup, not a surviving mailbox.
--
-- Key custody:
--   * hosted: the org key is process env and never appears in any row.
--   * bring_your_own: the workspace's own org or pod key, secretbox-sealed.
--   * per-inbox: the inbox-scoped AgentMail key, sealed with the same box.
--     This is the only credential that ever reaches an agent runtime.
--
-- Per the project migration rules these tables carry NO foreign keys or
-- cascades: relationships and dependent cleanup are resolved in application
-- code. The inline UNIQUE / PRIMARY KEY constraints stay — they back the
-- ON CONFLICT upsert targets in agentmail.sql. Secondary indexes live in
-- follow-up single-statement CREATE INDEX CONCURRENTLY migrations (460-461),
-- which cannot share a file with these CREATE TABLEs.

CREATE TABLE IF NOT EXISTS agentmail_connection (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id      UUID NOT NULL,
    source            TEXT NOT NULL
        CHECK (source IN ('hosted', 'bring_your_own')),
    state             TEXT NOT NULL
        CHECK (state IN ('provisioning', 'active', 'disabling', 'disabled')),
    authority_kind    TEXT NOT NULL
        CHECK (authority_kind IN ('hosted_org', 'byo_org', 'byo_pod')),
    pod_id            TEXT,
    org_key_encrypted TEXT,
    pod_client_id     TEXT NOT NULL,
    domain            TEXT NOT NULL DEFAULT '',
    connected_by_id   UUID,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_id),
    CONSTRAINT agentmail_connection_source_fields CHECK (
        (source = 'hosted'
            AND org_key_encrypted IS NULL
            AND authority_kind = 'hosted_org')
        OR
        (source = 'bring_your_own'
            AND authority_kind IN ('byo_org', 'byo_pod')
            AND (org_key_encrypted IS NOT NULL OR state IN ('disabling', 'disabled')))
    ),
    CONSTRAINT agentmail_connection_hosted_active_pod CHECK (
        NOT (state = 'active' AND source = 'hosted')
        OR (pod_id IS NOT NULL AND pod_id <> '')
    )
);

CREATE TABLE IF NOT EXISTS agentmail_inbox (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id        UUID NOT NULL,
    agent_id            UUID NOT NULL,
    client_id           TEXT NOT NULL,
    state               TEXT NOT NULL
        CHECK (state IN ('provisioning', 'minting_key', 'active', 'disabling', 'disabled')),
    remote_inbox_id     TEXT,
    address             TEXT,
    display_name        TEXT NOT NULL DEFAULT '',
    inbox_key_encrypted TEXT,
    key_attempt_id      UUID,
    created_by_id       UUID,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, agent_id),
    CONSTRAINT agentmail_inbox_state_fields CHECK (
        (state = 'provisioning'
            AND remote_inbox_id IS NULL
            AND address IS NULL
            AND inbox_key_encrypted IS NULL
            AND key_attempt_id IS NULL)
        OR
        (state = 'minting_key'
            AND remote_inbox_id IS NOT NULL
            AND address IS NOT NULL
            AND inbox_key_encrypted IS NULL
            AND key_attempt_id IS NOT NULL)
        OR
        (state = 'active'
            AND remote_inbox_id IS NOT NULL
            AND address IS NOT NULL
            AND inbox_key_encrypted IS NOT NULL
            AND key_attempt_id IS NULL)
        OR
        (state = 'disabling'
            AND inbox_key_encrypted IS NULL)
        OR
        (state = 'disabled'
            AND inbox_key_encrypted IS NULL
            AND key_attempt_id IS NULL)
    )
);

CREATE TABLE IF NOT EXISTS agentmail_purge (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id      UUID NOT NULL,
    kind              TEXT NOT NULL
        CHECK (kind IN ('inbox', 'pod')),
    remote_id         TEXT NOT NULL,
    source            TEXT NOT NULL
        CHECK (source IN ('hosted', 'bring_your_own')),
    org_key_encrypted TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
