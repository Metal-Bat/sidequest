# Sidequest implementation prompt

Extend this existing repository into a small production-quality fantasy web app called **Sidequest** using:

- Go
- `net/http`
- PostgreSQL
- `github.com/jackc/pgx/v5`
- `html/template`
- HTMX
- plain CSS
- very few dependencies
- no frontend framework
- no REST API unless truly necessary

The goal is to showcase how fast and simple a server-rendered Go + HTMX app can be.

## Product concept

Sidequest is a fantasy-themed personal task manager.

The main idea is:

> “I have some free time. Give me a quest.”

Users can register/login, create sidequests, estimate how long they take, assign difficulty, complete them, earn XP, and ask the app to choose a suitable quest based on available time.

The app should feel like a fantasy adventurer guild / quest journal, but remain lightweight and fast.

## Existing repository: inspect first, preserve its structure

This is an existing scaffold, not a greenfield repository. Keep the Go module inside `src/`; do not move it to the repository root or introduce `cmd/sidequest` just to follow a generic layout.

Observed starting state (recheck before editing):

- `src/go.mod`: module `github.com/Metal-Bat/sidequest`, Go directive `1.27.1`.
- `src/main.go`: only `package cmd`; no working server yet. Turn this into `package main` with the application entry point.
- `src/static/index.html`: fantasy quest-board prototype with placeholder player/quest data and HTMX attributes; currently loads HTMX 2.0.8 from a CDN.
- `src/static/sidequest.css`: existing parchment, warm background, serif typography, gold accents, responsive styles, and fantasy borders.
- Actual selected assets: `src/static/assets/{guild-frame,quest-frame,button-frame,small-frame}.svg`.
- The CSS currently references `assets/kenney/`, which does not match those files. Fix URLs to use the actual assets; do not assume a Kenney subdirectory exists.
- `src/static/README.md` describes the prototype, but some paths and license-file claims are stale. Neither of the `LICENSE-KENNEY.txt` files mentioned there/in editor tabs is present on disk at inspection time. Verify provenance and restore the appropriate license notice from the original source if needed; do not invent license text or claim a missing file exists.
- Root `.mise.toml` already defines format, lint, vet, build, test, race, coverage, and other tasks, mostly with `dir = "src"`. Extend this file; do not create another `mise.toml`.
- Root `.air.toml` already targets `src/`; check build and executable paths from the actual launch directory and watch CSS/SVG changes too.
- Root `docker-compose.yml` and `compose/{backend,postgres,pgadmin}/` already exist. The backend references undefined `otel-collector` and `redis` services. Remove these stale dependencies; this app does not need those services.
- The backend Dockerfile pins Go 1.25.11, unlike the module/mise configuration, and references a not-yet-present `src/go.sum`. Reconcile the toolchain deliberately and make the build work after dependencies are added. Do not silently choose a different version or claim the configured version is available without checking.
- `.envs/` is used by Compose. Preserve private values and provide safe example configuration without committing secrets.
- Root `README.md` is currently minimal; `.pre-commit-config.yaml`, `.gitignore`, and editor configuration already exist.

Use `src/static/index.html` and `src/static/sidequest.css` as the visual source of truth. Preserve the fantasy design while converting sample content into real server-rendered data. Keep the prototype as a reference and place live Go templates under `src/templates/`.

Do not replace the design with Bootstrap, Tailwind, React, Vue, Alpine, or another UI framework. Avoid overwriting unrelated local work, including untracked files.

## Main page

After login, `/` should show one main parchment-style page with:

- username
- level
- XP progress bar
- time picker:
  - 15 min
  - 30 min
  - 1 hour
  - any
- “Find me a quest”
- selected/suggested quest card
- “Reroll”
- “Accept Quest”
- available quest list
- “New Quest”
- completed quests

Keep navigation minimal.

## Quest fields

Use a model roughly like:

```go
type Quest struct {
    ID               int64
    UserID           int64
    Title            string
    Description      string
    Difficulty       int
    EstimatedMinutes int
    RewardXP         int
    Status           string
    CreatedAt        time.Time
    CompletedAt      *time.Time
}
```

Statuses:

```text
available
active
completed
abandoned
```

Use simple internal names in Go/database, and fantasy wording only in presentation.

Example fantasy labels:

```text
available  -> Unclaimed
active     -> Accepted
completed  -> Completed
abandoned  -> Abandoned
```

## Database

Use PostgreSQL with migrations.

Start with these tables:

### users

