# Feature: System Administration

## Overview

Admins have access to a System section in the sidebar with three pages: platform
settings, user list, and create user. Non-admins cannot access these pages.

## Implementation details

### Last-admin guard

The promote/demote admin and disable account actions check whether the target user is
the last active admin. If so, the operation is rejected — the platform must always have
at least one active admin.

### Global settings schema

Settings are stored in a `global_settings` key-value table (`GlobalSetting` struct).
Configurable fields:

| Key | Description |
|-----|-------------|
| `organization_name` | Display name for the org, shown in the UI |
| `date_format` | Display date format (e.g. `YYYY-MM-DD`) |
| `time_format` | `12h` or `24h` |
| `locale` | Locale string |
| `week_start_day` | Which day the week starts on |

Note: per-user timezone is a field on `User`, not a global setting — there's no
platform-wide timezone. There's also no admin-configurable session timeout (refresh
token lifetime is fixed via the `JWT_REFRESH_TTL` env var) or upload-size setting (the
attachment upload cap is a hardcoded route-level body limit, not an admin setting).

### User enable/disable model

- Disabled users cannot log in (login endpoint returns 401)
- Disabling doesn't revoke the user's current access token immediately (it's still valid
  until it expires, typically minutes), but the *next* token refresh attempt is rejected
  — `AuthService.Refresh` checks `IsActive` and fails closed. There's no separate
  admin-triggered force-logout action; the short access-token TTL is what bounds the
  window.
- User records are never deleted from the DB — disable is a soft deactivation

The sidebar System section is rendered only when `authStore.isAdmin` is true.

## API endpoints

See [arch-api.md](arch-api.md). Key admin routes:

- `GET /api/v1/settings`
- `PATCH /api/v1/settings`
- `GET /api/v1/admin/users`
- `PATCH /api/v1/admin/users/:id` — body may include `is_admin` and/or `is_active`
- `POST /api/v1/auth/register` (admin JWT required)
