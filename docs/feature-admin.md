# Feature: System Administration

## Overview

Admins have access to a System section in the sidebar with three pages: platform
settings, user list, and create user. Non-admins cannot access these pages. The UI is
role-aware — the create affordances shown to a user depend on their project role.

## Implementation details

### Last-admin guard

The promote/demote admin and disable account actions check whether the target user is
the last active admin. If so, the operation is rejected — the platform must always have
at least one active admin.

### System settings schema

Settings are stored in a `system_settings` key-value table. Configurable fields:

| Key | Description |
|-----|-------------|
| `date_format` | Display date format (e.g. `YYYY-MM-DD`) |
| `time_format` | `12h` or `24h` |
| `timezone` | IANA timezone string (e.g. `America/New_York`) |
| `session_timeout` | Refresh token lifetime in minutes |
| `max_upload_size` | Maximum file upload size in bytes |

### User enable/disable model

- Disabled users cannot log in (login endpoint returns 403)
- Existing sessions are not immediately revoked on disable — they expire naturally unless
  the admin also triggers a logout for that user
- User records are never deleted from the DB — disable is a soft deactivation

### Role-aware UI

The create affordance shown depends on the user's role:
- **Admin** → dropdown with multiple creation options
- **Member** → single-action button
- **No project membership** → no create affordance shown

The sidebar System section is rendered only when `authStore.isAdmin` is true.

## API endpoints

See [arch-api.md](arch-api.md). Key admin routes:

- `GET/PUT /api/v1/system/settings`
- `GET /api/v1/system/users`
- `POST /api/v1/auth/register` (admin JWT required)
- `PUT /api/v1/system/users/:id/role`
- `PUT /api/v1/system/users/:id/status`