```sql
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    xp INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### sessions

```sql
CREATE TABLE sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### quests

```sql
CREATE TABLE quests (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,

    title TEXT NOT NULL,
    description TEXT,

    difficulty SMALLINT NOT NULL DEFAULT 1,
    estimated_minutes INTEGER NOT NULL,
    reward_xp INTEGER NOT NULL,

    status TEXT NOT NULL DEFAULT 'available',

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,

    CONSTRAINT quest_difficulty
        CHECK (difficulty BETWEEN 1 AND 5),

    CONSTRAINT quest_estimated_minutes
        CHECK (estimated_minutes > 0),

    CONSTRAINT quest_status
        CHECK (status IN ('available', 'active', 'completed', 'abandoned'))
);
```

Use SQL directly with `pgx`. Do not use an ORM.

## Authentication

Use normal server-side sessions.

Do not use JWT.

Requirements:

- register
- login
- logout
- secure random session token
- store only a hash of the session token in PostgreSQL
- cookie should be:
  - HttpOnly
  - Secure when appropriate
  - SameSite=Lax or stricter
- session expiration
- auth middleware for protected routes

Use a safe password hashing implementation, preferably Argon2id or bcrypt.

Keep dependencies minimal.

## HTTP routes

Use Go’s standard `http.ServeMux`.

Suggested routes:

```text
GET  /login
POST /login

GET  /register
POST /register

POST /logout

GET  /{$}

GET  /quests/new
POST /quests

GET  /quests/pick

POST /quests/{id}/accept
POST /quests/{id}/complete
POST /quests/{id}/abandon
POST /quests/{id}/delete
```

Prefer standard `net/http` route patterns such as:

```go
mux.HandleFunc("POST /quests/{id}/complete", handler)
```

Use `r.PathValue("id")`.

## HTMX behavior

Do not return JSON for normal UI actions.

Return HTML fragments.

Examples:

### New quest

Click:

```text
+ New Quest
```

HTMX:

```text
GET /quests/new
```

Response:

small HTML form fragment.

Submitting:

```text
POST /quests
```

Response:

new quest row fragment.

### Pick quest

```text
GET /quests/pick?minutes=30
```

PostgreSQL should choose a suitable quest for the logged-in user.

Prefer quests where:

```text
estimated_minutes <= requested minutes
status = available
```

For `minutes=0`, treat it as any duration.

Return only the featured quest card. Preserve the selected duration when rerolling (the prototype hard-codes 30); exclude the previous suggestion when another eligible quest exists. Picking must not mutate quest status. Render a useful empty state when no quest fits.

### Complete quest

```text
POST /quests/{id}/complete
```

On completion:

- use one database transaction with a conditional status update so repeated or concurrent requests award XP exactly once
- allow completion from available or active, matching the prototype’s direct-complete buttons; completed and abandoned quests cannot earn XP again
- mark quest completed
- set `completed_at`
- add XP to user
- return updated quest fragment
- move the row into the completed list and remove its previous representation without duplicate IDs
- refresh or clear a featured card for the same quest
- update XP bar and displayed level using HTMX out-of-band swaps

Example response shape:

```html
<article id="quest-42" class="quest completed">
  ...
</article>

<div id="player-xp" hx-swap-oob="true">
  ...
</div>
```

This is important because the app should demonstrate how one tiny HTMX response can update multiple parts of the page.

Provide normal form `action`/`method` attributes and full-page fallbacks for core actions. Use `HX-Request` to distinguish fragments from full pages, and redirect normal POST submissions with 303. Give active and completed lists stable IDs and keep all visible representations consistent after mutations.

Validation failures must retain entered values and display inline errors. Explicitly configure HTMX to swap the chosen error status responses, or use another documented consistent approach; do not assume HTMX swaps 4xx responses by default. Handle expired sessions on HTMX requests without inserting a login page into a quest row.

Define legal status transitions explicitly: available → active, available/active → completed, available/active → abandoned. Deletion must not re-award XP; document whether deleting completed history retains earned XP. Compute rewards on the server, never trust submitted reward values. Add an edit form/route if implementing full CRUD; otherwise describe the scope accurately as create/list/status actions/delete.

## XP system

Keep XP simple.

Suggested base values:

```text
difficulty 1 -> 20 XP
difficulty 2 -> 40 XP
difficulty 3 -> 70 XP
difficulty 4 -> 110 XP
difficulty 5 -> 170 XP
```

It is okay to add a small bonus for quests estimated at 60+ minutes.

