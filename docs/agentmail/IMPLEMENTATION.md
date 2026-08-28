# How AgentMail is implemented

Map from the shipped rules to code. Product voice lives in
`apps/docs/content/docs/agentmail-integration.mdx`. Waiting work lives in
`NEXT.md`.

## Gates

Connect, grant, claim overlay, and the viewer need a secret box. The router
always constructs the service so workspace delete can sweep rows.

| Gate | Default | Where |
| --- | --- | --- |
| `MULTICA_AGENTMAIL_SECRET_KEY` | unset | dedicated `secretbox`. Without it, `Available()` is false and Email stays hidden |
| `MULTICA_AGENTMAIL_ORG_KEY` | unset | process env only. Alias `AGENTMAIL_ORG_API_KEY`. Turns on hosted connect |
| `MULTICA_AGENTMAIL_API_BASE` | `https://api.agentmail.to/v0` | include `/v0`. EU is `https://api.agentmail.eu/v0` |

`/api/config` publishes `agentmail_available` and `agentmail_hosted_available`
with omitempty. The UI treats missing as false (`=== true`).

## Layout

```
docs/agentmail/                              this folder
server/migrations/459–461                    tables + concurrent indexes
server/pkg/db/queries/agentmail.sql          sqlc
server/internal/integrations/agentmail/      service, live client, secrets, overlay
server/internal/handler/integrations_agentmail.go
server/internal/handler/daemon.go            ClaimOverlay after Composio merge
server/internal/handler/workspace.go         delete step "delete agentmail"
packages/core/agentmail/                     query keys
packages/core/api/{client,schemas}.ts        parseWithFallback
packages/views/settings/components/agentmail-tab.tsx
packages/views/agents/components/tabs/email-tab.tsx
apps/docs/content/docs/agentmail-integration.mdx
```

Shared files that already have other owners: `router.go`,
`settings-page.tsx` `WORKSPACE_TAB_KEYS`, `workspace.go` delete steps,
`handler.go`, `packages/core/api/client.ts`.

## Persistence

No foreign keys and no cascades. Application code sweeps.

| Table | Role | Delete |
| --- | --- | --- |
| `agentmail_connection` | Workspace minting authority. One row per workspace | `workspaceDelete` |
| `agentmail_inbox` | Per-agent capability. Join on `workspace_id` only. No `connection_id` | `workspaceDelete` and agent delete |
| `agentmail_purge` | Unfinished remote cleanup. Not a mailbox that survives as a product row | `workspaceDeleteSettle`. KEEP is forbidden because the table has `workspace_id` |

`UNIQUE (workspace_id)` on the connection. `UNIQUE (workspace_id, agent_id)`
on the inbox. Secondary indexes are concurrent, one statement per file
(`agent_id`, `purge.workspace_id`). There is no unique index on
`remote_inbox_id` yet. See `NEXT.md`.

Inbox states: `provisioning`, `minting_key`, `active`, `disabling`,
`disabled`. `active` requires address, remote id, and a sealed inbox key.
`minting_key` owns the one-shot key crash. The next Grant deletes the
unpublished remote inbox, recreates it with the same `client_id`, and mints
again.

## Credentials

Connect is a closed sum: `HostedCredential()` or `ParseBYOCredential(raw)`.
Mode switch without disconnect is `ErrModeConflict` (HTTP 409).

| Mode | Authority | Key storage |
| --- | --- | --- |
| hosted | one AgentMail pod per workspace, `client_id` = workspace UUID | org key stays in process env. Never stored |
| bring your own | `InspectKey` via `GET /v0/auth/me` | sealed org or pod key on the connection. Inbox-scoped BYO is rejected |

BYO org-root does not create a nested pod.

Sealed values use prefixes `agentmail:org:v1:` and `agentmail:inbox:v1:`,
then base64, same pattern as VCS. Inbox key permissions are the AgentMail
whitelist `inbox_read`, `message_read`, `message_send`, `draft_read`,
`draft_create`, `draft_send`. Management scopes stay off that list.

Hosted inbox cap defaults to 5. The count is in-flight states
`provisioning`, `minting_key`, `active`, `disabling`.

## HTTP

Workspace (any member can GET. Writes are owner/admin plus human actor):

- `GET /api/workspaces/{id}/agentmail`
- `POST /api/workspaces/{id}/agentmail` body `{ "mode": "hosted" or "bring_your_own", "org_key"? }`
- `DELETE /api/workspaces/{id}/agentmail`

Agent (GET needs `canViewAgentSecrets`. Writes need `canManageAgent` plus
human. Agents are denied):

- `GET /api/agents/{id}/agentmail`
- `PUT /api/agents/{id}/agentmail`
- `DELETE /api/agents/{id}/agentmail`
- `GET /api/agents/{id}/agentmail/threads`
- `GET /api/agents/{id}/agentmail/threads/{threadId}`

Public views have no pod id and no keys. Address is only on `active`.
Thread responses carry text, not HTML. The live client drops `html` and
`extracted_html` and picks `text`, then `extracted_text`, then `preview`.

Members see roster addresses on Settings GET. They do not see the agent
Email tab.

## Claim

`buildClaimedTaskResponse` calls `ClaimOverlay` after the Composio
`runtime_mcp_overlay` merge, then `mergeMCPOverlay`. SQL projects only
`inbox_key_encrypted` from an active inbox joined to an active connection.
The overlay name `agentmail` wins collisions. Overlay failures are logged.
They do not fail the claim. The overlay is never written to the task row.

Default MCP URL is `https://mcp.agentmail.to/mcp`.

## UI

Settings chrome matches GitHub. Tab key `agentmail`, copy says Email. Hidden
unless `configStore.agentmailAvailable`. A direct `?tab=agentmail` while
hidden falls back to workspace General.

The agent Email tab is a capability tab, not the IM Integrations tab. Visible
only when `agentmailAvailable && canEdit`. If the workspace is not connected,
the tab links to `{paths.settings()}?tab=agentmail`.

Mobile is out of scope. Mobile may import types from `@multica/core` later.
It does not share this UI.

## Tests

```bash
cd server && go test ./internal/integrations/agentmail/ -count=1
cd server && go test ./internal/handler/ -count=1 -run 'AgentMail|GetConfigExposesAgentMail|DeleteWorkspace_SweepsAgentMail|ClaimTaskByRuntime_CarriesAgentMail'
pnpm exec vitest run api/schemas.test.ts
pnpm exec vitest run settings/components/settings-page.test.tsx settings/components/agentmail-tab.test.tsx
pnpm exec vitest run agents/components/agent-overview-pane.test.tsx agents/components/tabs/email-tab.test.tsx
```

Service tests use `NewMemory`. Production `New` attaches `liveClient`.
Handler tests that decode a revoke GET must use a fresh
`AgentMailInboxResponse`. `omitempty` does not clear a reused struct.
