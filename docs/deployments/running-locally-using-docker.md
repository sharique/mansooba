# Running locally with Docker

This guide covers how to run the full stack (Go backend + Nuxt frontend + LocalStack S3-compatible object storage) locally using containers.

The simplest path is **Docker Compose** with the bundled `compose.yml` — it starts everything with a single command and requires no additional tooling.

---

## Contents

- [Environment variables reference](#environment-variables-reference)
  - [Server](#server)
  - [JWT / Authentication](#jwt-authentication)
  - [Database](#database)
  - [Database idle auto-stop (AWS RDS only — no-op locally)](#database-idle-auto-stop-aws-rds-only-no-op-locally)
  - [S3 / Attachment storage](#s3-attachment-storage)
  - [SMTP / Email (Mailpit)](#smtp-email-mailpit)
- [Option 1 — Docker Compose (recommended)](#option-1-docker-compose-recommended)
  - [Running](#running)
  - [First run](#first-run)
  - [Catching outbound emails (Mailpit)](#catching-outbound-emails-mailpit)
  - [Rebuilding after code changes](#rebuilding-after-code-changes)
- [Development mode (hot reload)](#development-mode-hot-reload)
  - [Running backend tests](#running-backend-tests)
  - [Running frontend tests](#running-frontend-tests)
- [Option 2 — Docker Compose with PostgreSQL or MariaDB](#option-2-docker-compose-with-postgresql-or-mariadb)
  - [Quick database containers](#quick-database-containers)
  - [Full compose override (Postgres + LocalStack)](#full-compose-override-postgres-localstack)
- [Troubleshooting](#troubleshooting)

---

## Environment variables reference

### Server

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `8080` | Port the Go API listens on |
| `CORS_ORIGINS` | `http://localhost:3000` | Comma-separated allowed origins |
| `LOG_LEVEL` | `debug` | `debug`, `info`, `warn`, `error` |
| `BODY_SIZE_LIMIT` | `4M` | Max request body size (rejects larger payloads with 413) |
| `REQUEST_TIMEOUT` | `30s` | Per-request timeout (returns 503 when exceeded) |
| `AUTH_RATE_LIMIT` | `20` | Max login/register requests per second per IP |
| `SHUTDOWN_TIMEOUT` | `30s` | Graceful shutdown window |

### JWT / Authentication

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET` | *(required)* | Secret used to sign JWTs — use a long random string |
| `JWT_ACCESS_TTL` | `15m` | Access token lifetime |
| `JWT_REFRESH_TTL` | `168h` | Refresh token lifetime |

### Database

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_DRIVER` | `sqlite` | `sqlite`, `postgres` / `postgresql`, `mysql` / `mariadb` |
| `DB_DSN` | `./dev.db` | SQLite path **or** database connection string |
| `DB_MAX_OPEN_CONNS` | `0` | Max open DB connections (`0` = unlimited; SQLite is always capped at 1) |
| `DB_MAX_IDLE_CONNS` | `2` | Max idle DB connections |
| `DB_CONN_MAX_LIFETIME` | `0` | Max connection lifetime (e.g. `5m`; `0` = never expire) |

### Database idle auto-stop (AWS RDS only — no-op locally)

Powers the demo deployment's cost-saving auto-stop/wake-on-hit behavior (spec 010, ADR-030). **You
never need to touch these vars for local development, including Option 2 below (local Postgres via
docker-compose)** — the backend confirms `DB_DSN`'s hostname is the *specific* AWS RDS instance named
by `RDS_INSTANCE_IDENTIFIER` (it must both end in `.rds.amazonaws.com` and start with
`<RDS_INSTANCE_IDENTIFIER>.`) before engaging at all. A local Postgres/MySQL/MariaDB container's DSN
host (`localhost`, or a Docker service name) can never satisfy that check, so this feature stays
inert for every local setup — even if `RDS_AUTOSTOP_ENABLED` and `RDS_INSTANCE_IDENTIFIER` both
happen to be set, e.g. from a copied `.env` file.

| Variable | Default | Description |
|----------|---------|-------------|
| `RDS_AUTOSTOP_ENABLED` | `true` | Database idle auto-stop/wake-on-hit (spec 010, ADR-030). Only takes effect when `DB_DSN`'s hostname is confirmed as the specific AWS RDS instance named by `RDS_INSTANCE_IDENTIFIER` — always a no-op otherwise (including local Postgres/MySQL/MariaDB). |
| `RDS_INSTANCE_IDENTIFIER` | *(unset)* | The RDS instance identifier to stop/start. Required, and its value MUST match the leading label of `DB_DSN`'s host (e.g. identifier `mansooba-db` requires a DSN host like `mansooba-db.<random>.<region>.rds.amazonaws.com`) — that match is what confirms the feature should actually engage. |
| `RDS_IDLE_TIMEOUT` | `10m` | How long the database can sit idle before being stopped |
| `RDS_IDLE_CHECK_INTERVAL` | `1m` | How often the idle/pending-start check runs |
| `RDS_START_FAILURE_BOUND` | `3` | Consecutive failed start attempts before giving up |
| `AWS_REGION` | *(unset)* | Required on the real AWS deployment once auto-stop is enabled — the RDS SDK client needs an explicit region (unlike credentials, it isn't inferred from the EC2 instance automatically). Leave unset locally. |

### S3 / Attachment storage

Powers issue file attachments. The backend never requires storage connectivity to start —
it only touches `STORAGE_*` config when a file is actually uploaded, downloaded, or deleted.

| Variable | Default | Description |
|----------|---------|-------------|
| `STORAGE_ENDPOINT` | *(unset)* | S3-compatible endpoint URL; set to `http://localstack:4566` for the bundled LocalStack container, leave unset for real AWS S3 |
| `STORAGE_PRESIGN_ENDPOINT` | *(unset, falls back to `STORAGE_ENDPOINT`)* | Overrides the host baked into presigned download URLs. Needed only when `STORAGE_ENDPOINT` is a Docker-internal hostname (like `localstack`) that a browser on the host can't resolve — set to `http://localhost:4566` in that case. Real AWS S3's hostname is reachable identically everywhere, so production never needs this. |
| `STORAGE_BUCKET` | `mansooba-attachments` | Bucket name |
| `STORAGE_ACCESS_KEY_ID` | *(unset)* | LocalStack only — use `test`; leave unset in production (the EC2 instance's IAM role is used instead, see ADR-029) |
| `STORAGE_SECRET_ACCESS_KEY` | *(unset)* | LocalStack only — use `test` |
| `STORAGE_REGION` | `us-east-1` | AWS region (LocalStack ignores this; any string works) |
| `STORAGE_PRESIGN_TTL` | `1h` | How long pre-signed download URLs remain valid |
| `STORAGE_USE_PATH_STYLE` | `false` | Set `true` for LocalStack and most self-hosted S3 alternatives |

### SMTP / Email (Mailpit)

| Variable | Default | Description |
|----------|---------|-------------|
| `SMTP_HOST` | `mailpit` | SMTP server host — use `mailpit` when running with Compose |
| `SMTP_PORT` | `1025` | SMTP port |
| `SMTP_FROM` | `noreply@mansooba.local` | Sender address for outbound email |

Connection string formats:

```
# PostgreSQL
host=<host> port=5432 user=<user> password=<pass> dbname=<db> sslmode=disable

# MySQL / MariaDB
<user>:<pass>@tcp(<host>:3306)/<db>?charset=utf8mb4&parseTime=True&loc=Local
```

---

## Option 1 — Docker Compose (recommended)

The `compose.yml` at the repo root starts the complete dev stack:

| Service | What it does |
|---------|-------------|
| `backend` | Go API server, SQLite database, port 8080 |
| `frontend` | Static Nuxt app served by nginx, port 3000 |
| `localstack` | S3-compatible object store for issue attachments (community edition), port 4566 |
| `localstack-init` | One-shot job that creates the `mansooba-attachments` bucket before the backend starts |
| `mailpit` | SMTP mail catcher — captures all outbound email; web inbox on port 8025 |

The frontend nginx proxies `/api/` to the backend, so the browser only needs to talk to one origin (`localhost:3000`).

### Running

```bash
cd code

# Build images and start all services
docker compose up --build

# Run in the background
docker compose up --build -d

# Follow logs
docker compose logs -f

# Stop
docker compose down

# Stop and wipe all data (SQLite + LocalStack volumes)
docker compose down -v
```

App is at **http://localhost:3000** · API at **http://localhost:8080** · LocalStack health check at **http://localhost:4566/_localstack/health** · Mail inbox at **http://localhost:8025**

LocalStack's community edition has no web console — inspect the bucket with the AWS CLI instead:

```bash
docker run --rm --network host -e AWS_ACCESS_KEY_ID=test -e AWS_SECRET_ACCESS_KEY=test \
  amazon/aws-cli:latest --endpoint-url http://localhost:4566 s3 ls s3://mansooba-attachments --recursive
```

### First run

When the app starts for the first time there are no users. Navigate to **http://localhost:3000/setup** to run the superadmin setup wizard — it creates the first account and seeds initial organisation settings.

### Catching outbound emails (Mailpit)

Compose starts a [Mailpit](https://mailpit.axllent.org) container that captures every outbound SMTP message — nothing reaches real inboxes.

Open **http://localhost:8025** to browse the captured inbox after triggering any email flow (e.g. password reset).

When the backend is configured for email delivery, set these env vars to point at Mailpit:

```bash
SMTP_HOST=mailpit
SMTP_PORT=1025
SMTP_FROM=noreply@mansooba.local
```

In `compose.yml` the Mailpit container is already present; just add the three vars to the `backend` environment block when implementing email delivery (007-email-delivery).

### Rebuilding after code changes

```bash
# Rebuild only the backend image (faster than rebuilding everything)
docker compose build backend
docker compose up -d --no-deps backend

# Rebuild only the frontend
docker compose build frontend
docker compose up -d --no-deps frontend
```

---

## Development mode (hot reload)

Run LocalStack in Docker and start the backend and frontend natively for instant code feedback.

```bash
# Start only LocalStack (and create the bucket)
docker compose up localstack localstack-init

# Backend (separate terminal)
cd backend
JWT_SECRET=dev-secret \
STORAGE_ENDPOINT=http://localhost:4566 \
STORAGE_ACCESS_KEY_ID=test \
STORAGE_SECRET_ACCESS_KEY=test \
STORAGE_USE_PATH_STYLE=true \
go run ./cmd/server

# Frontend (separate terminal)
cd frontend
NUXT_PUBLIC_API_BASE_URL=http://localhost:8080/api/v1 npm run dev
```

The backend hot-reloads with [Air](https://github.com/air-verse/air) if installed:

```bash
cd backend
go install github.com/air-verse/air@latest
air
```

### Running backend tests

```bash
cd backend
go test ./...
go vet ./...
```

### Running frontend tests

```bash
cd frontend
npx vitest run       # all tests once
npx vitest           # watch mode
npx nuxi typecheck   # TypeScript
```

---

## Option 2 — Docker Compose with PostgreSQL or MariaDB

The default `compose.yml` uses SQLite. Swap in a real database by running a Docker container for it and pointing the backend at it. This is useful for testing migrations, replicating a staging/production schema, or if you simply prefer a server-based database locally.

### Quick database containers

Spin up a database with a single `docker run` — no compose file needed:

**PostgreSQL 16**
```bash
docker run --rm --name pg-local \
  -e POSTGRES_USER=mansooba \
  -e POSTGRES_PASSWORD=mansooba \
  -e POSTGRES_DB=mansooba \
  -p 5432:5432 \
  postgres:17-alpine
```

**MariaDB 11**
```bash
docker run --rm --name mariadb-local \
  -e MYSQL_USER=mansooba \
  -e MYSQL_PASSWORD=mansooba \
  -e MYSQL_DATABASE=mansooba \
  -e MYSQL_ROOT_PASSWORD=root \
  -p 3306:3306 \
  mariadb:11
```

Then start the backend natively (or in Docker) with the matching env vars:

```bash
# PostgreSQL
cd backend
DB_DRIVER=postgres \
DB_DSN="host=localhost port=5432 user=mansooba password=mansooba dbname=mansooba sslmode=disable" \
JWT_SECRET=dev-secret \
STORAGE_ENDPOINT=http://localhost:4566 \
STORAGE_ACCESS_KEY_ID=test \
STORAGE_SECRET_ACCESS_KEY=test \
STORAGE_USE_PATH_STYLE=true \
go run ./cmd/server

# MariaDB
cd backend
DB_DRIVER=mysql \
DB_DSN="mansooba:mansooba@tcp(localhost:3306)/mansooba?charset=utf8mb4&parseTime=True&loc=Local" \
JWT_SECRET=dev-secret \
STORAGE_ENDPOINT=http://localhost:4566 \
STORAGE_ACCESS_KEY_ID=test \
STORAGE_SECRET_ACCESS_KEY=test \
STORAGE_USE_PATH_STYLE=true \
go run ./cmd/server
```

> The backend starts fine without LocalStack running — it only needs storage connectivity when
> an attachment is actually uploaded/downloaded/deleted. See the **Development mode** section for
> how to run LocalStack alone from `compose.yml` if you're working on that feature.

### Full compose override (Postgres + LocalStack)

If you want Docker Compose to manage the database too, create a `compose.override.yml` at `code/`:

```yaml
services:
  db:
    image: postgres:17-alpine
    restart: unless-stopped
    environment:
      POSTGRES_USER: mansooba
      POSTGRES_PASSWORD: mansooba
      POSTGRES_DB: mansooba
    ports:
      - "5432:5432"
    volumes:
      - pg_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U mansooba"]
      interval: 5s
      timeout: 5s
      retries: 10

  backend:
    depends_on:
      db:
        condition: service_healthy
    environment:
      DB_DRIVER: postgres
      DB_DSN: "host=db port=5432 user=mansooba password=mansooba dbname=mansooba sslmode=disable"

volumes:
  pg_data:
```

For **MariaDB** instead, swap `db` to:

```yaml
  db:
    image: mariadb:11
    restart: unless-stopped
    environment:
      MYSQL_USER: mansooba
      MYSQL_PASSWORD: mansooba
      MYSQL_DATABASE: mansooba
      MYSQL_ROOT_PASSWORD: root
    ports:
      - "3306:3306"
    volumes:
      - mariadb_data:/var/lib/mysql
    healthcheck:
      test: ["CMD", "healthcheck.sh", "--connect", "--innodb_initialized"]
      interval: 5s
      timeout: 5s
      retries: 10

  backend:
    depends_on:
      db:
        condition: service_healthy
    environment:
      DB_DRIVER: mysql
      DB_DSN: "mansooba:mansooba@tcp(db:3306)/mansooba?charset=utf8mb4&parseTime=True&loc=Local"

volumes:
  mariadb_data:
```

Start with both files:

```bash
docker compose -f compose.yml -f compose.override.yml up --build
```

Docker Compose merges the two files — the override adds the `db` service and patches the backend's `DB_DRIVER` and `DB_DSN`.

---

## Troubleshooting

**`jwt: JWT_SECRET must not be empty` panic**  
Set `JWT_SECRET` in the environment. The backend refuses to start without it — this is the only
env var the backend requires at boot. It does *not* need working `STORAGE_*`/database connectivity
to start; those are checked lazily, on the first request that actually needs them.

**Attachment upload/download/delete returns 502 "storage temporarily unavailable"**  
The backend can't reach LocalStack (or, in production, real S3). If using `compose.yml`, verify
the `localstack-init` job completed and `localstack` itself is healthy:

```bash
docker compose logs localstack-init
docker compose ps localstack
```

**Port conflicts**  
If `:4566`, `:8080`, or `:3000` are already in use, stop the conflicting service or change the host
port mapping in `compose.yml`:

```yaml
localstack:
  ports:
    - "4567:4566"   # LocalStack API on host 4567 — also update STORAGE_ENDPOINT accordingly
```

**Frontend shows API errors / blank page**  
`CORS_ORIGINS` must match the exact origin you're accessing from (scheme + host + port). If accessing via `http://localhost:3000`, set `CORS_ORIGINS=http://localhost:3000`.

**Attachment download link doesn't work when accessed from a different machine**  
`STORAGE_PRESIGN_ENDPOINT` (or `STORAGE_ENDPOINT` if the override is unset) must be a host your
*browser* can resolve, not just the backend container. `http://localhost:4566` only works when the
browser is on the same machine as the LocalStack container.
