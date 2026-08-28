# Waiting AgentMail slices

Work that was designed and then parked. Do not restart architect unless
someone contests the shape. Do not implement MCP-only, `channel.Channel`,
or `custom_env` shortcuts.

## Inbound webhook

AgentMail can POST when mail arrives. v1 has no inbound route. Humans and
agents discover mail by opening the Email tab or by using the claim-time MCP
tools.

A later slice should add a workspace or inbox-scoped webhook, verify it, and
fan out to the right agent. That handler looks up Multica state by AgentMail's
inbox id, not by our agent UUID. Build the unique index in the same PR as the
first reader of that id.

## Unique index on `remote_inbox_id`

Grant, revoke, claim, and the viewer all go through
`UNIQUE (workspace_id, agent_id)`. They do not need this index.

A webhook payload names AgentMail's inbox id (`inb_…`, sometimes the
address). The invert is `WHERE remote_inbox_id = $1`. A unique index makes
that a key lookup and enforces one AgentMail inbox to one Multica row. Two
agents or two workspaces that share a BYO key cannot both own the same remote
inbox. Without that, the webhook cannot pick a workspace or an agent.

Do not add a naive `UNIQUE (remote_inbox_id)` on the whole column.

- `remote_inbox_id` is NULL in `provisioning`. Postgres treats NULLs as
  distinct, so empty rows are fine.
- A `disabled` row may still hold the old id after revoke. A full-table
  unique index would block a *different* agent from taking that id if
  AgentMail reused it. Re-grant of the *same* agent is an upsert of the same
  `(workspace_id, agent_id)` row, so that path is fine.

The index that matches the webhook invariant is partial:

```sql
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_agentmail_inbox_remote
    ON agentmail_inbox (remote_inbox_id)
    WHERE remote_inbox_id IS NOT NULL
      AND state IN ('minting_key', 'active', 'disabling');
```

Keep it in its own single-statement migration. Concurrent indexes cannot
share a file with other SQL. Decide in that PR whether `disabled` should
clear `remote_inbox_id`. Clearing makes reuse obvious. Keeping it is a
breadcrumb for the last known remote id and fights a full unique constraint.

## Pause

Out of this work. The later rule, if we keep it, is keep the address and drop
only the claim overlay while the agent is paused. Do not revoke on pause.
Revoke deletes the remote inbox and spends hosted quota to get the address
back.

## Archive

Restore exists, so archive does not revoke. Many archived agents can hold
in-flight hosted inboxes and leak the cap of 5. That is a known gap. Do not
"fix" it by revoking on archive unless product drops Restore of the address.

## Mobile

Mobile shares `@multica/core` types and pure functions only. It does not get
the Settings or agent Email UI in this work. Product semantics (counts,
permissions, identity) must match if a later mobile slice adds Email.

## Landing copy

Settings and the agent Email tab are in-app. They are not landing-page
features. Skip `apps/web/features/landing/i18n/` until someone puts Email on
the marketing list.

## Viewer follow-ups

The shipped viewer is text-only, owner and workspace owner/admin, live fetch.
It does not send, reply, download attachments, or render HTML. Agents send
through the MCP overlay. Do not add `dangerouslySetInnerHTML` for AgentMail
HTML.

## Defaults already applied

These were left open at Agree. The build picked them. Contest them in product,
not by forking the service.

1. Hosted when the org key is set. BYO is still accepted on `Connect`.
2. Hosted per-workspace inbox cap is 5.
3. Pause is out of scope, as above.
