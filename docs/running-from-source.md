# Running from source

Use this guide when you want a tight Go + Node dev loop without building Docker images. The backend and frontend run natively on your machine.

**Quickest path:** `docker compose up --build` (see [`running-locally-using-docker.md`](running-locally-using-docker.md)) — Docker Compose is the recommended default.

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

See `backend/.env.example` for the full variable reference.

---

## 2. Start Mailpit

Mailpit captures outbound SMTP so password-reset emails don't reach real inboxes. Start it from `compose.yml`:

```sh
cd code
docker compose up mailpit -d
```

Mailpit inbox: **http://localhost:8025**

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
