# Sidebar navigation

Sidebar navigation moves a signed-in user between Inbox, Issues, Agents, and Settings. Each destination updates the URL and the browser tab title.

## Sub-features

- `nav-inbox` opens Inbox (`/{slug}/inbox`, title `Inbox | Multica`).
- `nav-issues` opens Issues (`/{slug}/issues`, title `Issues | Multica`).
- `nav-agents` opens Agents (`/{slug}/agents`, title `Agents | Multica`).
- `nav-settings` opens Settings (`/{slug}/settings`) with tabs `General` and `Members`.

## How to get to it (user POV)

- Choose the matching sidebar link while signed in to a workspace.

## Driving it with the browser

Preconditions:

- `control-multica doctor` exits 0.
- Signed in. Any workspace-scoped page is fine (`/{slug}/issues` is the usual start).

- **Inbox.** Choose link `Inbox`. URL matches `/inbox`. Visible text includes `Inbox`. Document title is `Inbox | Multica`.
- **Agents.** Choose link `Agents`. URL matches `/agents`. Visible text includes `Agents`. Document title is `Agents | Multica`.
- **Issues.** Choose link `Issues` (exact — not `My issues`). URL matches `/issues`. `New Issue` is visible. Document title is `Issues | Multica`.
- **Settings.** Choose link `Settings` (exact). URL matches `/settings`. Tabs `General` and `Members` are visible.
- **Proof.** Save one screenshot per destination under `{artifacts}/sidebar-navigation/` (`inbox.png`, `agents.png`, `issues.png`, `settings.png`) with the sidebar selection and page heading visible. Record each URL and document title.

## Gotchas

- `Issues` vs `My issues` are different links. The exact name `Issues` is the board/list home.
- `Settings` vs other settings-adjacent items: use the exact sidebar link, then assert `/settings` and the `General` tab.
- A URL change without the matching document title is incomplete proof (tab titles are how several open destinations stay distinguishable).
