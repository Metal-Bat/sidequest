# Sidequest Guild

**Small deeds. Lasting legends.**

Turn spare time into small adventures. Sidequest is a personal quest journal where
you record tasks, find something that fits the time you have, and earn experience
for completing it.

The homepage takes inspiration from tabletop character sheets: light parchment on
a warm walnut background, framed level and XP stats, gold accents, and a personal
quest board. Cinzel headings and Source Sans 3 text are bundled locally.

## Your adventurer’s ledger

- **Character summary:** your name, Adventurer or Guildmaster role, level, total XP,
  and progress toward the next level.
- **Time for an adventure?:** choose 15 minutes, 30 minutes, 1 hour, or Any, then
  select **Find me a quest**. Review its difficulty, estimated time, and XP reward.
  Accept the suggestion or reroll for another.
- **Quest Board:** keep track of Unclaimed, Accepted, Completed, and Abandoned
  quests. Use **New Quest** to add a title, optional description, difficulty, and
  estimated duration.
- **Guildmaster’s desk:** administrators see **Gather your party** and a
  **Create user** button. The same action is available in the main navigation.

The quest picker and board sit side by side on desktop and stack on mobile.
The [static preview](src/static/index.html) contains example data; forms and
navigation require the running application. Live pages are in
[`src/templates/`](src/templates/), with styles in
[`src/static/sidequest.css`](src/static/sidequest.css).

## Start with Docker

From the repository root, copy the example configuration files if you do not
already have them:

```sh
cp -n .envs/.postgres.example .envs/.postgres
cp -n .envs/.backend.example .envs/.backend
cp -n .envs/.pgadmin.example .envs/.pgadmin
```

Match `DATABASE_URL` in `.envs/.backend` to the credentials in `.envs/.postgres`.
The database hostname inside Docker is `postgres`.

```sh
docker compose up -d --build backend
```

Open <http://localhost:8000> and choose **Join the guild** to create a regular
account. The backend waits for PostgreSQL to become healthy, applies migrations,
and starts the server. If migration fails, startup stops.

To start the optional database administration UI:

```sh
docker compose up -d pgadmin
```

pgAdmin is available at <http://localhost:5050>. PostgreSQL data persists in a
Docker volume. Changing environment files does not change credentials already
stored in an existing database; keep those credentials when reusing a volume.

## Create users as a Guildmaster

Create an administrator from an interactive terminal:

```sh
docker compose run --rm backend create-admin
```

The command applies migrations, then prompts for a username and a hidden password
with confirmation. It creates a new administrator; it does not promote or replace
an existing account.

Sign in with that account. Click **Create user** in the navigation or the
**Gather your party** panel, or visit <http://localhost:8000/admin/users/new>.
New accounts created there are regular members. Share their credentials privately
so they can sign in.

Regular accounts do not see these controls and cannot access the admin page.
Public registration also creates regular members. Administrators cannot view or
manage other members’ quests.

## Run locally

The project uses Go **1.27.1** and PostgreSQL **18.1**. Mise pins the development
tools, including Node **26.1.0** for HTML and CSS checks. The Go module is in `src/`.

```sh
cp -n .envs/.local.example .envs/.local
mise install
mise run ui-install
docker compose up -d postgres
```

Set `DATABASE_URL` in `.envs/.local` to match your database credentials, using
`localhost` as the hostname for native development. Mise loads this file
automatically; private environment files are ignored by Git.

```sh
mise run migrate
mise run dev
```

For a native administrator account, run `mise run create-admin`.
Without Mise, export the application environment variables into your shell first:

```sh
cd src
go run . migrate
go run .
```

Native startup does not apply migrations automatically. Migrations are embedded,
tracked in `schema_migrations`, and serialized with a PostgreSQL advisory lock.
Running them again is safe.

## Quest rules

| Difficulty | Completion reward |
| --- | --- |
| 1 | 20 XP |
| 2 | 40 XP |
| 3 | 70 XP |
| 4 | 110 XP |
| 5 | 170 XP |

Every 200 XP adds a level. Completing a quest credits XP once, even if completion
requests arrive together. Deleting a completed quest keeps the XP you earned.

Unclaimed quests can be accepted, completed, or abandoned. Accepted quests can be
completed or abandoned. Completed and abandoned quests cannot be reopened.
Quests can be deleted; editing is not currently available.

