# API Reference

All routes are prefixed `/api/v1`. Protected routes require `Authorization: Bearer <token>` (obtained from `/auth/login` or `/auth/refresh`).

Swagger UI is available at `/swagger/index.html` when the server is running.

---

## Auth & user

| Method | Path | Auth | Description |
|--------|------|:----:|-------------|
| POST | `/auth/register` | — | Register a new user |
| POST | `/auth/login` | — | Login; returns `access_token` + `refresh_token` |
| POST | `/auth/refresh` | — | Exchange refresh token for a new access token |
| GET | `/auth/me` | ✓ | Get current user profile (includes `is_super_admin`) |
| PUT | `/auth/me` | ✓ | Update profile (name, timezone) |
| GET | `/auth/me/activity` | ✓ | My recent activity (paginated) |
| GET | `/auth/me/issues` | ✓ | Issues assigned to me across all projects |
| POST | `/auth/me/avatar` | ✓ | Upload avatar image (multipart, max 4 MB) |
| DELETE | `/auth/me/avatar` | ✓ | Remove avatar |
| GET | `/uploads/*` | — | Static avatar files — intentionally unauthenticated (ADR-026) |

---

## Setup wizard (first run only)

These routes bypass JWT auth and work only when no users exist yet.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/setup/status` | Returns `{ "setup_required": true/false }` |
| POST | `/api/v1/setup` | Create the superadmin account |
| POST | `/api/v1/setup/initial-user` | Alias for the above |

---

## Projects

| Method | Path | Auth | Description |
|--------|------|:----:|-------------|
| GET | `/projects` | ✓ | List all projects the caller is a member of |
| POST | `/projects` | ✓ | Create a project |
| GET | `/projects/:key` | ✓ | Get a project by key |
| PUT | `/projects/:key` | ✓ | Update a project (admin only) |
| DELETE | `/projects/:key` | ✓ | Delete a project (admin only) |
| GET | `/projects/:key/members` | ✓ | List project members |
| POST | `/projects/:key/members` | ✓ | Add a member (admin only) |
| DELETE | `/projects/:key/members/:userId` | ✓ | Remove a member (admin only) |

---

## Labels

| Method | Path | Auth | Description |
|--------|------|:----:|-------------|
| GET | `/projects/:key/labels` | ✓ | List labels for a project |
| POST | `/projects/:key/labels` | ✓ | Create a label |
| DELETE | `/projects/:key/labels/:lid` | ✓ | Delete a label |

---

## Issues

| Method | Path | Auth | Description |
|--------|------|:----:|-------------|
| GET | `/projects/:key/issues` | ✓ | List issues with optional filters (see below) |
| POST | `/projects/:key/issues` | ✓ | Create an issue |
| GET | `/projects/:key/issues/:id` | ✓ | Get an issue |
| PUT | `/projects/:key/issues/:id` | ✓ | Update an issue |
| DELETE | `/projects/:key/issues/:id` | ✓ | Delete an issue (reporter or admin) |

**Issue list query parameters:**

| Param | Type | Description |
|-------|------|-------------|
| `q` | string | Full-text search on title and description |
| `type` | string | `bug`, `story`, `task`, `epic` |
| `status` | string | `backlog`, `todo`, `in_progress`, `in_review`, `done` |
| `priority` | string | `low`, `medium`, `high`, `critical` |
| `assignee_id` | uint | Filter by assignee user ID |
| `label_id` | uint | Filter by label |
| `page` | int | Page number (default 1) |
| `limit` | int | Page size (default 50, max 100) |

---

## Issue sub-resources

All routes below are scoped to a specific issue (`/issues/:id/`).

### Comments

| Method | Path | Description |
|--------|------|-------------|
| GET | `/issues/:id/comments` | List comments for an issue |
| POST | `/issues/:id/comments` | Add a comment |
| PUT | `/issues/:id/comments/:cid` | Update a comment (author only) |
| DELETE | `/issues/:id/comments/:cid` | Delete a comment (author or admin) |

### Activity

| Method | Path | Description |
|--------|------|-------------|
| GET | `/issues/:id/activity` | Field-change audit log for an issue |

### Labels (attach/detach)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/issues/:id/labels` | Labels currently attached to this issue |
| POST | `/issues/:id/labels/:lid` | Attach a label |
| DELETE | `/issues/:id/labels/:lid` | Detach a label |

