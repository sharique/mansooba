# Feature: First-Run Setup Wizard & Seed Data

## Overview

On a fresh install (no admin account exists), all routes redirect to `/setup` via a
global Nuxt middleware. A six-step wizard guides the first admin through account
creation, optional team member invite, optional project creation, optional sample data
import, and a summary screen.

## Implementation details

### Middleware redirect

- The Nuxt middleware checks `GET /api/v1/setup/status` on every navigation
- If no admin exists, the response instructs the frontend to redirect to `/setup`
- Once setup is complete, visiting `/setup` redirects to `/login`

### Setup endpoint self-deactivation

- `POST /api/v1/setup/admin` is a public, rate-limited endpoint (no JWT required)
- Once an admin account exists, this endpoint returns 409 on all subsequent calls
- All other setup endpoints require the JWT issued at the admin-creation step

### DEMO key conflict

If the admin creates a project with key `DEMO` during wizard Step 4, the sample-data import
falls back to key `SDEMO` (name: "Seed Demo Project"). A notice on the wizard summary screen
explains this.

### Wizard steps

1. **Welcome** — intro screen
2. **Admin account** — creates the first admin user; issues a JWT for subsequent steps
3. **Team member** — optional; creates a second user account
4. **Project** — optional; creates a project and optionally adds the team member as a
   member in a single atomic DB transaction
5. **Sample data** — optional; imports a demo project (`DEMO` key), 1 active sprint, 7
   issues, 2 labels, and 2 comments in a single DB transaction
6. **Summary** — confirms what was created

### Step 5 retry behaviour

If the sample-data API call fails, the wizard shows an inline error and a **Try again**
button. After a second failure the button is replaced by permanent recovery instructions
pointing to the seed CLI (see below).

### Password complexity

Validated in real time on keystroke with per-rule pass/fail indicators:
- Minimum length
- At least one uppercase letter
- At least one lowercase letter
- At least one digit

Applied on the wizard admin-creation step and all password-change forms.

### Seed CLI

The same sample dataset from Step 5 can be imported from the command line:

```bash
cd backend
go run ./cmd/seed
```

- Requires setup wizard to have been completed (admin account must exist)
- Idempotent: running it twice is safe — it checks for existing records before inserting

#### CLI output

**Success:**
```
Seed data created:
  Project:  Mansooba Demo [DEMO]
  Sprint:   Sprint 1 (active)
  Issues: 7  Labels: 2  Comments: 2
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

## API endpoints

See [arch-api.md](arch-api.md). Setup routes:

- `GET /api/v1/setup/status`
- `POST /api/v1/setup/admin` (public, rate-limited)
- `POST /api/v1/setup/member` (JWT required)
- `POST /api/v1/setup/project` (JWT required)
- `POST /api/v1/setup/sample-data` (JWT required)
