# Running from source

Use this guide when you want a tight Go + Node dev loop without building Docker images. The backend and frontend run natively on your machine.

**Quickest path:** `docker compose up --build` (see [`running-locally-using-docker.md`](running-locally-using-docker.md)) — Docker Compose is the recommended default.

---

## Contents

- [Prerequisites](#prerequisites)
- [1. Environment](#1-environment)
- [2. Start Mailpit, LocalStack, and Loki](#2-start-mailpit-localstack-and-loki)
- [3. Backend](#3-backend)
- [4. Frontend](#4-frontend)
- [5. First run](#5-first-run)
- [Running tests](#running-tests)
- [Troubleshooting](#troubleshooting)

---

## Prerequisites

- Go 1.27 (the version declared in `backend/go.mod`; it wins over whatever you have installed)
- Node 24 LTS + npm 11 (the version declared in `frontend/.nvmrc`)
- Docker (for Mailpit; needed for password-reset email in dev)

> **Go toolchain:** Go downloads the toolchain named in `go.mod` automatically only when
> `GOTOOLCHAIN` is `auto` (the default). If your machine sets `GOTOOLCHAIN=local`, either
> install that Go release or switch it (`go env -w GOTOOLCHAIN=auto`); otherwise builds fail with
> "go.mod requires go >= 1.27 (running go 1.2x; GOTOOLCHAIN=local)".

---

## 1. Environment

```sh
cp backend/.env.example backend/.env
# Edit backend/.env as needed — SQLite defaults work out of the box
```

Key variables:

| Variable | Default | Description |
|---|---|---|
| `SERVER_PORT` | `8080` | Port the API listens on |
| `CORS_ORIGINS` | `http://localhost:3000` | Browser origins allowed to call the API; must include the frontend's address |
| `LOG_LEVEL` | `debug` in `.env.example` | Log verbosity |
| `JWT_SECRET` | *(required)* | HMAC secret used to sign all JWTs — use any long random string locally |
| `JWT_ACCESS_TTL` | `15m` | Access token lifetime |
| `JWT_REFRESH_TTL` | `168h` | Refresh token lifetime (7 days) |
| `APP_ENV` | `production` | Set to `development` to omit the `Secure` flag on cookies (required for plain HTTP local dev) |
| `DB_DRIVER` | `sqlite` | `sqlite`, `postgres`, `mysql` / `mariadb` |
| `DB_DSN` | `./dev.db` | SQLite file path, or a full connection string for Postgres/MySQL |
| `REVOKED_TOKEN_CLEANUP_INTERVAL` | `15m` | How often the background goroutine purges expired revoked-token records |
| `SMTP_HOST` | `mailpit` | SMTP server host; use `localhost` when running Mailpit outside Compose |
| `SMTP_PORT` | `1025` | SMTP port |
| `SMTP_FROM` | `noreply@mansooba.local` | Sender address for password-reset emails |
| `STORAGE_ENDPOINT` | *(unset)* | S3-compatible endpoint; set to `http://localhost:4566` for LocalStack, leave unset for real AWS S3 |
| `STORAGE_ACCESS_KEY_ID` / `STORAGE_SECRET_ACCESS_KEY` | *(unset)* | LocalStack only — use `test`/`test`; leave unset in production (IAM role instead) |
| `STORAGE_USE_PATH_STYLE` | `false` | Set `true` for LocalStack |

See `backend/.env.example` for the full variable reference.

The backend starts fine with no `STORAGE_*` vars set at all — it only touches storage when an
attachment is actually uploaded/downloaded/deleted, so you only need this section if you're
working on that feature.

---

## 2. Start Mailpit, LocalStack, and Loki

Mailpit captures outbound SMTP so password-reset emails don't reach real inboxes. LocalStack
provides local S3-compatible storage for issue attachments. Loki (011-system-logs) is the backing
store for System Logs, the admin-only audit trail. Start all three from `compose.yml` without
starting the rest of the stack:

```sh
cd code
docker compose up mailpit localstack localstack-init loki -d
```

Mailpit inbox: **http://localhost:8025**

Since the backend now runs natively (not inside the Compose network), point it at LocalStack via
`localhost` directly — no `STORAGE_PRESIGN_ENDPOINT` split needed here, unlike the Docker Compose
guide, because both the backend and your browser are on the host machine:

```sh
export STORAGE_ENDPOINT=http://localhost:4566
export STORAGE_ACCESS_KEY_ID=test
export STORAGE_SECRET_ACCESS_KEY=test
export STORAGE_USE_PATH_STYLE=true
```

`LOKI_BASE_URL`'s default (`http://localhost:3100`) already works unmodified here, for the same
reason as `STORAGE_ENDPOINT` above — Loki's container port is published to the host. But
`LOKI_RUNTIME_OVERRIDES_PATH` defaults to the *relative* path `./loki/runtime-overrides.yaml`,
resolved against whatever directory the backend process runs from — Step 3 below does `cd backend`
first, so left at its default this would resolve to `backend/loki/runtime-overrides.yaml`, which
doesn't exist (the real file, the one Loki's own container has bind-mounted, lives at
`code/loki/runtime-overrides.yaml`, one level up). Export the correct path as an absolute one so it
stays right regardless of working directory:

```sh
export LOKI_RUNTIME_OVERRIDES_PATH="$(pwd)/loki/runtime-overrides.yaml"
```

Run that while still in `code/` (as above) so the path resolves correctly regardless of which
directory you're in once you `cd backend` next.

Alloy (which tails Docker containers' own logs into Loki) and Grafana (the optional dashboard UI)
both still work if started the same way (`docker compose up alloy -d`, `docker compose --profile
observability up -d grafana`) — but Alloy is much less useful in this guide specifically, since the
backend and frontend aren't running in Docker at all here and so have no container logs for it to
collect; it would only pick up Mailpit/LocalStack/Loki's own output.

---

## 3. Backend

```sh
cd backend
go run ./cmd/server
# API at http://localhost:8080
# Health check: GET http://localhost:8080/health
```

For hot reload, install [Air](https://github.com/air-verse/air) and run `air` instead:

```sh
go install github.com/air-verse/air@latest
air
```

---

## 4. Frontend

```sh
cd frontend
npm install
npm run dev
# App at http://localhost:3000
```

The frontend calls the API at `http://localhost:8080/api/v1` by default. If the backend runs on
another host or port, start the dev server with `NUXT_PUBLIC_API_BASE_URL` set to the full API
URL (for example `NUXT_PUBLIC_API_BASE_URL=http://localhost:9090/api/v1 npm run dev`) and add the
frontend's origin to the backend's `CORS_ORIGINS`.

On the first start Vite may print "New dependencies found" and reload the page once; this is
normal. The chart libraries are pre-bundled so it does not happen again mid-session.

---

## 5. First run

With an empty database there are no users. Open **http://localhost:3000/setup** and follow the
wizard: it creates the administrator account (with its permanent username), and can optionally
create a second user, a first project and the example data. See
[the first-run wizard guide](../features/first-run-wizard.md).

The example data is a `DEMO` project with three sprints (one completed, one in progress, one
planned), 20 tickets with story points, two labels and two comments. If you skipped it, import it
later, once an administrator exists:

```sh
cd backend
go run ./cmd/seed
```

The import is skipped when the demo project already exists, so to see new demo data you need a
fresh database (delete `backend/dev.db` for SQLite).

---

## Running tests

```sh
# Backend
cd backend
go test ./...
go vet ./...

# Frontend
cd frontend
npx vitest run        # all tests once
npx vitest            # watch mode
npx nuxi typecheck    # TypeScript check
npm run build         # production build; run it before opening a PR
```

The storage tests (attachments, avatars) run against the LocalStack from step 2 and are skipped
when it is not running. To make a missing LocalStack a failure instead, set `REQUIRE_LOCALSTACK=1`.
LocalStack does not keep its bucket across restarts. If the tests fail with `NoSuchBucket`,
recreate it:

```sh
AWS_ACCESS_KEY_ID=test AWS_SECRET_ACCESS_KEY=test \
  aws --endpoint-url http://localhost:4566 s3 mb s3://mansooba-attachments --region us-east-1
```

---

## Troubleshooting

| Symptom | Fix |
|---|---|
| `go.mod requires go >= 1.27 (running go 1.2x; GOTOOLCHAIN=local)` | See the Go toolchain note under [Prerequisites](#prerequisites). |
| Browser console shows `Cannot read properties of null (reading 'emitsOptions')`, or pages break after a reload | The dev server's cache is stale, usually after switching branches or running `npm run build` while `npm run dev` was up. Stop the dev server, run `rm -rf frontend/.nuxt frontend/node_modules/.cache/vite`, restart it and hard-refresh the browser. |
| The frontend loads but every request fails with a CORS error | Add the frontend's origin to `CORS_ORIGINS` and restart the backend. |
| Password-reset email never arrives | Start Mailpit (step 2) and read the message at http://localhost:8025. |
