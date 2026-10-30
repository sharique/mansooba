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
- An `@` starts a mention only when the character before it is not a letter, digit, underscore, dot or another `@`, so `(@sam)` and `**@sam**` notify but `a@b.com` does not

### Mention autocomplete and warning

- Typing `@` in a comment box (new comment or editing one) lists the other active members of the issue's project: display name over `@username`, never an email address. It filters as you type (name or username, any case, prefix matches first); arrow keys move, Enter or Tab insert `@username `, Escape closes
- About 400 ms after typing pauses, a non-blocking line warns about each `@handle` that is not a member of the project. It never stops the comment from being posted
- The list uses the member list already loaded for the issue; it makes no extra request. A mention of a non-member still notifies that user on the server (only the writer is warned)

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
  [issues.md](issues.md#attachments)
- Task deletion (`issue_deleted`): the task and its feed are gone, so this event is not on any
  task's page. It appears in the project overview's Recent activity and in the deleter's My
  Activity, named from a snapshot (key, title, project, sprint) stored on the event

### My Activity feed

- Aggregates activity records across all projects the user belongs to
- Paginated; most recent events first
- Includes the tasks the user deleted ("deleted the task …"). Older events of a task that has since been deleted still show, without the task's key

## API endpoints

See [api.md](../arch/api.md). Key routes:

- `GET/POST /api/v1/projects/:id/issues/:iid/comments`
- `PUT/DELETE /api/v1/projects/:id/issues/:iid/comments/:cid`
- `GET /api/v1/projects/:id/issues/:iid/activity`
- `GET /api/v1/notifications`
- `POST /api/v1/notifications/:id/read`
- `POST /api/v1/notifications/read-all`
- `GET /api/v1/users/me/activity`
