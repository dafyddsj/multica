# Clerk webhooks (deferred)

Clerk dashboard actions have no push path today. `GetMe` is a **pull**: it asks Clerk which orgs *this* user belongs to, then upserts those workspaces and memberships. `ListWorkspaces` does not sync.

Until Multica is reachable at a public HTTPS origin, do not wait on `CLERK_WEBHOOK_SECRET`. Localhost and this cloud VM cannot receive Clerk POSTs. A tunnel can fake it; it is not worth the time.

## What is missing without webhooks

These stay invisible in Multica until the affected person signs in and `GetMe` runs — or they never appear if that person never uses the app:

- User created in the Clerk dashboard (no Multica row until first Clerk session / `Resolve`)
- Added to a mapped org in Clerk (no member row until *their* `GetMe`)
- New org created in Clerk (workspace created on the creator's next `GetMe`)
- Removed from an org in Clerk (thin `DeleteMember` on their next `GetMe`; last owner is kept)
- Email changed in Clerk while idle (`SyncProfile` only runs on `GetMe`)
- User deleted in Clerk (Multica user and memberships stay until something else cleans them up)

`GetMe` / `SyncProfile` / `SyncOrgs` should stay after webhooks land. They are the reconcile when a delivery is missed.

## When we have a public origin

1. Add a Clerk webhook endpoint (Svix signature via `CLERK_WEBHOOK_SECRET`).
2. Handle at least: `user.created`, `user.updated`, `user.deleted`, `organization.created`, `organization.deleted`, `organizationMembership.created`, `organizationMembership.updated`, `organizationMembership.deleted`.
3. Drive the **same Multica application paths** as the members tab — seat capacity, `revokeAndRemoveMember`, realtime `member:*` events — not the thin sqlc writes `SyncOrgs` uses for inbound leave.
4. Pre-provisioning a Multica user from `user.created` / membership-created, before they sign in, is a product choice. Default should match today's invite flow: a Multica user exists after Clerk sign-in or after they accept a Multica invite.
5. Keep overlay-off a no-op: empty `CLERK_WEBHOOK_SECRET` means the route rejects or 404s.

## Do not do in the webhook slice

- Delete native OTP, Google, or cookies
- Mount Clerk `<OrganizationProfile>` / invite UI on the members tab
- Switch workspace selection to Clerk Active Organization
- Electron or Expo Clerk SDKs
