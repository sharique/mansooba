# Architecture Overview

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

---

## API overview

All routes are prefixed with `/api/v1` and require a JWT `Authorization: Bearer <token>` header except `/auth/register` and `/auth/login`.

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
