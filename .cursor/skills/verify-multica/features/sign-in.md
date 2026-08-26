# Sign in

Sign in lets a user request a six-digit email code, enter it, and land in Multica (onboarding or their workspace). An already-signed-in user who opens `/login` is sent onward. Log out returns them to the login screen.

## Sub-features

- `signin-email` shows the email form and keeps Continue disabled until an email is entered.
- `signin-code` sends a code and shows the OTP step for that email.
- `signin-accept` accepts the local development code and leaves `/login`.
- `signin-session` sends an already-authenticated visit away from `/login`.
- `signin-logout` returns a signed-in user to `/login`.

## How to get to it (user POV)

- Open `/login` in the web app.
- Open a workspace URL while signed out; the app redirects to `/login`.
- Choose `Log out` in the workspace switcher menu.

## Driving it with the browser

Preconditions:

- `control-multica doctor` exits 0.
- Web URL is the `web` line from `control-multica urls`.
- Sign-in pair is the `sign-in` line from `urls` (typically `dev@localhost` / `888888`).
- Use a browser profile that does not already have `multica_token` unless you are proving `signin-session`.

- **Open login.** Go to `{web}/login`. The page shows `Sign in to Multica` and a textbox named `Email`. The button `Continue` is disabled.
- **Enter email.** Fill the `Email` textbox with the urls email. `Continue` becomes enabled.
- **Send code.** Choose `Continue`. The heading becomes `Check your email` and the description includes the address you typed.
- **Enter code.** Focus `[data-slot="input-otp"]` (hidden textbox) and type the six-digit urls code. The form submits when the sixth digit is entered. Do not look for a separate submit button.
- **Landed.** The URL is no longer `/login`. First-time users see `Continue on web` (onboarding). A user who has already onboarded in this database lands on `/{slug}/issues` with a `New Issue` button, or another post-auth destination. Document title contains `Multica`.
- **Session bounce.** While still signed in, go to `{web}/login` again. The app leaves `/login` without asking for a new code.
- **Log out.** From a workspace page, open the workspace-name button in the sidebar, choose menuitem `Log out`. The URL is `/login` and `Sign in to Multica` is visible.
- **Proof.** Save `{artifacts}/sign-in/email-step.png` (title visible), `{artifacts}/sign-in/code-step.png` (email visible in the description), `{artifacts}/sign-in/after-login.png` plus an ARIA snapshot of the landed page. The after-login artifacts must not show the email form.

## Gotchas

- The OTP widget is one hidden textbox, not six separately named fields. Type the whole code into that textbox.
- `888888` works only when `APP_ENV` is not `production`. Doctor does not check this; a rejected code with copy `Invalid or expired code` means the environment is not using the local override.
- `dev@localhost` is shared across runs of this checkout. It may already be onboarded. That does not fail sign-in; it changes the landing page. Record which landing you got.
- Injecting `localStorage.multica_token` proves session restore, not sign-in. Do not use it for `signin-email` / `signin-code` / `signin-accept`.
- Google sign-in (`Continue with Google`) needs OAuth client config. Skip it unless this checkout's env has `GOOGLE_CLIENT_ID` and you intend to complete the real Google UI.
