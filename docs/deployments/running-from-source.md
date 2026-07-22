# Running from source

Use this guide when you want a tight Go + Node dev loop without building Docker images. The backend and frontend run natively on your machine.

**Quickest path:** `docker compose up --build` (see [`running-locally-using-docker.md`](running-locally-using-docker.md)) — Docker Compose is the recommended default.

---

## Contents

- [Prerequisites](#prerequisites)
- [1. Environment](#1-environment)
- [2. Start Mailpit and LocalStack](#2-start-mailpit-and-localstack)
- [3. Backend](#3-backend)
- [4. Frontend](#4-frontend)
- [Running tests](#running-tests)

---

## Prerequisites

- Go 1.21+
- Node 22 + npm 10
- Docker (for Mailpit; needed for password-reset email in dev)

---

## 1. Environment

```sh
cp backend/.env.example backend/.env
# Edit backend/.env as needed — SQLite defaults work out of the box
```

Key variables:

| Variable | Default | Description |
|---|---|---|
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

## 2. Start Mailpit and LocalStack

Mailpit captures outbound SMTP so password-reset emails don't reach real inboxes. LocalStack
provides local S3-compatible storage for issue attachments. Start both from `compose.yml` without
starting the rest of the stack:

```sh
cd code
docker compose up mailpit localstack localstack-init -d
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
```
