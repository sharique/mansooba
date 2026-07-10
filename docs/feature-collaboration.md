# Feature: Collaboration

## Overview

Issues support comments, @mention notifications, and a per-issue activity feed. Users
also have a personal My Activity feed showing their recent events across all projects
they belong to.

## Implementation details

### @mention parsing

- Comment text is scanned for `@username` patterns on save
- Each mentioned username is resolved to a user ID; a notification record is created for
  each resolved user
- Unresolved @mentions (unknown usernames) are silently ignored

### Notification model

- Notifications are stored per-user in the `notifications` table
- Each notification has a `read` boolean; the unread badge count is a `COUNT WHERE
  read = false`
- `POST /api/v1/notifications/:id/read` marks a single notification read
- `POST /api/v1/notifications/read-all` marks all notifications read

### Activity feed per issue

Events recorded:
- Status changes (old status → new status, actor, timestamp)
- Assignee changes
- Comment creation, edit, and deletion
- Sprint assignment — the sprint name is captured at the time of the move, not stored by
  ID, so history is stable even if the sprint is later renamed (e.g. "Sprint Alpha"
  remains "Sprint Alpha" in older feed entries)
- Attachment upload (`attachment_added`) and deletion (`attachment_removed`) — see
  [feature-issues.md](feature-issues.md#attachments)

### My Activity feed

- Aggregates activity records across all projects the user belongs to
- Paginated; most recent events first

## API endpoints

See [arch-api.md](arch-api.md). Key routes:

- `GET/POST /api/v1/projects/:id/issues/:iid/comments`
- `PUT/DELETE /api/v1/projects/:id/issues/:iid/comments/:cid`
- `GET /api/v1/projects/:id/issues/:iid/activity`
- `GET /api/v1/notifications`
- `POST /api/v1/notifications/:id/read`
- `POST /api/v1/notifications/read-all`
- `GET /api/v1/users/me/activity`
