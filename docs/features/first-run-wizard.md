# First-Run Wizard

This guide explains Mansooba's first-run setup wizard and the related `go run ./cmd/seed` CLI utility.

## Wizard step sequence

| Step | Screen | Description |
|------|--------|-------------|
| 0 | Welcome | Start page — introduces the wizard |
| 1 | Admin account | Create the platform admin (email + password) |
| 2 | Team member | Optionally invite a first team member |
| 3 | First project | Optionally create a project |
| 4 | Sample data | Choose to import example data or start clean |
| 5 | Complete | Summary and link to the dashboard |

The wizard is shown once on first boot (when no admin user exists). After the admin account is created the remaining steps require the admin JWT.

## Step 4 — Import example data

When the admin chooses **Import example data**, the backend endpoint `POST /api/v1/setup/seed` runs a transactional seed that creates:

| Entity | Count | Details |
|--------|-------|---------|
| Project | 1 | Key: `DEMO`, Name: `Mansooba Demo` |
| Sprints | 3 | `Sprint 1` completed, `Sprint 2` active, `Sprint 3` planned; dates are relative to the day of the import |
| Issues | 20 | Mix of task/story/bug with story points: 6 in the completed sprint (all done), 7 in the active sprint (spread over every board column), 5 in the planned sprint, 2 in the backlog |
| Labels | 2 | `bug` (#e11d48), `feature` (#3b82f6) |
| Comments | 2 | On the login-bug and Kanban-story issues |

Done issues carry completion dates, so the Reports page has a burndown for the active sprint and the completed one, and a velocity bar for the completed sprint. Some issues are assigned to the admin, so My Desk and the overview show tasks.

### DEMO project key conflict

If the admin already created a project with key `DEMO` during wizard Step 3, the seed falls back to key `SDEMO` with the name **Seed Demo Project**. A notice on the completion screen explains this.

### Retry behaviour

If the seed API call fails, the wizard shows an inline error and a **Try again** button. After a second failure the button is replaced by permanent recovery instructions (see CLI section below).

### Choosing "Start with a clean workspace"

The admin can skip Step 4 at any time. The workspace remains empty and seed data can be imported later using the CLI.

## CLI fallback — `go run ./cmd/seed`

The seed CLI is the recovery path when Step 4 was skipped or failed.

### Prerequisites

- The setup wizard must have been completed (at least Step 1 — admin account created).
- The `DB_DRIVER` and `DB_DSN` environment variables must match the running server.

### Usage

```bash
# From the backend/ directory:
go run ./cmd/seed
```

### Output

**Success:**
```
Seed data created:
  Project:  Mansooba Demo [DEMO]
  Sprints:  3 (1 completed, 1 active, 1 planned)
  Issues: 20  Labels: 2  Comments: 2
```

**Already present (idempotent):**
```
Seed data already present — skipping.
```

**No admin found:**
```
Error: run the setup wizard first (no admin user found)
exit status 1
```

The command is idempotent — running it twice is safe.

## User management (admin-controlled registration)

`POST /api/v1/auth/register` is **not** a public endpoint. It requires:
1. A valid admin JWT in the `Authorization: Bearer <token>` header.
2. The caller must have `is_admin = true`.

Non-admin callers receive `403 Forbidden`. Unauthenticated callers receive `401 Unauthorized`.

**How admins create new user accounts:**

1. Log in as an admin.
2. Navigate to `/system/createuser` (linked from the sidebar's System section).
3. Fill in the new user's details and submit.

Unauthenticated users who visit `/system/createuser` are redirected to `/login`.
Authenticated non-admin users are redirected to `/system/users`.
