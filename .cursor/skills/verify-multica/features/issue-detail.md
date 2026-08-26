# Issue detail

Issue detail shows a saved issue's title, properties, and a back path to the issues list. The browser tab names the issue so several open issues stay distinguishable.

## Sub-features

- `detail-open-list` opens an issue from the issues list or board.
- `detail-properties` shows the `Properties` region.
- `detail-title` shows the issue title and sets `document.title` to `{identifier}: {title} | Multica`.
- `detail-back` returns to the issues list via the Issues breadcrumb or sidebar.

## How to get to it (user POV)

- Choose an issue title or card on `/{slug}/issues`.
- Open `/{slug}/issues/{id}` directly (reload, bookmark, toast `View issue`).

## Driving it with the browser

Preconditions:

- `control-multica doctor` exits 0.
- Signed in on `/{slug}/issues`.
- At least one issue exists. If the workspace is empty, create one with [create-issue](./create-issue.md) first (user path), or this feature is blocked.

- **Find a row.** On `/{slug}/issues`, locate a visible issue title. Board columns include `Backlog`, `Todo`, `In Progress`. List view is the `List` control if cards are collapsed.
- **Open from list.** Choose the issue's link (href ends with `/issues/{id}`). The URL matches `/{slug}/issues/{id}`.
- **Read properties.** Text `Properties` is visible. The issue title from the list is visible on the page.
- **Read tab title.** `document.title` is `{identifier}: {title} | Multica` (identifier looks like `ABC-123`).
- **Back to list.** Choose the breadcrumb or sidebar link `Issues` (exact). The URL matches `/{slug}/issues` and `New Issue` is visible.
- **Direct URL.** Open the same `/{slug}/issues/{id}` again. Properties and title still match.
- **Proof.** Save `{artifacts}/issue-detail/open.png` (Properties + title), `{artifacts}/issue-detail/open.aria.yml`, and record the observed `document.title`.

## Gotchas

- Board cards can be off-screen. Switch to `List` before declaring the issue missing.
- Title-only screenshots that omit `Properties` do not prove you are on the detail page (the list also shows titles).
- Creating the issue via API and only opening the URL proves navigation, not create. That is allowed here if create-issue was already proven or is out of scope for this run — say which.
