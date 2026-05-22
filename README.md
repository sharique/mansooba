# go-jira

A project management application (similar to Jira) built in Go and Nuxt.js as a learning and portfolio project.

Built using a spec-driven approach where I worked as architect/manager and Claude worked as engineer.

**Backend:** Go + Echo v4 · GORM · SQLite (local) / PostgreSQL (prod) · JWT auth  
**Frontend:** Nuxt 4 (SPA) · Pinia · Tailwind CSS v4 · DaisyUI

> Design spec, ADRs, and task plans live in the docs repo: [sharique/jira-go](https://github.com/sharique/jira-go)

---

## Features

### Authentication
- JWT-based login and registration
- User profile: view and update display name, email, timezone
- My Activity feed — paginated list of your recent project events

### Projects
- Create, read, update, delete projects
- Project membership with roles (`admin`, `member`)
- Role-based access control: owners and admins can manage sprints, labels, and members

### Issues
- Full CRUD with fields: title, description, type, status, priority, story points, reporter, assignee
- Type: `task`, `story`, `bug`, `epic`
- Status workflow: `backlog → todo → in_progress → in_review → done`
- Priority levels: `critical`, `high`, `medium`, `low`
- Labels: create project labels, tag issues, filter by label
- Issue search: text search across title/description + filter by type, status, priority, label

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

### Reports
- `/reports` page with project selector
- Sprint velocity chart showing committed vs completed story points across all completed sprints
- CSS-only bar chart — no external charting dependency

---

## Prerequisites

- Go 1.21+
- Node 22 + npm 10
- SQLite development headers (`sqlite-devel` on Fedora / `libsqlite3-dev` on Debian/Ubuntu)

---

## Running locally

### 1. Environment

```sh
cp backend/.env.example backend/.env
# Edit backend/.env if needed — defaults work out of the box with SQLite
```

### 2. Backend

```sh
cd backend
go run ./cmd/server
# API at http://localhost:8080
# Health check: GET http://localhost:8080/health
```

### 3. Frontend

```sh
cd frontend
npm install
npm run dev
# App at http://localhost:3000
```

---

## API overview

| Method | Path | Description |
|--------|------|-------------|
| POST | `/auth/register` | Register a new user |
| POST | `/auth/login` | Login, returns JWT |
| GET | `/auth/me` | Get current user profile |
| PUT | `/auth/me` | Update profile (name, timezone) |
| GET | `/auth/me/activity` | My recent activity (paginated) |
| GET | `/projects` | List all projects |
| POST | `/projects` | Create a project |
| GET/PUT/DELETE | `/projects/:key` | Get, update, or delete a project |
| GET/POST | `/projects/:key/members` | List or add project members |
| GET | `/projects/:key/issues` | List issues (filters: q, type, status, priority, label_id) |
| POST | `/projects/:key/issues` | Create an issue |
| GET/PUT/DELETE | `/projects/:key/issues/:id` | Get, update, or delete an issue |
| GET/POST | `/projects/:key/issues/:id/comments` | List or add comments |
| PUT/DELETE | `/projects/:key/issues/:id/comments/:cid` | Update or delete a comment |
| GET | `/projects/:key/issues/:id/activity` | Issue activity feed |
| GET/POST | `/projects/:key/sprints` | List or create sprints |
| POST | `/projects/:key/sprints/:id/start` | Start a sprint |
| POST | `/projects/:key/sprints/:id/complete` | Complete a sprint |
| GET | `/projects/:key/sprints/:id/burndown` | Burndown chart data |
| GET | `/projects/:key/backlog` | Backlog issues |
| GET | `/projects/:key/velocity` | Velocity chart data |
| GET/POST | `/projects/:key/labels` | List or create labels |
| DELETE | `/projects/:key/labels/:id` | Delete a label |
| GET | `/notifications` | My unread notifications |
| PUT | `/notifications/:id/read` | Mark a notification as read |

---

## Project structure

```
backend/
  cmd/server/           # Entry point — route wiring and server startup
  internal/
    domain/             # Entities + repository interfaces (zero external imports)
    repository/         # GORM implementations of all repositories
    service/            # Business logic layer
    handler/            # Echo HTTP handlers + request/response DTOs
    middleware/         # JWT auth middleware
  pkg/
    config/             # Env config (Viper)
    database/           # GORM connection + auto-migrations
    logger/             # Zap logger

frontend/
  app/
    pages/              # Nuxt file-based routes
    components/         # Vue SFCs (board/, backlog/, sprints/, issues/, labels/, reports/, ...)
    stores/             # Pinia stores (auth, issues, sprints, labels, notifications, projects)
    services/           # API layer wrapping $api ($fetch proxy)
    types/              # TypeScript domain types (domain.types.ts)
    layouts/            # Default layout with nav bar
```

---

## Architecture

**Backend** follows Clean Architecture:  
`domain` → `repository` → `service` → `handler`

- `domain/` defines entities and repository interfaces with zero external imports
- `repository/` implements those interfaces using GORM
- `service/` owns all business logic and works only with domain types + DTOs
- `handler/` translates HTTP ↔ service calls; error mapping in `apierror/`

**Frontend** uses Options API Pinia stores + Composition API pages/components:

- `services/` wraps `$api` (Nuxt server-side proxy to backend) — each service mirrors a backend resource
- `stores/` hold fetched state and expose actions that call services
- Component names are prefixed by their directory (e.g. `IssuesIssueCard`, `SprintsSprintList`)
- Vue auto-imports for composables; TypeScript throughout
