# Running locally with Docker and PostgreSQL

This guide covers three ways to run the full stack (Go backend + Nuxt frontend + PostgreSQL) locally using containers:

1. **Plain Docker Compose** — the simplest approach, no extra tooling
2. **ddev** — lightweight dev environment manager
3. **Lando** — flexible local dev platform

---

## Environment variables reference

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `8080` | Port the Go API listens on |
| `DB_DRIVER` | `sqlite` | `sqlite`, `postgres` / `postgresql`, `mysql` / `mariadb` |
| `DB_DSN` | `./dev.db` | SQLite path **or** database connection string |
| `DB_MAX_OPEN_CONNS` | `0` | Max open DB connections (`0` = unlimited) |
| `DB_MAX_IDLE_CONNS` | `2` | Max idle DB connections |
| `DB_CONN_MAX_LIFETIME` | `0` | Max connection lifetime (e.g. `5m`; `0` = never expire) |
| `JWT_SECRET` | *(required)* | Secret used to sign JWTs — use a long random string |
| `JWT_ACCESS_TTL` | `15m` | Access token lifetime |
| `JWT_REFRESH_TTL` | `168h` | Refresh token lifetime |
| `CORS_ORIGINS` | `http://localhost:3000` | Comma-separated allowed origins |
| `LOG_LEVEL` | `debug` | `debug`, `info`, `warn`, `error` |

Postgres DSN format:
```
host=localhost port=5432 user=jira password=jira dbname=jira sslmode=disable
```

---

## Option 1 — Docker Compose

### Files to create

**`backend/Dockerfile`**

```dockerfile
# ── Build stage ───────────────────────────────────────────────────────────────
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
```

> `CGO_ENABLED=0` is safe when using the Postgres driver — it is pure Go.  
> Keep SQLite out of Docker builds; use Postgres instead.

---

**`frontend/Dockerfile`**

```dockerfile
# ── Build stage ───────────────────────────────────────────────────────────────
FROM node:22-alpine AS builder

WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

COPY . .
RUN npm run build

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM node:22-alpine
WORKDIR /app
COPY --from=builder /app/.output ./.output
EXPOSE 3000
CMD ["node", ".output/server/index.mjs"]
```

---

**`docker-compose.yml`** (place at the repo root `code/`)

```yaml
services:

  db:
    image: postgres:16-alpine
    restart: unless-stopped
    environment:
      POSTGRES_USER: jira
      POSTGRES_PASSWORD: jira
      POSTGRES_DB: jira
    ports:
      - "5432:5432"
    volumes:
      - pg_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U jira"]
      interval: 5s
      timeout: 5s
      retries: 10

  backend:
    build: ./backend
    restart: unless-stopped
    depends_on:
      db:
        condition: service_healthy
    ports:
      - "8080:8080"
    environment:
      SERVER_PORT: "8080"
      DB_DRIVER: postgres
      DB_DSN: "host=db port=5432 user=jira password=jira dbname=jira sslmode=disable"
      JWT_SECRET: "change-me-use-a-long-random-string"
      JWT_ACCESS_TTL: "15m"
      JWT_REFRESH_TTL: "168h"
      CORS_ORIGINS: "http://localhost:3000"
      LOG_LEVEL: info

  frontend:
    build: ./frontend
    restart: unless-stopped
    depends_on:
      - backend
    ports:
      - "3000:3000"
    environment:
      NUXT_PUBLIC_API_BASE_URL: "http://localhost:8080/api/v1"

volumes:
  pg_data:
```

### Running

```bash
cd code

# Build and start everything
docker compose up --build

# Stop
docker compose down

# Stop and remove database volume (full reset)
docker compose down -v
```

App is at **http://localhost:3000** · API at **http://localhost:8080**

### Development mode (hot reload)

Run only the database in Docker and start backend/frontend natively:

```bash
# Start just Postgres
docker compose up db

# Backend (separate terminal)
cd backend
DB_DRIVER=postgres \
DB_DSN="host=localhost port=5432 user=jira password=jira dbname=jira sslmode=disable" \
JWT_SECRET=dev-secret \
go run ./cmd/server

# Frontend (separate terminal)
cd frontend
npm run dev
```

