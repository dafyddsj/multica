# Multica web verification map

This directory is the maintained source for verifying user-facing behavior of the Multica **web** app in this checkout. Read the index before driving, then use the matching feature file as the recipe.

## Baseline preconditions

- `control-multica prereqs` exits 0 (`go`, `docker`, `node`, `pnpm` on PATH).
- This checkout's environment is healthy. Run `.cursor/skills/verify-multica/control-multica doctor` and require matching `dir`, running `api` + `web`, and `/health` commit equal to `git rev-parse --short HEAD`.
- Web URL is the `web` line from `control-multica urls`. Never drive `localhost:3000` unless doctor printed that port for **this** repo.
- Local sign-in uses the email/code pair from `control-multica urls` (`dev@localhost` / `888888` unless the env file overrides them). `APP_ENV` must not be `production` or the fixed code is ignored.
- Never drive an instance doctor did not accept.
- Playwright `e2e/` may be running against the same origin. Prefer a dedicated browser tab. Do not share a page Playwright is mid-test on.

## Driving conventions

- Start every recipe from the baseline unless its preconditions say otherwise.
- Prefer ARIA roles and accessible names. Stable test ids in this repo are rare; `comment-composer-shell` is one.
- Treat every command as literal. Keep quoted names unchanged.
- Browser actions go through the Cursor browser tools against the doctor URL.
- After a mutation, reopen or reload before calling it persisted.
- Prefix created issue titles with `verify-` and a timestamp. Do not delete unrelated issues on cleanup.

## Proof and skip reporting

- Capture the user action and the resulting state, not only the final screen.
- UI proof includes an ARIA snapshot and a screenshot with Multica identity visible (login copy, workspace name, or document title `… | Multica`).
- Mutation proof includes a second user-facing read of the stored value.
- Record the feature ID and entry point on every artifact.
- Report an unreachable path with the attempted action and the unmet precondition.
- Do not report a skipped entry point as verified through a different path.

## Feature entry contract

Each feature file starts with an H1 title and one paragraph describing the user-visible behavior. It then uses exactly four H2 sections in this order.

1. `Sub-features` lists short IDs with one line for each behavior.
2. `How to get to it (user POV)` lists every user entry point.
3. `Driving it with the browser` starts with `Preconditions:` and uses labeled bullets that pair each user action with an exact handle and observable result.
4. `Gotchas` lists traps that can waste or invalidate a verification run.

Keep implementation details out of the map. Name only user paths, stable handles, required state, commands, and observable proof.

## Features

- [Sign in](./sign-in.md) covers the email + code path, already-signed-in bounce, and logout.
- [Create an issue](./create-issue.md) covers opening the composer, switching to manual, saving, cancelling, and reopening.
- [Issue detail](./issue-detail.md) covers opening an issue from the list and reading title, properties, and document title.
- [Comments](./comments.md) covers leaving a comment on an issue and seeing it in activity.
- [Sidebar navigation](./sidebar-navigation.md) covers Inbox, Issues, Agents, and Settings from the signed-in shell.