Do not build a complicated economy.

## Level system

Implement a simple deterministic level calculation from total XP.

Keep it easy to understand and test.

For example, use a progression function or small threshold table.

The XP bar should show progress toward the next level.

## Templates

Use Go templates with reusable fragments.

Suggested layout, inside the existing Go module:

```text
src/templates/
├── layout.html
├── home.html
├── auth/
│   ├── login.html
│   └── register.html
└── components/
    ├── quest-list.html
    ├── quest-row.html
    ├── quest-picker.html
    ├── quest-card.html
    ├── quest-form.html
    ├── player-header.html
    └── toast.html
```

Keep template logic simple.

## Static assets

Use the provided fantasy visual assets.

The prototype identifies its border artwork as Kenney Fantasy UI Borders (CC0). Verify the asset provenance and include the appropriate notice; the referenced license file is currently missing.

Use those assets mainly for:

- parchment/frame borders
- featured quest card borders
- button accents
- decorative fantasy framing

Do not make every UI control image-based.

Buttons should remain real HTML `<button>` elements.

Prefer SVG and CSS over heavy PNG usage.

## Styling goals

The page should feel like:

- adventurer guild ledger
- parchment journal
- quest board
- subtle medieval fantasy
- readable
- elegant
- lightweight
- not cheesy
- not pixel-art-heavy
- not cyberpunk
- not a developer dashboard

Use:

- parchment colors
- dark warm background
- ink-like text
- gold/brown accents
- subtle red/wax-seal accents
- fantasy border assets
- simple serif typography
- minimal animation

Keep all animations small and optional.

Respect `prefers-reduced-motion`.

## Performance goals

The project should intentionally show how lightweight this stack is.

Aim for:

- one Go binary
- embedded templates
- embedded CSS
- embedded selected assets
- embedded local HTMX file if practical
- no npm
- no build pipeline
- no frontend bundler
- no SPA
- no hydration
- no large JS dependencies

Use `go:embed` for templates/static assets.

The deployment should ideally be:

```text
sidequest binary
+
PostgreSQL
```

## Project structure

Keep root tooling and deployment files in place. Grow the existing `src/` module with a small structure like this (new files below are proposed, not already implemented):

```text
sidequest/
├── .mise.toml
├── .air.toml
├── .pre-commit-config.yaml
├── .envs/                    # private local env files; safe examples only in git
├── docker-compose.yml
├── compose/
│   ├── backend/              # existing Dockerfile and start script
│   ├── postgres/             # existing Dockerfile and maintenance scripts
│   └── pgadmin/              # existing optional database UI
├── docs/
├── src/
│   ├── go.mod                # github.com/Metal-Bat/sidequest
│   ├── go.sum
│   ├── main.go               # package main; configuration and server startup
│   ├── embed.go              # embed templates, runtime static files, migrations
│   ├── internal/
│   │   ├── auth/             # sessions, passwords, auth handlers
│   │   ├── quest/            # model, SQL, business rules, handlers
│   │   ├── user/             # user data and XP access, if needed separately
│   │   ├── database/         # pgx pool and migration support
│   │   └── web/              # rendering and shared middleware
│   ├── templates/
│   │   ├── layout.html
│   │   ├── home.html
│   │   ├── auth/
│   │   └── components/
│   ├── migrations/
│   └── static/
│       ├── index.html        # preserved visual prototype
│       ├── README.md
│       ├── sidequest.css
│       ├── vendor/htmx.min.js
│       └── assets/           # existing SVG borders and verified license notice
├── LICENSE
└── README.md
```

Do not over-engineer the layers. Add packages only when they have a clear purpose; avoid empty scaffolding and unnecessary interfaces.

Place embed declarations in `src/embed.go`, where `templates/`, `static/`, and `migrations/` are descendants. Pass the embedded filesystem/sub-filesystems into internal packages; `go:embed` cannot reference parent directories with `..`.

Serve runtime assets at `/static/` with the correct filesystem prefix. Live templates should link `/static/sidequest.css` and `/static/vendor/htmx.min.js`; CSS border URLs should resolve to `/static/assets/*.svg`. Embed selected runtime assets rather than unused packs. Keep prototype HTML and its README out of the public runtime asset handler.

## Security

Implement sensible defaults:

- CSRF protection for mutating form requests
- secure password hashing
- secure session tokens
- SQL parameterization
- users can only access their own quests
- validate quest title length
- validate difficulty
- validate estimated time
- reject invalid IDs
- proper status codes
- do not leak database errors to users

