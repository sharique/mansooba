# Running locally with Docker

This guide covers how to run the full stack (Go backend + Nuxt frontend + MinIO object storage) locally using containers.

The simplest path is **Docker Compose** with the bundled `compose.yml` — it starts everything with a single command and requires no additional tooling.

---

## Environment variables reference

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `8080` | Port the Go API listens on |
| `DB_DRIVER` | `sqlite` | `sqlite`, `postgres` / `postgresql`, `mysql` / `mariadb` |
| `DB_DSN` | `./dev.db` | SQLite path **or** database connection string |
| `DB_MAX_OPEN_CONNS` | `0` | Max open DB connections (`0` = unlimited; SQLite is always capped at 1) |
| `DB_MAX_IDLE_CONNS` | `2` | Max idle DB connections |
| `DB_CONN_MAX_LIFETIME` | `0` | Max connection lifetime (e.g. `5m`; `0` = never expire) |
| `JWT_SECRET` | *(required)* | Secret used to sign JWTs — use a long random string |
| `JWT_ACCESS_TTL` | `15m` | Access token lifetime |
| `JWT_REFRESH_TTL` | `168h` | Refresh token lifetime |
| `CORS_ORIGINS` | `http://localhost:3000` | Comma-separated allowed origins |
| `LOG_LEVEL` | `debug` | `debug`, `info`, `warn`, `error` |
| `BODY_SIZE_LIMIT` | `4M` | Max request body size (rejects larger payloads with 413) |
| `REQUEST_TIMEOUT` | `30s` | Per-request timeout (returns 503 when exceeded) |
| `AUTH_RATE_LIMIT` | `20` | Max login/register requests per second per IP |
| `SHUTDOWN_TIMEOUT` | `30s` | Graceful shutdown window |
| `STORAGE_ENDPOINT` | `http://localhost:9000` | S3-compatible endpoint URL (leave empty for AWS S3) |
| `STORAGE_BUCKET` | `mansooba` | Bucket name |
| `STORAGE_ACCESS_KEY_ID` | `minioadmin` | Access key (MinIO root user in dev) |
| `STORAGE_SECRET_ACCESS_KEY` | `minioadmin` | Secret key (MinIO root password in dev) |
| `STORAGE_REGION` | `us-east-1` | AWS region (MinIO ignores this; any string works) |
| `STORAGE_PRESIGN_TTL` | `1h` | How long pre-signed download URLs remain valid |
| `STORAGE_USE_PATH_STYLE` | `true` | Must be `true` for MinIO and most self-hosted S3 alternatives |
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
| `minio` | S3-compatible object store for file uploads, port 9000 (API) + 9001 (console) |
| `minio-init` | One-shot job that creates the `mansooba` bucket before the backend starts |
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

# Stop and wipe all data (SQLite + MinIO volumes)
docker compose down -v
```

App is at **http://localhost:3000** · API at **http://localhost:8080** · MinIO console at **http://localhost:9001** (user: `minioadmin`, password: `minioadmin`) · Mail inbox at **http://localhost:8025**

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

Run MinIO in Docker and start the backend and frontend natively for instant code feedback.

```bash
# Start only MinIO (and create the bucket)
docker compose up minio minio-init

# Backend (separate terminal)
cd backend
JWT_SECRET=dev-secret \
STORAGE_ENDPOINT=http://localhost:9000 \
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
STORAGE_ENDPOINT=http://localhost:9000 \
STORAGE_USE_PATH_STYLE=true \
go run ./cmd/server

# MariaDB
cd backend
DB_DRIVER=mysql \
DB_DSN="mansooba:mansooba@tcp(localhost:3306)/mansooba?charset=utf8mb4&parseTime=True&loc=Local" \
JWT_SECRET=dev-secret \
STORAGE_ENDPOINT=http://localhost:9000 \
STORAGE_USE_PATH_STYLE=true \
go run ./cmd/server
```

> MinIO must also be running for the backend to fully start. See the **Development mode** section for how to run MinIO alone from `compose.yml`.

### Full compose override (Postgres + MinIO)

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

## Option 3 — ddev

[ddev](https://ddev.readthedocs.io) supports custom project types and is suited for polyglot stacks.

> **Note**: The ddev config below does not include MinIO. You will need to add a `docker-compose.minio.yaml` service file analogous to the `minio` and `minio-init` services in `compose.yml`, and point `STORAGE_ENDPOINT` at it.

### Install ddev

```bash
# macOS
brew install ddev/ddev/ddev