Finding or rerolling a quest does not change its status. Suggestions fit the
selected time limit; **Any** removes that limit. Reroll prefers a different quest
when another one fits.

## Development checks

Install the UI tools once with `mise run ui-install`, then use:

```sh
mise run fmt          # Format Go, HTML, and CSS
mise run lint         # Run Go and UI linters
mise run vet
mise run test
mise run test-race
```

For just the frontend files:

```sh
mise run ui-format
mise run ui-lint
mise run ui-fix
```

Prettier formats HTML and Go templates. HTMLHint checks markup while ignoring Go
template actions; Go tests validate template parsing and rendering. Stylelint
checks CSS. Vendored HTMX and fonts are excluded from formatting.

Install the configured Git hooks with `pre-commit install`. The hooks include
HTML/CSS formatting and lint checks. Node is only a development dependency; the
application and production container do not need it.

For hot reload, install Air and run `air -c .air.toml` from the repository root.

### Database integration tests

Use a disposable PostgreSQL database:

```sh
docker run --rm -d --name sidequest-test-db \
  -p 127.0.0.1:55439:5432 \
  -e POSTGRES_PASSWORD=sidequest_test -e POSTGRES_DB=sidequest_test \
  postgres:18.1-bookworm
export TEST_DATABASE_URL='postgres://postgres:sidequest_test@127.0.0.1:55439/sidequest_test?sslmode=disable'
# Wait for PostgreSQL to be ready before running tests.
mise run test-integration
docker stop sidequest-test-db
```

Each database test creates and removes its own random schema. Database tests skip
when `TEST_DATABASE_URL` is unset. They cover quest ownership, concurrent
completion, migrations, authentication, and admin user creation.

## Configuration and deployment

| Variable | Meaning |
| --- | --- |
| `DATABASE_URL` | Required PostgreSQL connection string. |
| `APP_ADDR` | HTTP listen address; defaults to `:8000`. |
| `APP_ENV` | Empty or `development`, `test`, or `production`. Production enables Secure cookies, HSTS, and JSON logs. |
| `TEST_DATABASE_URL` | Optional disposable database for integration tests. |

For a standalone binary:

```sh
mise run release
# Export DATABASE_URL before running the binary.
APP_ENV=production ./bin/sidequest migrate
APP_ENV=production ./bin/sidequest
```

Use a TLS reverse proxy that preserves the original Host in production, and TLS
for remote database connections. The binary embeds templates, migrations, CSS,
fonts, and HTMX; no source tree or CDN is needed at runtime.

Logging covers database connection, commands, server startup and shutdown, request
method/path/status/duration, and errors. Production uses JSON; other environments
use text logs. HTTP timeouts and graceful shutdown are configured.

## Project structure

| Location | Purpose |
| --- | --- |
| `src/main.go`, `src/app.go` | Entry point and application startup. |
| `src/config.go`, `src/logging.go` | Environment configuration and logging setup. |
| `src/server.go` | HTTP server and graceful shutdown. |
| `src/commands.go`, `src/admin_cli.go` | Migration and administrator commands. |
| `src/internal/database/` | Connection pool and migration runner. |
| `src/internal/account/` | Account creation and password hashing. |
| `src/internal/quest/` | Quest validation, rewards, and database operations. |
| `src/internal/web/` | Routes, authentication, CSRF, templates, and request logging. |
| `src/templates/`, `src/static/` | Live templates, stylesheet, preview, local fonts, and HTMX. |
| `src/migrations/` | Versioned SQL migrations. |
| `compose/` | Container builds, startup script, and PostgreSQL maintenance tools. |

Go serves HTML directly. HTMX updates quest rows, suggestions, and the XP summary
without full-page reloads; core forms also work without JavaScript. PostgreSQL
stores users, sessions, quests, and migration history.

Passwords use bcrypt. Sessions expire after seven days, and only hashes of session
tokens are stored. Forms use CSRF protection, quest queries are scoped to their
owner, and authentication attempts are rate-limited per peer IP.

## Bundled licenses

Cinzel and Source Sans 3 include their SIL Open Font License files in
[`src/static/fonts/`](src/static/fonts/). HTMX’s license is in
[`src/static/vendor/LICENSE-HTMX.txt`](src/static/vendor/LICENSE-HTMX.txt).
