# Mansooba

A project management application built in Go and Nuxt.js as a learning and portfolio project.

Built using a spec-driven approach where I worked as architect/manager and Claude worked as engineer.

**Backend:** Go + Echo v4 · GORM · SQLite (local) / PostgreSQL (prod) · JWT auth  
**Frontend:** Nuxt 4 (SPA) · Pinia · Tailwind CSS v4 · DaisyUI

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

For API reference, project structure, and architecture details see [`docs/arch-overview.md`](docs/arch-overview.md).
