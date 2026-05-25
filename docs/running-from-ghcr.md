# Running Mansooba from GHCR Images

Pull and run the pre-built backend and frontend containers from GitHub Container Registry — no Go toolchain or Node.js required.

**Images:**
- `ghcr.io/sharique/mansooba-backend:latest`
- `ghcr.io/sharique/mansooba-frontend:latest`

Images are built and pushed automatically on every merge to `main` via GitHub Actions.

---

## Prerequisites

- [Docker Desktop](https://docs.docker.com/get-docker/) (or Docker Engine + Compose plugin on Linux)

Verify:
```bash
docker --version          # Docker 24.x or later
docker compose version    # Docker Compose v2.x
```

---

## Option A — Quick local run (SQLite, no database setup)

The backend supports SQLite out of the box. This is the fastest way to get the app running locally.

```bash
docker run --rm \
  -e JWT_SECRET=dev-secret-change-me \
  -e DB_DRIVER=sqlite \
  -e DB_DSN=/data/dev.db \
  -e CORS_ORIGINS=http://localhost:3000 \
  -p 8080:8080 \
  -v mansooba-data:/data \
  ghcr.io/sharique/mansooba-backend:latest
```

In a second terminal, start the frontend:
```bash
docker run --rm \
  -e NUXT_PUBLIC_API_BASE_URL=http://localhost:8080/api/v1 \
  -p 3000:3000 \
  ghcr.io/sharique/mansooba-frontend:latest
```

App is at **http://localhost:3000** · API at **http://localhost:8080**

> Stop both containers with `Ctrl+C`. The SQLite database persists in the `mansooba-data` Docker volume between runs.

---

## Option B — Full stack with PostgreSQL (compose.prod.yml)

Uses the production compose file with a local Postgres container. Closest to the real production setup.

### Step 1 — Authenticate to GHCR

**If the packages are public** (check at `github.com/sharique` → Packages), skip this step.

**If the packages are private**, log in with a GitHub Personal Access Token:

1. Create a PAT at [github.com/settings/tokens](https://github.com/settings/tokens) → **Tokens (classic)**
2. Scopes: tick only **`read:packages`**
3. Log in:
   ```bash
   echo "ghp_your_token_here" | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin
   ```

### Step 2 — Create a `.env` file

Create `.env` in the repo root (next to `compose.prod.yml`). This file is gitignored — never commit real values.

```bash
# .env
DB_DRIVER=postgres
DB_DSN=host=db port=5432 user=mansooba password=mansooba dbname=mansooba sslmode=disable
JWT_SECRET=change-me-use-a-long-random-string
LOG_LEVEL=info
CORS_ORIGINS=http://localhost
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m
```

> For production (EC2 deployment), `DB_DSN` points at the RDS endpoint and uses `sslmode=require`. Locally, `sslmode=disable` is fine.

### Step 3 — Add a local Postgres service

`compose.prod.yml` connects to an external database. For local use, add Postgres alongside it with an override file:

Create `compose.override.yml` in the repo root:

```yaml
services:
  db:
    image: postgres:16-alpine
    restart: unless-stopped
    environment:
      POSTGRES_USER: mansooba
      POSTGRES_PASSWORD: mansooba
      POSTGRES_DB: mansooba
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

volumes:
  pg_data:
```

### Step 4 — Pull and start

```bash
# Pull the latest images from GHCR
docker compose -f compose.prod.yml -f compose.override.yml pull

# Start all services (Postgres + backend + frontend)
docker compose -f compose.prod.yml -f compose.override.yml up -d
```

App is at **http://localhost** (port 80) · API at **http://localhost:8080**

### Step 5 — Verify

```bash
# Check all three containers are running
docker compose -f compose.prod.yml -f compose.override.yml ps

# Check backend health
curl http://localhost:8080/health
# Expected: {"status":"ok","db":"ok","db_latency_ms":1}
```

### Stopping and cleanup

```bash
# Stop containers (data is preserved in volumes)
docker compose -f compose.prod.yml -f compose.override.yml down

# Stop and remove all data volumes (full reset)
docker compose -f compose.prod.yml -f compose.override.yml down -v
```

---

## Option C — Pin to a specific image version

Every merge to `main` also tags images with a short SHA (`sha-abc1234`). Use a pinned tag for reproducible deployments:

```bash
# List available tags at:
# https://github.com/sharique/mansooba/pkgs/container/mansooba-backend

# Pull a specific version
docker pull ghcr.io/sharique/mansooba-backend:sha-abc1234
docker pull ghcr.io/sharique/mansooba-frontend:sha-abc1234
```

To use a pinned tag in the compose file, set `BACKEND_TAG` and `FRONTEND_TAG` as environment variables before `docker compose up`:

```bash
BACKEND_TAG=sha-abc1234 FRONTEND_TAG=sha-abc1234 \
  docker compose -f compose.prod.yml -f compose.override.yml up -d
```

Then update `compose.prod.yml` image references to use the variable:
```yaml
# compose.prod.yml
services:
  backend:
    image: ghcr.io/sharique/mansooba-backend:${BACKEND_TAG:-latest}
  frontend:
    image: ghcr.io/sharique/mansooba-frontend:${FRONTEND_TAG:-latest}
```

---

## Environment Variables Reference

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `JWT_SECRET` | ✅ | — | Secret for signing JWTs. Use `openssl rand -hex 32`. |
| `DB_DRIVER` | ✅ | — | `sqlite` or `postgres` |
| `DB_DSN` | ✅ | — | SQLite path or Postgres connection string |
| `CORS_ORIGINS` | ✅ | — | Comma-separated allowed origins (e.g. `http://localhost`) |
| `LOG_LEVEL` | | `info` | `debug`, `info`, `warn`, `error` |
| `DB_MAX_OPEN_CONNS` | | `25` | Max open DB connections |
| `DB_MAX_IDLE_CONNS` | | `5` | Max idle DB connections |
| `DB_CONN_MAX_LIFETIME` | | `5m` | Connection max lifetime |
| `SERVER_PORT` | | `8080` | Port the backend listens on |

**Postgres DSN format:**
```
host=<hostname> port=5432 user=<user> password=<pass> dbname=<db> sslmode=disable
```

---

## Troubleshooting

**`docker pull` returns 401 Unauthorized**  
The GHCR packages are private. Log in with a `read:packages` PAT (see Step 1 in Option B).

**Backend container exits immediately**  
`JWT_SECRET` is missing or empty. The backend panics without it. Check:
```bash
docker logs mansooba-backend-1
```

**Backend returns `"db":"error"` in /health**  
Database connection failed. For Option B, check that `db` container is healthy before `backend` starts:
```bash
docker compose -f compose.prod.yml -f compose.override.yml ps
# db should show "healthy"
```

**Frontend shows API errors / blank page**  
`CORS_ORIGINS` doesn't match the origin you're accessing the app from. If accessing via `http://localhost`, set `CORS_ORIGINS=http://localhost`. If via `http://localhost:3000`, set accordingly.

**Port 80 already in use**  
Another service is using port 80. Either stop it, or change the frontend port mapping in `compose.override.yml`:
```yaml
services:
  frontend:
    ports:
      - "8081:80"   # access app at http://localhost:8081
```
