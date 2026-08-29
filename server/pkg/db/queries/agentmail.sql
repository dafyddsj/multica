-- =====================
-- AgentMail connection
-- =====================

-- name: GetAgentMailConnectionByWorkspace :one
SELECT * FROM agentmail_connection
WHERE workspace_id = $1;

-- name: UpsertAgentMailConnection :one
INSERT INTO agentmail_connection (
    workspace_id, source, state, authority_kind, pod_id, org_key_encrypted,
    pod_client_id, domain, connected_by_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, sqlc.narg('connected_by_id')
)
ON CONFLICT (workspace_id) DO UPDATE SET
    source            = EXCLUDED.source,
    state             = EXCLUDED.state,
    authority_kind    = EXCLUDED.authority_kind,
    pod_id            = EXCLUDED.pod_id,
    org_key_encrypted = EXCLUDED.org_key_encrypted,
    pod_client_id     = EXCLUDED.pod_client_id,
    domain            = EXCLUDED.domain,
    connected_by_id   = EXCLUDED.connected_by_id,
    updated_at        = now()
RETURNING *;

-- name: DeleteAgentMailConnectionByWorkspace :exec
DELETE FROM agentmail_connection WHERE workspace_id = $1;

-- =====================
-- AgentMail inbox
-- =====================

-- name: GetAgentMailInbox :one
SELECT * FROM agentmail_inbox
WHERE workspace_id = $1 AND agent_id = $2;

-- name: ListAgentMailInboxesByWorkspace :many
SELECT * FROM agentmail_inbox
WHERE workspace_id = $1
ORDER BY created_at ASC;

-- name: CountAgentMailInboxesInFlight :one
-- Hosted quota: a slot is consumed for as long as the remote inbox may exist.
SELECT count(*) FROM agentmail_inbox
WHERE workspace_id = $1
  AND state IN ('provisioning', 'minting_key', 'active', 'disabling');

-- name: UpsertAgentMailInbox :one
INSERT INTO agentmail_inbox (
    workspace_id, agent_id, client_id, state, remote_inbox_id, address,
    display_name, inbox_key_encrypted, key_attempt_id, created_by_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, sqlc.narg('created_by_id')
)
ON CONFLICT (workspace_id, agent_id) DO UPDATE SET
    client_id           = EXCLUDED.client_id,
    state               = EXCLUDED.state,
    remote_inbox_id     = EXCLUDED.remote_inbox_id,
    address             = EXCLUDED.address,
    display_name        = EXCLUDED.display_name,
    inbox_key_encrypted = EXCLUDED.inbox_key_encrypted,
    key_attempt_id      = EXCLUDED.key_attempt_id,
    created_by_id       = EXCLUDED.created_by_id,
    updated_at          = now()
RETURNING *;

-- name: DeleteAgentMailInbox :exec
DELETE FROM agentmail_inbox
WHERE workspace_id = $1 AND agent_id = $2;

-- name: DeleteAgentMailInboxesByWorkspace :exec
DELETE FROM agentmail_inbox WHERE workspace_id = $1;

-- name: GetAgentMailActiveInboxKey :one
-- Claim path: inbox ciphertext only. The org column is not in this projection.
SELECT i.inbox_key_encrypted
FROM agentmail_inbox i
JOIN agentmail_connection c ON c.workspace_id = i.workspace_id
WHERE i.workspace_id = $1
  AND i.agent_id = $2
  AND i.state = 'active'
  AND c.state = 'active';

-- =====================
-- AgentMail purge ledger
-- =====================

-- name: InsertAgentMailPurge :one
INSERT INTO agentmail_purge (
    workspace_id, kind, remote_id, source, org_key_encrypted
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: ListAgentMailPurgesByWorkspace :many
SELECT * FROM agentmail_purge
WHERE workspace_id = $1
ORDER BY created_at ASC;

-- name: DeleteAgentMailPurge :exec
DELETE FROM agentmail_purge WHERE id = $1;
