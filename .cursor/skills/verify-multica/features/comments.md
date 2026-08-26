# Comments

Comments lets a user leave a note on an issue and see it in the issue's activity. An empty composer cannot send.

## Sub-features

- `comment-open` activates the composer from its idle shell.
- `comment-submit` publishes a non-empty comment with `ControlOrMeta+Enter`.
- `comment-visible` shows the new comment body on the same issue page.
- `comment-empty` keeps send disabled while the composer has no text.

## How to get to it (user POV)

- Open any issue detail page and use the comment composer at the bottom of the thread.

## Driving it with the browser

Preconditions:

- `control-multica doctor` exits 0.
- Signed in on an issue detail URL (`/{slug}/issues/{id}`) with `Properties` visible. Use an issue created in this run (`verify-…`) when possible.

- **See shell.** `[data-testid="comment-composer-shell"]` is visible. The send control (arrow-up button in that composer) is disabled while empty.
- **Activate.** Choose the shell. A `.ProseMirror` editor with placeholder `Leave a comment...` is focused.
- **Type.** Enter `verify-comment-<timestamp>`.
- **Send.** Press `ControlOrMeta+Enter`. The page shows `verify-comment-<timestamp>` in the activity/thread, not only inside the editor.
- **Confirm persistence.** Reload the same issue URL. The comment body is still visible after `Properties` appears.
- **Proof.** Save `{artifacts}/comments/before.png` (shell visible, empty), `{artifacts}/comments/after.png` (body visible in the thread), and an ARIA snapshot that includes the comment text.

## Gotchas

- The composer is a static shell until clicked. Typing before activating it does not count.
- Submit is `ControlOrMeta+Enter`, not a required click on the arrow. Either path is valid; record which you used.
- A string that exists only in the still-focused editor is not published. Reload or blur and read the thread.
- Do not insert the comment with `POST /api/issues/{id}/comments` and call this feature proven.