---

## Option 2 — ddev

[ddev](https://ddev.readthedocs.io) supports custom project types and is well-suited for polyglot stacks.

### Install ddev

```bash
# macOS
brew install ddev/ddev/ddev

# Linux (or WSL2 on Windows)
curl -fsSL https://ddev.com/install.sh | bash
```

### Configuration files

**`.ddev/config.yaml`** (at `code/`)

```yaml
name: mansooba
type: generic
docroot: ""
php_version: "8.3"       # unused but required by ddev
webserver_type: generic
router_http_port: "3000"
router_https_port: "3443"
database:
  type: postgres
  version: "16"

hooks:
  post-start:
    - exec: echo "Backend and frontend start via docker-compose.backend.yaml and docker-compose.frontend.yaml"
```

**`.ddev/docker-compose.backend.yaml`**

```yaml
services:
  backend:
    image: golang:1.21-alpine
    working_dir: /app
    volumes:
      - ../backend:/app
    command: >
      sh -c "go run ./cmd/server"
    environment:
      SERVER_PORT: "8080"
      DB_DRIVER: postgres
      DB_DSN: "host=db port=5432 user=db password=db dbname=db sslmode=disable"
      JWT_SECRET: "ddev-dev-secret"
      CORS_ORIGINS: "http://mansooba.ddev.site:3000"
      LOG_LEVEL: debug
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

# View logs
ddev logs -s backend
ddev logs -s frontend

# Stop
ddev stop

# Destroy everything (including DB)
ddev delete --omit-snapshot
```

> ddev automatically manages a Postgres 16 container. The DSN host is `db` (ddev's internal service name).  
> The site is accessible at **http://mansooba.ddev.site:3000**

---

## Option 3 — Lando

[Lando](https://lando.dev) is a Docker-based dev tool with first-class support for custom multi-service stacks.

### Install Lando

```bash
# macOS
brew install lando

# Linux / Windows
# Download the installer from https://lando.dev/download
```

### Configuration file

**`.lando.yml`** (at `code/`)

```yaml
name: mansooba
recipe: lamp           # base recipe; services below override it entirely

services:

  db:
    type: postgres:16
    portforward: 5432
    creds:
      user: jira
      password: jira
      database: jira

  backend:
    type: go:1.21
    ssl: false
    command: go run ./cmd/server
    overrides:
      environment:
        SERVER_PORT: "8080"
        DB_DRIVER: postgres
        DB_DSN: "host=database port=5432 user=jira password=jira dbname=jira sslmode=disable"
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

tooling:
  go:
    service: backend
  npm:
    service: frontend
  psql:
    service: db
    cmd: psql -U jira jira
```

### Running with Lando

```bash
cd code

# Start all services
lando start

# Run Go commands inside the backend container
lando go test ./...

# Run npm commands inside the frontend container
lando npm run typecheck

# Open a Postgres shell
lando psql

# Stop
lando stop

# Destroy
lando destroy
```

App at **http://mansooba.lndo.site:3000** · API at **http://api.mansooba.lndo.site:8080**

---

## Troubleshooting

**Backend fails to connect to Postgres on startup**  
The backend starts before Postgres is ready. Docker Compose handles this with the `healthcheck` + `depends_on: condition: service_healthy` setup. With ddev/Lando, add a small startup delay or retry loop if needed.

**`jwt: JWT_SECRET must not be empty` panic**  
Set `JWT_SECRET` in the environment. It has no default and the app will panic without it.

**Port conflicts**  
If `:5432`, `:8080`, or `:3000` are already in use, stop the conflicting service or change the host-side port mapping (e.g. `"5433:5432"` for Postgres).

**Frontend can't reach the backend**  
Inside Docker Compose networks, services talk to each other by service name — not `localhost`. The `NUXT_PUBLIC_API_BASE_URL` in the frontend container should use the **backend service name** (`http://backend:8080/api/v1`) for server-side rendering, and `http://localhost:8080/api/v1` for browser-side calls. Since this app runs as a SPA (SSR disabled per route), `localhost:8080` works fine from the browser.