### Attachments

| Method | Path | Description |
|--------|------|-------------|
| POST | `/issues/:id/attachments` | Upload one or more files (multipart; max 10 MB per file, 25 MB per request — a route-scoped override of the global 4 MB body limit) |
| GET | `/issues/:id/attachments` | List attachments (returns metadata, not binaries) |
| GET | `/issues/:id/attachments/:aid/download` | Get a pre-signed download URL as JSON (1 h TTL, not a 302 redirect) |
| DELETE | `/issues/:id/attachments/:aid` | Delete an attachment (removes from S3 too) |

### Relations

| Method | Path | Description |
|--------|------|-------------|
| POST | `/issues/:id/relations` | Link this issue to another (`{ "related_task_id": N }`) |
| GET | `/issues/:id/relations` | List all "related to" links for this issue |
| DELETE | `/issues/:id/relations/:rid` | Remove a relation |

Relations are symmetric and stored once per pair (`task_a_id < task_b_id`). The API returns all relations where this issue appears on either side.

**Relation error codes:**

| Condition | Status |
|-----------|--------|
| Target issue not found | 404 |
| Linking an issue to itself | 422 |
| Linking issues from different projects | 422 |
| Relation already exists | 409 |

---

## Board

| Method | Path | Auth | Description |
|--------|------|:----:|-------------|
| GET | `/projects/:key/board` | ✓ | Issues grouped by status (kanban columns) |

---

## Sprints

| Method | Path | Auth | Description |
|--------|------|:----:|-------------|
| GET | `/projects/:key/sprints` | ✓ | List sprints |
| POST | `/projects/:key/sprints` | ✓ | Create a sprint |
| GET | `/projects/:key/sprints/:id` | ✓ | Get a sprint |
| PUT | `/projects/:key/sprints/:id` | ✓ | Update a sprint |
| DELETE | `/projects/:key/sprints/:id` | ✓ | Delete a sprint |
| POST | `/projects/:key/sprints/:id/start` | ✓ | Start a sprint (sets `active` status) |
| POST | `/projects/:key/sprints/:id/complete` | ✓ | Complete a sprint (moves unfinished issues to backlog) |
| GET | `/projects/:key/sprints/:id/burndown` | ✓ | Burndown chart data points |
| GET | `/projects/:key/sprints/:id/issues` | ✓ | Issues assigned to this sprint |
| GET | `/projects/:key/backlog` | ✓ | Issues not assigned to any sprint |
| GET | `/projects/:key/velocity` | ✓ | Velocity data (story points completed per sprint) |

---

## Notifications

| Method | Path | Auth | Description |
|--------|------|:----:|-------------|
| GET | `/notifications` | ✓ | My unread notifications |
| PUT | `/notifications/:id/read` | ✓ | Mark a notification as read |

---

## Settings

| Method | Path | Auth | Description |
|--------|------|:----:|-------------|
| GET | `/settings` | ✓ | Get all org-wide settings |
| PUT | `/settings` | ✓ (superadmin) | Update settings (`{ "values": { "org.timezone": "UTC" } }`) |

**Available setting keys:**

| Key | Description |
|-----|-------------|
| `org.timezone` | Organisation default timezone (IANA format, e.g. `America/New_York`) |
| `org.locale` | Organisation locale (e.g. `en-US`) |

---

## Health

| Method | Path | Auth | Description |
|--------|------|:----:|-------------|
| GET | `/health` | — | Liveness + readiness probe |

**Response (200 — healthy):**
```json
{
  "status": "ok",
  "db": "ok",
  "db_latency_ms": 1,
  "storage": "ok"
}
```

**Response (503 — degraded):**
```json
{
  "status": "degraded",
  "db": "ok",
  "db_latency_ms": 2,
  "storage": "error",
  "error": "<reason>"
}
```