## Error handling

Use a small error page or inline error fragment.

For HTMX requests, return appropriate HTML fragments.

For normal page loads, render complete error pages.

## Logging

Use standard library logging or `log/slog`.

Log:

- request method
- path
- status
- duration
- server errors

Avoid adding a heavy logging dependency.

## Testing

Add useful tests for:

- auth/session behavior
- XP calculation
- level calculation
- quest selection by available time
- quest ownership
- complete quest flow
- validation

Prefer `httptest` and normal Go testing.

Do not introduce a large testing framework.

## README

Create a concise README with:

- what Sidequest is
- screenshots or placeholder section
- stack
- how to run
- PostgreSQL setup
- migrations
- environment variables
- development commands
- production build
- architecture overview
- why HTMX + Go was chosen
- Kenney asset attribution note even though CC0 does not require attribution

Suggested tagline:

> Turn spare time into small adventures.

Also mention:

> Sidequest is intentionally server-rendered. Most interactions return tiny HTML fragments instead of JSON.

## Environment variables

Support at least:

```text
DATABASE_URL
APP_ADDR
APP_ENV
```

Use opaque random session tokens whose hashes are stored in the database. Add a separate secret environment variable only if the chosen CSRF/session implementation actually requires one, and document its purpose.

Example:

```text
APP_ADDR=:8000
APP_ENV=development
DATABASE_URL=postgres://postgres:postgres@localhost:5432/sidequest?sslmode=disable
```

## Developer experience

Include a simple local-development setup.

If Docker is used, use it only for PostgreSQL unless necessary.

Reuse the existing `docker-compose.yml`, PostgreSQL Dockerfile, persistent volume, and maintenance scripts. Keep pgAdmin and the backend container optional. Preserve database data; do not reset volumes to repair configuration. Align the HTTP listen address with the existing `8000:8000` backend port mapping.

Extend the existing root `.mise.toml` with missing development and migration tasks, keeping Go commands rooted in `src/`. Existing test/fmt/lint tasks should remain useful. Remove the stale `bundle-swagger` task if its referenced API documents are absent; this HTML app needs neither OpenAPI generation nor npx. Provide commands such as:

```text
mise run dev
mise run test
mise run fmt
mise run lint
mise run migrate
```

Keep the app itself outside Docker during development unless there is a strong reason.

## Important constraints

Do not use:

- React
- Vue
- Svelte
- Angular
- Alpine
- Tailwind
- Bootstrap
- an ORM
- JWT auth
- GraphQL
- a separate REST API
- frontend build tools unless unavoidable
- unnecessary abstraction
- unnecessary third-party packages

Prefer:

```text
Go standard library
pgx
HTMX
plain CSS
html/template
PostgreSQL
```

## Implementation order

Please implement in this order:

1. inspect and preserve the existing frontend prototype
2. make `src/main.go` executable and extend the existing `src/` structure
3. configure PostgreSQL
4. create migrations
5. implement template rendering
6. serve static assets
7. implement CSRF protection and register/login/logout together
8. implement auth middleware
9. implement quest CRUD
10. implement quest picker
11. implement XP and levels
12. connect all interactions using HTMX
13. verify security, HTMX error handling, and normal-form fallbacks
14. add tests
15. add README and development setup

Do not redesign the visual theme unless necessary.

Prioritize:

- simple code
- readable architecture
- low dependency count
- tiny HTMX responses
- fast server rendering
- good HTML semantics
- polished fantasy presentation
- production-sensible security

When finished, provide a short summary of:
- architecture
- dependency list
- routes
- database tables
- how to run it
- any design decisions or tradeoffs

## Acceptance criteria

- Build and test from `src/`, or via root mise tasks; do not assume a root `go.mod`.
- Document and verify database startup, migrations, native development, and a production binary build. A small explicit migration command such as `go run . migrate` is sufficient if implemented.
- Set HTTP timeouts, gracefully shut down the server/pool, and avoid exposing internal errors.
- Verify isolation between two users and concurrent/repeated completion with database-backed tests. Do not run destructive integration tests against a personal database.
- Exercise registration, login, logout, creation, selection/reroll, acceptance, completion, abandonment, and deletion; verify XP, level, and list updates.
- Check the fantasy layout on narrow screens, keyboard accessibility, reduced motion, all asset URLs, and empty/error states.
- Report checks actually run and any blockers honestly. Do not claim the app is implemented merely because the scaffold builds.
