# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project overview

Gophish is an open-source phishing framework (Go backend + JS/Bootstrap frontend) used to set up and run simulated
phishing campaigns for security awareness training and penetration testing. It ships as a single binary that runs
two HTTP servers side by side: an admin/API server and a phishing (landing page + tracking) server.

## Build, test, and run

```sh
go build .                        # Build the gophish binary (requires Go 1.21+, CGO for sqlite3)
go test ./...                     # Run the full Go test suite
go test ./models/...              # Run tests for a single package
go test ./models/ -run TestGetUser -v   # Run a single test by name
gofmt -d .                        # Check formatting (CI fails on any diff; run `gofmt -w .` to fix)
./gophish                         # Run the built binary (reads ./config.json, needs static/ and templates/ alongside it)
```

Frontend assets (jQuery/Bootstrap-based admin UI, not a SPA):

```sh
yarn install                      # or npm install
npx gulp build                    # Concatenates/minifies vendor + app JS and CSS -> static/{js,css}/dist
npx webpack                       # Bundles a few standalone entrypoints (passwords, users, webhooks) via Babel
```

CI (`.github/workflows/ci.yml`) runs on Go 1.21/1.22/1.23 and does: `go build`, `gofmt -d .` (must be empty), and
`go test ./...`. Match this locally before pushing.

Go tests that touch the database use an in-memory sqlite3 DB and the migrations in `db/db_sqlite3/migrations/`
(see `models/models_test.go` for the `config.Config` test fixture) — no external DB is needed to run `go test`.

## Architecture

Two servers, one binary (`gophish.go`), selectable via `--mode {all,admin,phish}`:

- **Admin server** (`controllers/route.go`, `AdminServer`) — serves the dashboard UI (Go `html/template` files in
  `templates/`, static JS/CSS in `static/`), session-based login/logout, settings, user management, and mounts the
  REST API under `/api/`.
- **API server** (`controllers/api/`, `Server`) — the REST API consumed by both the admin UI and external API
  clients (authenticated via `X-Api-Key` / API token, see `middleware.RequireAPIKey`). One file per resource:
  `campaign.go`, `group.go`, `page.go`, `template.go`, `smtp.go` (sending profiles), `user.go`, `webhook.go`,
  `imap.go`, `import.go`, `reset.go`.
- **Phishing server** (`controllers/phish.go`, `PhishingServer`) — the public-facing server that serves cloned
  landing pages and records campaign events (email sent/opened, link clicked, data submitted, email reported) by
  decoding the tracking `rid` parameter embedded in campaign emails.

Supporting packages, each with its own `doc.go` describing intent:

- `models/` — GORM (`jinzhu/gorm`) models and nearly all business logic: campaigns, groups, results, templates,
  landing pages, sending profiles (SMTP), users, RBAC roles, webhooks, mail logs. `models.Setup(conf)` opens the DB,
  runs `goose` migrations from `db/db_mysql` or `db/db_sqlite3` (chosen by `config.json`'s `db_name`), and creates
  the initial admin user (random password/API key logged on first run, or seeded via the
  `GOPHISH_INITIAL_ADMIN_PASSWORD` / `GOPHISH_INITIAL_ADMIN_API_TOKEN` env vars).
- `models/rbac.go` — simple two-role RBAC (`admin`, `user`) mapped to permissions (`view_objects`,
  `modify_objects`, `modify_system`), checked via `user.HasPermission(perm)`.
- `mailer/` + `worker/` — `worker` polls every minute for scheduled/queued campaigns, gathers pending `MailLog`
  entries from `models`, and hands them to `mailer` (interface over `jordan-wright/email` + `gophish/gomail`) to
  actually send. Campaigns are throttled/spread over time via mail logs with `send_date`.
- `imap/` — IMAP monitor (`imap.NewMonitor()`) used for the in-app "email reporting" feature (users forward
  suspicious emails to a mailbox Gophish polls).
- `auth/` — password hashing/validation, API key and secure token generation.
- `middleware/` — request/session plumbing: `RequireLogin`, `RequireAPIKey`, `RequirePermission`, CSRF exemptions,
  security headers; `middleware/ratelimit` rate-limits POST endpoints (e.g. login).
- `context/` — per-request value storage (predates Go's stdlib `context` in this codebase's history).
- `webhook/` — outbound webhook delivery for campaign events, with an allowed-host `dialer` to restrict SSRF-prone
  outbound requests (`dialer/`).
- `config/` — loads `config.json` into a `Config` struct (admin server, phish server, DB, logging settings).
- `util/` — misc helpers (e.g. self-signed cert generation for `use_tls`).

Data flow for a campaign: API/admin creates a `Campaign` (models) referencing a `Group`, `Template`, `Page`, and
`SMTP` sending profile -> `worker` schedules `MailLog` rows -> `mailer` sends templated emails with a unique
tracking `rid` per recipient -> recipient interactions hit the phishing server, which updates `Result`/`Event`
records used by the dashboard and API `.../results` endpoints.

Frontend (`static/js/src/app/*.js`) is plain jQuery/Bootstrap per admin page (one file per section, mirroring the
`AdminServer` routes), built with `gulp` (concat + uglify, no bundler) plus a small number of `webpack`/Babel
entrypoints for newer code (`passwords`, `users`, `webhooks`). Server-rendered pages live in `templates/*.html`
(Go `html/template`, composed with `base.html`/`nav.html`/`flashes.html`).

## Notes

- Database support is MySQL or SQLite3 only, selected by `db_name` in `config.json`; migrations for each live under
  `db/db_mysql/migrations` and `db/db_sqlite3/migrations` respectively (goose format).
- `dialer.SetAllowedHosts` restricts outbound connections (webhooks, IMAP) to guard against SSRF — be careful not
  to bypass this when touching `webhook/` or `imap/`.
- Security vulnerabilities should be reported privately to `security@getgophish.com`, not as public issues (see
  `CONTRIBUTING.md` / `SECURITY.md`).
