# AgentMail native integration

Workspace-wide connection, per-agent inbox, claim-time MCP overlay, Settings and agent Email UI. Mail bodies stay at AgentMail. Multica stores connection and inbox metadata plus sealed keys.

This folder is the implementation record. Product copy lives in
`apps/docs/content/docs/agentmail-integration.mdx`.

## What shipped

- Two product tables plus a purge ledger (migrations 459–461)
- Domain service in `server/internal/integrations/agentmail/`
- Live AgentMail `/v0` HTTP client, plus an in-process memory client for tests
- HTTP under `/api/workspaces/{id}/agentmail` and `/api/agents/{id}/agentmail`
- Config flags `agentmail_available` and `agentmail_hosted_available`
- Settings → Email (`?tab=agentmail`) and a first-class agent Email tab
- Chosen username + AgentMail domain (including custom domains) at grant
- Text-only mailbox on the agent Email tab (live fetch, no Postgres mail)
- Claim-time overlay `mcpServers.agentmail` with an inbox-scoped `x-api-key`
- Workspace delete sweep of product rows. Purge rows stay

## What this is not

- Not an MCP-library install of a shared org key
- Not a `channel.Channel` and not `custom_env`
- Not the notification Inbox at `/{slug}/inbox`
- Not a generic integration registry
- Not a mailbox stored in Postgres

## Waiting

Inbound webhooks, a partial unique index on `remote_inbox_id`, pause (keep the address, drop only the overlay), mobile, and landing-page copy. See `NEXT.md`.

## Names

| In the product | Meaning |
| --- | --- |
| Email | The Settings tab and the agent capability tab |
| AgentMail | The vendor. Stays English in every locale |
| Inbox / 收件箱 | The notification Inbox. Do not reuse that word for mail |
