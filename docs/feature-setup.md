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
- Once setup is complete, visiting `/setup` redirects to `/` (the dashboard), not `/login`

### Setup endpoint self-deactivation

- `POST /api/v1/setup/admin` is a public, rate-limited endpoint (no JWT required)
- Once an admin account exists, this endpoint returns 409 on all subsequent calls
- All other setup endpoints require the JWT issued at the admin-creation step

### DEMO key conflict

If the admin creates a project with key `DEMO` during the wizard's Project step, the
sample-data import falls back to key `SDEMO` (name: "Seed Demo Project"). A notice on the
wizard summary screen explains this.

### Wizard steps

Steps are 0-indexed in the UI (`Step X of 5` in `WizardShell.vue`) and in the setup
store:

0. **Welcome** — intro screen
1. **Admin account** — creates the first admin user; issues a JWT for subsequent steps
2. **Team member** — optional; creates a second user account
3. **Project** — optional; creates a project and optionally adds the team member as a
   member. This is **not** wrapped in a single DB transaction — project creation and the
   member-add are two independent service calls, so a failure partway through can leave
   the project created without the member added.
4. **Sample data** — optional; imports a demo project (`DEMO` key), 1 active sprint, 7
   issues, 2 labels, and 2 comments — this step *is* wrapped in a single DB transaction
5. **Summary** — confirms what was created

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
- `POST /api/v1/setup/user` (JWT required)
- `POST /api/v1/setup/project` (JWT required)
- `POST /api/v1/setup/seed` (JWT required)
