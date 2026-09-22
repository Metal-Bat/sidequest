# Verification

Verified locally with Go 1.27.1 and an isolated PostgreSQL 18.1 container, without
mounting or changing the existing personal database volume.

- `go test -race ./...` with `TEST_DATABASE_URL`: passed, including separate-user
  ownership, time selection/reroll, migrations applied twice, twelve concurrent
  completions awarding XP once, deletion retaining XP, and the HTTP auth/quest journey.
- `go vet ./...` and `golangci-lint run ./...`: passed (zero lint issues).
- Native binary build and its explicit migration command: passed.
- Backend Docker image build and `docker compose config --quiet`: passed.
- Air launched from the repository root: built and started successfully. Both
  `bin` (installed Air 1.63) and `entrypoint` settings point into `src/tmp`.
- Chromium: registration, HTMX creation, selection/reroll, acceptance, completion,
  level update, duplicate-ID check, history deletion and logout passed.
- Chromium at 375px width with reduced motion: no horizontal document overflow.
  Screenshot inspection caught and fixed SVG center fills obscuring the parchment.
- HTTP tests checked non-JavaScript form fallbacks, inline validation retaining
  input, escaped titles, runtime asset URLs, excluded prototype files, CSRF rejection
  and expired-session HTMX redirects.

Screenshots: [desktop](screenshots/desktop.png), [mobile](screenshots/mobile.png).
The original prototype remains in `src/static/index.html`.

Limits: browser testing used Chromium only. Keyboard focus styles and semantic
labels were reviewed in source; no screen-reader session was performed. Compose
configuration and backend build were checked, but the existing persistent database
stack was deliberately not started or modified. Authentication rate limits are
in-memory and per process; the ledger currently loads all of a user's quests.
