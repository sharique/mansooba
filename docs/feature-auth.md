# Feature: Authentication & Security

## Overview

Mansooba uses a JWT-based authentication system with HttpOnly cookie refresh tokens.
Registration is admin-controlled — new accounts are created by admins, not via
self-service signup. Password resets are delivered by email. User profiles include
display name, email, timezone, and an optional avatar photo.

## Implementation details

### Token strategy

- Access tokens: short-lived JWT, signed and verified on every API request
- Refresh tokens: stored in HttpOnly, SameSite=Strict cookies (Secure flag enabled in
  production)
- On refresh: the backend validates the refresh token JTI against the `revoked_tokens`
  table; if the DB lookup fails, the request is rejected with 503 (fail-closed — no
  silent grants)

### Server-side logout / token revocation

- On logout, the refresh token's JTI is inserted into `revoked_tokens`
- Every token refresh checks this table; a hit means the session is revoked
- A background goroutine runs on a configurable interval to purge expired revocation
  records from the table

### Password reset flow

1. User submits their email at `POST /api/v1/auth/forgot-password`
2. A reset token is generated and emailed (caught by Mailpit in dev)
3. The token pre-fills the `/reset-password` page
4. On submit, `POST /api/v1/auth/reset-password` validates the token and updates the
   password

### Avatar storage

- Uploaded via `POST /api/v1/auth/me/avatar`
- Stored on local disk under `uploads/avatars/`
- Served publicly at `/uploads/*` without auth (ADR-026)
- Falls back to OKLCH-coloured initials when no photo is set (see `UserAvatar` component)

### Admin-controlled registration

- `POST /api/v1/auth/register` requires a valid admin JWT — self-service signup is
  disabled
- Admins create accounts via `/system/createuser`, which shows the same password
  complexity checklist (8+ chars, uppercase, lowercase, digit) as the first-run setup
  wizard's Admin step
- Unauthenticated requests → 401; non-admin requests → 403
- New account credentials are shared directly with the user by the admin

### First-run admin bootstrap (the one true self-service path)

Registration above is admin-only in steady state, but the very first admin account has
no admin to create it. That's handled by a separate, one-time setup flow — see
[feature-setup.md](feature-setup.md) and [first-run-wizard.md](first-run-wizard.md):
`GET /api/v1/setup/status`, `POST /api/v1/setup/admin` (public, rate-limited), plus
JWT-gated `POST /api/v1/setup/user`, `POST /api/v1/setup/project`, `POST
/api/v1/setup/seed`. Once an admin exists, this flow is permanently unavailable.

## API endpoints

See [arch-api.md](arch-api.md) for the full endpoint list. Key auth routes:

- `POST /api/v1/auth/login`
- `POST /api/v1/auth/logout`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/register` (admin JWT required)
- `POST /api/v1/auth/forgot-password`
- `POST /api/v1/auth/reset-password`
- `GET/PUT /api/v1/auth/me`
- `GET /api/v1/auth/me/activity`
- `GET /api/v1/auth/me/issues`
- `POST /api/v1/auth/me/avatar`
- `DELETE /api/v1/auth/me/avatar`
