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

### Self-service password change

Distinct from the password-reset flow above — this is for an already-authenticated user
changing their own password from Settings → Profile, not a forgotten-password recovery.
See spec [012-change-password](../../../specs/012-change-password/spec.md) and
[ADR-032](../../../docs/decisions/ADR-032-password-change-session-invalidation.md).

1. User submits their current password + a new one (meeting the same complexity policy
   as registration) at `PUT /api/v1/auth/me/password`
2. The current password is verified with bcrypt; the new password must differ from it
   (checked against the current password only — no history is kept)
3. On success:
   - the new password is persisted
   - `TokenValidAfter` is set on the user record, which invalidates every other active
     session's refresh token on its next use (ADR-032) — the acting session's own tokens
     are reissued in the same response (a fresh `access_token` in the JSON body, a fresh
     `refresh_token` cookie) so it stays logged in
   - a confirmation email is sent (best-effort; delivery failure doesn't fail the request)
   - the change is recorded in System Logs (`password_change_success`)
4. On failure (wrong current password, weak/reused new password): the attempt is
   rejected, recorded in System Logs with a reason (`password_change_failed`), and a
   per-account counter of consecutive failures increments — reaching 3 in a row triggers
   a separate suspicious-activity alert email, distinct from the success-confirmation one
5. Rate-limited per source IP (`AUTH_RATE_LIMIT`), same as `/auth/login` — unlike this
   endpoint's `authMe` siblings (`/me`, `/me/avatar`, etc.), which are unrated

**Why the acting session needs reissued tokens**: `TokenValidAfter` invalidates every
refresh token issued *before* the change — including the one the acting session already
holds, since it was necessarily issued before this request. Without reissuing, the user
who just changed their password would also get logged out the next time their own
session tries to refresh. See ADR-032 for the full design rationale, including why the
comparison is truncated to whole-second precision (matches `jwt.NewNumericDate`'s own
truncation).

### First-run admin bootstrap (the one true self-service path)

Registration above is admin-only in steady state, but the very first admin account has
no admin to create it. That's handled by a separate, one-time setup flow — see
[setup.md](setup.md) and [first-run-wizard.md](first-run-wizard.md):
`GET /api/v1/setup/status`, `POST /api/v1/setup/admin` (public, rate-limited), plus
JWT-gated `POST /api/v1/setup/user`, `POST /api/v1/setup/project`, `POST
/api/v1/setup/seed`. Once an admin exists, this flow is permanently unavailable.

## API endpoints

See [api.md](../arch/api.md) for the full endpoint list. Key auth routes:

- `POST /api/v1/auth/login`
- `POST /api/v1/auth/logout`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/register` (admin JWT required)
- `POST /api/v1/auth/forgot-password`
- `POST /api/v1/auth/reset-password`
- `GET/PUT /api/v1/auth/me`
- `PUT /api/v1/auth/me/password` (rate-limited, unlike its `/me` siblings)
- `GET /api/v1/auth/me/activity`
- `GET /api/v1/auth/me/issues`
- `POST /api/v1/auth/me/avatar`
- `DELETE /api/v1/auth/me/avatar`
