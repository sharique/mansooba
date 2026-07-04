# Mansooba

*Mansooba (منصوبہ) — Urdu for "plan" or "project" (German: Projekt)*

A project management application built in Go and Nuxt.js as a learning and portfolio project.

Built using a spec-driven approach where I worked as architect/manager and Claude worked as engineer.

**Backend:** Go + Echo v4 · GORM · SQLite (local) / PostgreSQL (prod) · JWT auth  
**Frontend:** Nuxt 4 (SPA) · Pinia · Tailwind CSS v4 · DaisyUI · OKLCH design system

### Demo

[![Initial Setup Walkthrough](https://cdn.loom.com/sessions/thumbnails/61ce0f47d5f14ecbb51c7a314fa0a107-with-play.gif)](https://www.loom.com/share/61ce0f47d5f14ecbb51c7a314fa0a107)

---

## Features

### Authentication & Security
- JWT-based login with cookie-based refresh tokens (HttpOnly, SameSite=Strict, Secure in production); registration is admin-controlled (see **User Management** below)
- Server-side logout — refresh token JTI stored in `revoked_tokens` table; checked fail-closed on every token refresh (store error → 503, not a silent grant)
- Background goroutine purges expired revocation records on a configurable interval
- Password reset — request a reset token at `/forgot-password`; token shown on screen and pre-fills the `/reset-password` page
- User profile: view and update display name, email, timezone
- Avatar upload — upload a profile picture (stored on disk); falls back to OKLCH-colored initials when no photo is set
- My Activity feed — paginated list of your recent project events; sprint assignment entries display real sprint names (e.g. "Sprint Alpha") captured at the time of the move

### Projects
- Create, read, update, delete projects
- Project membership with roles (`admin`, `member`)
- Role-based access control: owners and admins can manage sprints, labels, and members

### Issues
- Full CRUD with fields: title, description, type, status, priority, story points, reporter, assignee
- Markdown-rendered descriptions
- Type: `task`, `story`, `bug`, `epic`
- Status workflow: `backlog → todo → in_progress → in_review → done`
- Priority levels: `critical`, `high`, `medium`, `low`
- Labels: create project labels, tag issues, filter by label
- Issue search: text search across title/description + filter by type, status, priority, label
- Related tasks: link issues via `blocks` / `is_blocked_by`, `relates_to`, or `duplicates`; reciprocal links maintained automatically; cascade-deleted with the parent issue

### Sprints
- Sprint CRUD with lifecycle: `planning → active → completed`
- One active sprint per project enforced
- Sprint completion with automatic migration of unfinished issues to next sprint or backlog
- Burndown chart (story points remaining over time)
- Velocity chart (committed vs completed story points per sprint)

### Backlog & Board
- Backlog view: issues not assigned to any sprint; assign to sprint from backlog
- Kanban board: issues grouped by status column

### Collaboration
- Comments on issues with edit and delete (owner or admin)
- `@mention` parsing — mentioned users receive in-app notifications
- Activity feed per issue: records status changes, assignments, comments
- Notifications: unread badge count + mark-read

### Dashboard
- Landing page with key project metrics at a glance
- My Desk — personal work hub: issues assigned to you, unread notifications, recent activity, pinned projects

### Reports
- `/reports` page with project selector
- Sprint velocity chart showing committed vs completed story points across all completed sprints
- CSS-only bar chart — no external charting dependency

### System Administration
- Global platform settings (`/system/settings`) — admin-only: date format, time format, timezone, session timeout, max file upload size
- User management (`/system/users`) — paginated list of all users with role and status badges; promote/demote admin, enable/disable accounts; last-admin guard prevents removing the final active admin
- Create user (`/system/createuser`) — admin-only form; new account credentials shared directly with the user
- Sidebar **System** section visible only to admins, with links to all three admin pages
- Role-aware UI: admin users get a create dropdown; members get a single-action button; users with no project membership see neither

### First-Run Setup Wizard
- On a fresh install (no admin account exists), all routes redirect to `/setup` automatically via a global Nuxt middleware
- Five-step wizard: Welcome → Admin account → Optional team member → Optional project → Sample data → Summary screen
- Admin creation uses a public, rate-limited endpoint (`POST /api/v1/setup/admin`); subsequent steps require the JWT issued at that step
- Password complexity enforced in real time (per-rule pass/fail indicators on keystroke: length, uppercase, lowercase, digit)
- Project step optionally adds the newly created team member as a project member in a single atomic request
- Step 4 "Sample data" imports a demo project (key `DEMO`), an active sprint, 7 issues, 2 labels, and 2 comments in a single DB transaction
- Setup endpoints self-deactivate once an admin exists (409 on subsequent calls); `/setup` redirects to login once setup is complete

### Sample data CLI (dev/demo utility)

If the wizard's Step 4 was skipped or failed, the same demo dataset can be imported from the command line:

```sh
cd backend
go run ./cmd/seed
```

The command is idempotent — running it twice is safe. It requires the setup wizard to have been completed (admin account must exist). See [`docs/first-run-wizard.md`](docs/first-run-wizard.md) for full details.

### User Management (admin-controlled registration)

User account creation is admin-only — `POST /api/v1/auth/register` requires a valid admin JWT.

- **Admins create accounts** via `/system/createuser` in the sidebar System section.
- **Unauthenticated users** are redirected to `/login`; non-admins are redirected to `/system/users`.
- Direct API calls without an admin JWT receive `401 Unauthorized` or `403 Forbidden`.

---

## Prerequisites

- Go 1.21+
- Node 22 + npm 10
- Database
  - SQLite (for quick set or local experiment)
  - Postgres or MariaDB

---

## Running locally

### Docker Compose (quickest)

```sh
docker compose up --build
```

- App at **http://localhost:3000** — on first visit, the setup wizard creates the admin account
- API at **http://localhost:8080**
- Mailpit inbox at **http://localhost:8025** — catches all password-reset emails in dev

See [`docs/running-locally-using-docker.md`](docs/running-locally-using-docker.md) for PostgreSQL and hot-reload dev mode options.  
For pre-built GHCR images (no Go toolchain needed) see [`docs/running-from-ghcr.md`](docs/running-from-ghcr.md).  
For running the backend and frontend natively (Go + Node, no Docker images) see [`docs/running-from-source.md`](docs/running-from-source.md).  
For API reference, project structure, and architecture details see [`docs/arch-overview.md`](docs/arch-overview.md).
