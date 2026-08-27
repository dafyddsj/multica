# Clerk auth overlay

Multica can use [Clerk](https://clerk.com) for **human web sign-in** when both `CLERK_SECRET_KEY` and `CLERK_PUBLISHABLE_KEY` are set. Either key empty is a no-op: native email OTP and Google stay in effect (same gate shape as `REDIS_URL`).

This is an **overlay**. Native auth is not deleted. Overlay-off, machine tokens, CLI HMAC, desktop, and mobile keep working so upstream merges stay possible.

Product docs for native sign-in stay in `apps/docs/content/docs/auth-setup.mdx`. This folder is the overlay implementation record.

## What shipped

- Env gate: both keys or neither. Half-set does nothing.
- `users.id` and `workspace.id` stay Multica UUIDs. Clerk ids live on `user.clerk_user_id` and `workspace.clerk_org_id`. Never send `user_xxx` as `X-User-ID`.
- Web: `@clerk/nextjs` `ClerkProvider` + `getToken()` as the API bearer. Native cookies are rejected while Clerk is on.
- `POST /api/cli-token` still mints a native HMAC JWT for CLI / desktop handoff.
- Machine prefixes stay native: `mul_`, `mat_`, `mdt_`, `mcn_`, `mpi_`, `mpc_`.
- Workspace create / delete, member role / remove / leave, invite accept, and share-link join dual-write the mapped Clerk org when `clerk_org_id` is set.
- `GET /api/me` pulls Clerk org memberships into Multica (`SyncOrgs`) and refreshes the bound user's **email** from Clerk (`SyncProfile`). `GET /api/workspaces` does not sync.
- Signup allowlists (`ALLOW_SIGNUP`, `ALLOWED_EMAILS`) are not re-checked on the Clerk path. `IsTemporarilyDisabledUser` still is.

## Overlay off vs on

| Surface | Overlay off | Overlay on |
| --- | --- | --- |
| Web login | Email OTP / Google | Clerk `SignIn` |
| `/auth/send-code`, `/auth/verify-code`, `/auth/google` | Live | 404 |
| Human API credential | `multica_auth` cookie or HMAC bearer | Clerk session JWT bearer |
| Invitation email | Multica Resend / SMTP | Same. Do not enable Clerk org-invite emails. |
| Auth emails (verify / magic link) | Multica OTP | Clerk |
| Desktop / Expo | Native | HMAC CLI-token handoff. No Clerk SDK yet. |

## Identity

Clerk is the identity plane for **email** after bind. Name, avatar, language, timezone, and profile blurb stay Multica (`UpdateMe` / Settings → Profile).

`Resolve` (every authenticated request) verifies the JWT and maps to a Multica user. It does **not** call Clerk `Users.Get` on the already-bound path. Email refresh runs on `GetMe` so we do not add a Clerk HTTP call to every API request.

An email change in Clerk is visible in Multica after that person's next `GetMe`. While they are idle, Multica still has the old email. That idle case is a webhook job — see `WEBHOOKS.md`.

## Workspaces and members

Multica is the product control plane. Clerk orgs are a shadow of mapped workspaces.

- Do **not** use Clerk Active Organization as the workspace switcher. Routing stays `/{slug}/…` and `X-Workspace-Slug`.
- Members tab stays Multica: invite, pending list, roles (`owner` / `admin` / `member`), share links, seats.
- Invite create / revoke does not call Clerk. Accept adds the Clerk user to the org.
- Clerk role map: inbound `org:admin` / `org:owner` → Multica owner; `org:member` → member. Local **admin** stays admin when Clerk still reports admin. Outbound, Multica owner and admin both become `org:admin`. Do not add a custom `org:owner` until a dedicated migration exists — today's owners are stored as `org:admin` in Clerk.
- Last owner: Multica handlers refuse to remove them. Inbound `SyncOrgs` also skips deleting the last owner.

### Clerk dashboard

Treat it as out of bounds for day-to-day member work. Dashboard create / add / kick is **pull-only**: it lands when *that person* hits `GetMe`, not when someone else loads the members tab. No Multica invite email, no pending row, no seat check, no `revokeAndRemoveMember` cleanup. Enable **organization slugs** in Clerk or workspace create from Multica fails (`organization_slugs_disabled`).

## Clients still to do

- **Webhooks** — need a public HTTPS origin. Documented in `WEBHOOKS.md`.
- **Electron** — `@clerk/electron` or keep HMAC handoff.
- **Expo** — mobile owns its auth stack.

## Code map

| Area | Path |
| --- | --- |
| Env gate | `server/internal/clerk/env.go`, `apps/web/lib/clerk-env.ts` |
| JWT → Multica user | `server/internal/clerk/resolve.go` |
| Email refresh | `server/internal/clerk/profile.go` |
| Org sync | `server/internal/clerk/sync.go` |
| Roles | `server/internal/clerk/role.go` |
| Handler attach | `server/internal/handler/clerk_orgs.go` |
| HTTP / cookie policy | `server/internal/middleware/clerk_auth.go` |
| Web mount | `apps/web/components/web-providers.tsx`, `apps/web/features/auth/` |
| Migrations | `454`–`457` (`clerk_user_id`, `clerk_org_id` + concurrent indexes) |
