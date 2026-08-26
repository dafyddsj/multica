# Create an issue

Create issue lets a signed-in user open a composer from the issues home, switch to manual fields, save a titled issue, cancel a draft, and reopen the saved issue from a second view.

## Sub-features

- `create-open` opens the composer from the issues-page `New Issue` button.
- `create-manual` switches the composer from agent mode to manual fields when needed.
- `create-save` persists a title and shows `Issue created`.
- `create-open-result` opens the new issue from the toast.
- `create-cancel` dismisses an unfinished composer with Escape.

## How to get to it (user POV)

- Choose `New Issue` on `/{slug}/issues`.
- Use the command palette command `New Issue` (search).

## Driving it with the browser

Preconditions:

- `control-multica doctor` exits 0.
- You are signed in and on `/{slug}/issues` with a visible `New Issue` button. If you landed on onboarding, finish or use an already-onboarded local user (`dev@localhost` after a prior `make up` login) before this feature.
- No need to pre-create rows. Use a fresh title `verify-create-<timestamp>`.

- **Open composer.** Choose the button `New Issue`. A dialog appears.
- **Switch to manual.** If there is no textbox named `Issue title`, choose `Switch to Manual`. The textbox `Issue title` is visible and `Create Issue` is present (disabled or not ready until the title is non-empty).
- **Enter title.** Fill `Issue title` with `verify-create-<timestamp>`. `Create Issue` becomes ready.
- **Save.** Choose `Create Issue`. A notification region shows `Issue created` and the title you typed.
- **Open result.** Choose `View issue`. The URL matches `/{slug}/issues/{id}` and `Properties` is visible. The heading or title area shows `verify-create-<timestamp>`.
- **Confirm persistence.** Go to `/{slug}/issues` (sidebar link `Issues`, exact). The new title is visible on the board or list. If the board is showing and the card is not in view, switch to `List`.
- **Cancel draft.** Choose `New Issue` again, switch to manual if needed, type `verify-discard-<timestamp>`, press `Escape`. The title textbox is gone. `New Issue` is visible again. The issues view has no `verify-discard-<timestamp>` row.
- **Proof.** Save `{artifacts}/create-issue/composer.png` (Issue title visible), `{artifacts}/create-issue/toast.png` (`Issue created` + title), `{artifacts}/create-issue/detail.png` (Properties + title), and an ARIA snapshot of the issues list that includes the saved title.

## Gotchas

- The composer often opens in **agent** mode first. Manual fields (`Issue title`, `Create Issue`) appear only after `Switch to Manual`. A missing title field is not a product failure until you have switched.
- Toast text `Issue created` alone is insufficient. Reopen from `View issue` or the list.
- The command-palette entry is a second path. If you skip it, say so; do not mark it verified.
- Do not create the issue through `POST /api/issues` or `TestApiClient` and call this feature proven.
