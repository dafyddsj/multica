---
name: verify-multica
description: Drive this checkout's Multica web app the way a user does — launch, doctor identity, sign in with the local email code, exercise issues — and capture proof. Use for live UI verification, proving a change works, or when a PR asks for a live box.
---

# Verify Multica (web)

Primary surface is the **web app** in this checkout (`apps/web`, Next.js). Desktop (Electron), mobile (Expo), the Go API, and `make cli` exist; do not treat those as this skill's drive target unless a mapped feature says so.

You are writing proof for a later reader. Drive the real UI. Do not call internal setters, `TestApiClient`, or test-only endpoints to claim a user-facing feature works. API/DB reads are allowed as a *second* view of a mutation you already performed in the UI.

## Launch

On PATH you need `go` (module is 1.26), `docker` (Compose starts the shared local Postgres), `node` 22, and `pnpm`. Check with `.cursor/skills/verify-multica/control-multica prereqs`.

From the repo root, or via the helper (preferred):

```bash
.cursor/skills/verify-multica/control-multica launch
```

That is `make up C=api,web` when this checkout is not already healthy. Ready when `control-multica doctor` exits 0.

- Web origin and API port come from `make status --json` (`frontend_port`, `backend_port`). Defaults on a fresh main checkout are often `http://localhost:3000` and `http://localhost:8080`. Worktrees differ. Never hardcode another checkout's ports.
- Local sign-in (non-production): email `dev@localhost` (or `MULTICA_DEV_EMAIL`) and code `888888` (or `MULTICA_DEV_VERIFICATION_CODE` from `.env` / `.env.worktree`). `control-multica urls` prints the pair this environment is using.
- Two checkouts can run side by side. `make up` allocates ports and a database name per directory. If `/health` answers on this checkout's API port with a pid or commit that is not ours, **stop**. Do not reuse, kill, or drive it.
- `make down` stops this environment and keeps the database. `make destroy` drops the database — never use it for verification cleanup.

If this run called `launch` and doctor failed, run `control-multica cleanup` before retrying so ports are not stranded.

## Doctor

Run first whenever anything looks off:

```bash
.cursor/skills/verify-multica/control-multica doctor
```

It is read-only. It must show all of:

- `make status --json` `dir` equals this repo root
- `components.api.state` and `components.web.state` are `running`
- `GET http://localhost:<backend_port>/health` returns JSON whose `commit` equals `git rev-parse --short HEAD`
- `GET http://localhost:<frontend_port>` succeeds

If doctor fails, do not drive. Launch or fix identity first.

## Drive

Harness: the Cursor browser tools against **this checkout's** web URL from `control-multica urls`. Prefer ARIA roles and accessible names. Do not click by coordinates.

Existing Playwright specs under `e2e/` are a regression suite, not a substitute for this skill. They often inject `multica_token` via `TestApiClient` and skip the email-code path. Use them only when a feature file says a scripted replay is allowed *in addition* to the user path.

Stable handles from this repo:

| Control | Handle |
|---|---|
| Login title | text `Sign in to Multica` |
| Email | role `textbox`, name `Email` (placeholder `you@example.com`) |
| Send code | role `button`, name `Continue` (disabled while email is empty) |
| Code step | heading `Check your email`; six-digit OTP at `[data-slot="input-otp"]` (hidden textbox; typing six digits submits) |
| Issues home | `/{workspaceSlug}/issues`; role `button`, name `New Issue` |
| Sidebar | role `link`, name `Inbox` / `Issues` (exact) / `Agents` / `Settings` (exact) |
| Create issue | role `textbox`, name `Issue title`; role `button`, name `Create Issue` |
| Agent vs manual create | role `button`, name `Switch to Manual` / `Switch to Agent` |
| Create toast | text `Issue created`; role `button`, name `View issue` |
| Issue detail | text `Properties`; comment shell `[data-testid="comment-composer-shell"]` |
| Comment editor | `.ProseMirror` with placeholder `Leave a comment...`; submit `ControlOrMeta+Enter` |
| Workspace menu | role `button` matching the workspace name; menuitem `Log out` |

Read `.cursor/skills/verify-multica/features/README.md` and the matching feature file before driving. A proof that uses one convenient entry point is incomplete when the map lists others.

Start from `/login` or `/{slug}/issues` as the feature file says. Do not paste `localStorage.multica_token` unless the feature's preconditions explicitly allow a seeded session *after* the user path has already been proven in this run.

## Evidence

Write proof under `.cursor/skills/verify-multica/artifacts/<feature-id>/`. Create the directory with:

```bash
.cursor/skills/verify-multica/control-multica artifacts <feature-id>
```

Standards:

- Exercise the real user path (email → code → UI), not an injected token, unless the feature file is documenting a second view.
- Capture the **action** and the **resulting state**, not only the last screen. For a mutation, take a screenshot or ARIA snapshot before and after, and a second user-facing read (reload, reopen, or a different page that must show the same value).
- Verify side effects that the user would notice: URL, document title, toast, list row, issue heading, comment body. A 200 from `POST /api/issues` is not enough.
- Mocks only at a production boundary the product already isolates (the settings Composio test is an example of that boundary — do not copy the pattern onto issue create).
- Name every artifact with the feature ID and entry point. Example: `artifacts/sign-in/email-step.png`, `artifacts/sign-in/after-code.aria.yml`.

Browser: screenshot + accessibility snapshot with Multica identity visible (login title, workspace name, or `Issues \| Multica` / `Inbox \| Multica` document title). CLI/API second views: command, stdout, stderr, exit code.

## Cleanup

```bash
.cursor/skills/verify-multica/control-multica cleanup
```

- Tears down only an environment **this helper launched** (`launched=1` in `.run-state`). If doctor reused an already-running checkout, cleanup leaves it running.
- Uses `make down` (keeps DB). Never `make destroy`. Never `kill` by process name (`multica`, `next`, `node`).
- Removes `.run-state` only. **Does not delete `artifacts/`.** If cleanup ate the proof, the run failed.

Issue titles created during a run may stay in the shared local DB. Prefix them `verify-` plus a timestamp so a later run can ignore them. Do not delete the user's existing issues.

## Helpers

`control-multica` lives next to this file and is executable.

```bash
.cursor/skills/verify-multica/control-multica prereqs
.cursor/skills/verify-multica/control-multica launch
.cursor/skills/verify-multica/control-multica doctor
.cursor/skills/verify-multica/control-multica urls
.cursor/skills/verify-multica/control-multica artifacts sign-in
.cursor/skills/verify-multica/control-multica cleanup
```

## Other surfaces (out of scope unless mapped)

- Desktop: `make up C=desktop`. Not this skill's default harness.
- Mobile: read `apps/mobile/CLAUDE.md` first. Separate stack.
- API-only: `GET /health` is for doctor, not product proof.