# Linux / WSL2
curl -fsSL https://ddev.com/install.sh | bash
```

### Configuration

**`.ddev/config.yaml`** (at `code/`)

```yaml
name: mansooba
type: generic
docroot: ""
webserver_type: generic
router_http_port: "3000"
router_https_port: "3443"
database:
  type: postgres
  version: "16"
```

**`.ddev/docker-compose.backend.yaml`**

```yaml
services:
  backend:
    image: golang:alpine
    working_dir: /app
    volumes:
      - ../backend:/app
    command: go run ./cmd/server
    environment:
      SERVER_PORT: "8080"
      DB_DRIVER: postgres
      DB_DSN: "host=db port=5432 user=db password=db dbname=db sslmode=disable"
      JWT_SECRET: "ddev-dev-secret"
      CORS_ORIGINS: "http://mansooba.ddev.site:3000"
      LOG_LEVEL: debug
      STORAGE_ENDPOINT: "http://minio:9000"
      STORAGE_USE_PATH_STYLE: "true"
    ports:
      - "8080:8080"
    depends_on:
      - db
```

**`.ddev/docker-compose.frontend.yaml`**

```yaml
services:
  frontend:
    image: node:22-alpine
    working_dir: /app
    volumes:
      - ../frontend:/app
    command: sh -c "npm install && npm run dev"
    environment:
      NUXT_PUBLIC_API_BASE_URL: "http://backend:8080/api/v1"
    ports:
      - "3000:3000"
    depends_on:
      - backend
```

### Running with ddev

```bash
cd code
ddev start

ddev logs -s backend
ddev logs -s frontend

ddev stop
ddev delete --omit-snapshot
```

---

## Option 4 — Lando

[Lando](https://lando.dev) is a Docker-based dev tool with first-class support for custom multi-service stacks.

> **Note**: Same caveat as ddev — MinIO needs to be added as a custom service.

### Install Lando

```bash
brew install lando   # macOS
# Linux/Windows: https://lando.dev/download
```

### Configuration

**`.lando.yml`** (at `code/`)

```yaml
name: mansooba
recipe: lamp

services:
  db:
    type: postgres:16
    portforward: 5432
    creds:
      user: mansooba
      password: mansooba
      database: mansooba

  backend:
    type: go:1.21
    ssl: false
    command: go run ./cmd/server
    overrides:
      environment:
        SERVER_PORT: "8080"
        DB_DRIVER: postgres
        DB_DSN: "host=database port=5432 user=mansooba password=mansooba dbname=mansooba sslmode=disable"
        JWT_SECRET: "lando-dev-secret"
        CORS_ORIGINS: "http://localhost:3000"
        LOG_LEVEL: debug

  frontend:
    type: node:22
    ssl: false
    command: npm run dev
    overrides:
      environment:
        NUXT_PUBLIC_API_BASE_URL: "http://backend:8080/api/v1"

proxy:
  frontend:
    - mansooba.lndo.site:3000
  backend:
    - api.mansooba.lndo.site:8080
```

### Running with Lando

```bash
cd code
lando start
lando go test ./...
lando npm run typecheck
lando stop
lando destroy
```

---

## Troubleshooting

**Backend fails to start — "failed to initialise object storage"**  
The `STORAGE_*` variables are missing or the MinIO container isn't ready yet. If using `compose.yml`, verify the `minio-init` job completed:

```bash
docker compose logs minio-init
```

**`jwt: JWT_SECRET must not be empty` panic**  
Set `JWT_SECRET` in the environment. The backend refuses to start without it.

**Port conflicts**  
If `:9000`, `:8080`, or `:3000` are already in use, stop the conflicting service or change the host port mapping in `compose.yml`:

```yaml
minio:
  ports:
    - "9002:9000"   # MinIO API on host 9002
```

**Frontend shows API errors / blank page**  
`CORS_ORIGINS` must match the exact origin you're accessing from (scheme + host + port). If accessing via `http://localhost:3000`, set `CORS_ORIGINS=http://localhost:3000`.

**Health endpoint shows `"storage":"error"`**  
The backend is running but can't reach MinIO. Check that the `minio` container is healthy:

```bash
docker compose ps minio
docker compose logs minio
```
