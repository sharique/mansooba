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

## Option A — Quick local run (SQLite + MinIO)

The fastest way to run the full stack locally. Uses SQLite for the database and a local MinIO container for object storage — no Postgres setup required.

Create a `compose.quickstart.yml` anywhere:

```yaml
services:
  minio:
    image: minio/minio:latest
    ports:
      - "9000:9000"
      - "9001:9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    command: server /data --console-address ":9001"
    volumes:
      - minio_data:/data
    healthcheck:
      test: ["CMD", "mc", "ready", "local"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 10s

  minio-init:
    image: minio/mc:latest
    depends_on:
      minio:
        condition: service_healthy
    entrypoint: >
      /bin/sh -c "
      mc alias set local http://minio:9000 minioadmin minioadmin &&
      mc mb --ignore-existing local/mansooba &&
      echo 'bucket ready'
      "

  backend:
    image: ghcr.io/sharique/mansooba-backend:latest
    ports:
      - "8080:8080"
    environment:
      JWT_SECRET: dev-secret-change-me
      DB_DRIVER: sqlite
      DB_DSN: /data/dev.db
      CORS_ORIGINS: http://localhost:3000
      STORAGE_ENDPOINT: http://minio:9000
      STORAGE_BUCKET: mansooba
      STORAGE_ACCESS_KEY_ID: minioadmin
      STORAGE_SECRET_ACCESS_KEY: minioadmin
      STORAGE_USE_PATH_STYLE: "true"
    volumes:
      - sqlite_data:/data
    depends_on:
      minio-init:
        condition: service_completed_successfully
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8080/health"]
      interval: 15s
      timeout: 5s
      retries: 3
      start_period: 5s

  frontend:
    image: ghcr.io/sharique/mansooba-frontend:latest
    ports:
      - "3000:80"
    depends_on:
      backend:
        condition: service_healthy

volumes:
  sqlite_data:
  minio_data:
```

Then:

```bash
docker compose -f compose.quickstart.yml pull
docker compose -f compose.quickstart.yml up -d
```

App is at **http://localhost:3000** · API at **http://localhost:8080** · MinIO console at **http://localhost:9001**

> SQLite data persists in the `sqlite_data` volume. Stop with `docker compose -f compose.quickstart.yml down` (add `-v` to wipe data too).

---

## Option B — Full stack with PostgreSQL (compose.prod.yml)

Uses `compose.prod.yml` from the repo with a local Postgres and MinIO container alongside it. Closest to the real production setup.

### Step 1 — Authenticate to GHCR (if images are private)

**If the packages are public** (check at `github.com/sharique` → Packages), skip this step.

**If the packages are private**, log in with a GitHub Personal Access Token:

1. Create a PAT at [github.com/settings/tokens](https://github.com/settings/tokens) → **Tokens (classic)**
2. Scopes: tick only **`read:packages`**
3. Log in:
   ```bash
   echo "ghp_your_token_here" | docker login ghcr.io -u YOUR_GITHUB_USERNAME --password-stdin
   ```

### Step 2 — Create a `.env` file

Create `.env` in the same directory as `compose.prod.yml`. This file is gitignored — never commit real values.

```bash
# .env
DB_DRIVER=postgres
DB_DSN=host=db port=5432 user=mansooba password=mansooba dbname=mansooba sslmode=disable
JWT_SECRET=change-me-use-a-long-random-string
LOG_LEVEL=info
CORS_ORIGINS=http://localhost
STORAGE_ENDPOINT=http://minio:9000
STORAGE_BUCKET=mansooba
STORAGE_ACCESS_KEY_ID=minioadmin
STORAGE_SECRET_ACCESS_KEY=minioadmin
STORAGE_REGION=us-east-1
STORAGE_USE_PATH_STYLE=true
STORAGE_PRESIGN_TTL=1h
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m
```

> The backend supports `DB_DRIVER=sqlite`, `postgres`, or `mysql` / `mariadb`. For local use, any of these work fine — swap the `DB_DRIVER` and `DB_DSN` values and add the matching database container to `compose.override.yml` if using MariaDB. For production (e.g. EC2 + RDS), set `DB_DSN` to the RDS endpoint with `sslmode=require` and replace the MinIO vars with real AWS S3 credentials (`STORAGE_ENDPOINT` empty, `STORAGE_USE_PATH_STYLE=false`).

### Step 3 — Create a local override file

`compose.prod.yml` connects to an external database and storage. For local use, add Postgres and MinIO via an override file.

Create `compose.override.yml` in the same directory:

```yaml
services:
  db:
    image: postgres:17-alpine
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

  minio:
    image: minio/minio:latest
    restart: unless-stopped
    ports:
      - "9000:9000"
      - "9001:9001"
    environment:
      MINIO_ROOT_USER: minioadmin
      MINIO_ROOT_PASSWORD: minioadmin
    command: server /data --console-address ":9001"
    volumes:
      - minio_data:/data
    healthcheck:
      test: ["CMD", "mc", "ready", "local"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 10s

  minio-init:
    image: minio/mc:latest
    depends_on:
      minio:
        condition: service_healthy
    entrypoint: >
      /bin/sh -c "
      mc alias set local http://minio:9000 minioadmin minioadmin &&
      mc mb --ignore-existing local/mansooba &&
      echo 'bucket ready'
      "

  mailpit:
    image: axllent/mailpit:latest
    ports:
      - "1025:1025"
      - "8025:8025"
    restart: unless-stopped

  backend:
    depends_on:
      db:
        condition: service_healthy
      minio-init:
        condition: service_completed_successfully

volumes:
  pg_data:
  minio_data:
```

### Step 4 — Pull and start

```bash
# Pull the latest images from GHCR
docker compose -f compose.prod.yml -f compose.override.yml pull

# Start all services
docker compose -f compose.prod.yml -f compose.override.yml up -d
```

App is at **http://localhost** (port 80) · API at **http://localhost:8080** · MinIO console at **http://localhost:9001** · Mail inbox at **http://localhost:8025**

### Step 5 — Verify

```bash
# Check all containers are running
docker compose -f compose.prod.yml -f compose.override.yml ps

# Check backend health (expect all green)
curl http://localhost:8080/health
# Expected: {"status":"ok","db":"ok","db_latency_ms":1,"storage":"ok"}
```

### Stopping and cleanup

```bash
# Stop containers (data preserved in volumes)
docker compose -f compose.prod.yml -f compose.override.yml down

# Stop and remove all data volumes (full reset)
docker compose -f compose.prod.yml -f compose.override.yml down -v
```

---

## Option C — Pin to a specific image version

Every merge to `main` also tags images with a short SHA (`sha-abc1234`). Use a pinned tag for reproducible deployments:

```bash
# Browse available tags:
# https://github.com/sharique/mansooba/pkgs/container/mansooba-backend

# Pull a specific version
docker pull ghcr.io/sharique/mansooba-backend:sha-abc1234
docker pull ghcr.io/sharique/mansooba-frontend:sha-abc1234
```

To use a pinned tag in the compose file, set env vars before `docker compose up`:

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
| `DB_DRIVER` | ✅ | — | `sqlite`, `postgres` / `postgresql`, or `mysql` / `mariadb` |
| `DB_DSN` | ✅ | — | SQLite path, Postgres DSN, or MySQL/MariaDB DSN |
| `CORS_ORIGINS` | ✅ | — | Comma-separated allowed origins (e.g. `http://localhost`) |
| `STORAGE_BUCKET` | ✅ | — | S3/MinIO bucket name |
| `STORAGE_ACCESS_KEY_ID` | ✅ | — | S3/MinIO access key |
| `STORAGE_SECRET_ACCESS_KEY` | ✅ | — | S3/MinIO secret key |
| `LOG_LEVEL` | | `info` | `debug`, `info`, `warn`, `error` |
| `SERVER_PORT` | | `8080` | Port the backend listens on |
| `BODY_SIZE_LIMIT` | | `4M` | Max request body size (rejects with 413 when exceeded) |
| `REQUEST_TIMEOUT` | | `30s` | Per-request timeout |
| `SHUTDOWN_TIMEOUT` | | `30s` | Graceful shutdown window |
| `STORAGE_ENDPOINT` | | *(AWS default)* | Leave empty for AWS S3; set to MinIO URL for self-hosted |
| `STORAGE_REGION` | | `us-east-1` | AWS region (MinIO ignores this) |
| `STORAGE_PRESIGN_TTL` | | `1h` | Pre-signed download URL expiry |
| `STORAGE_USE_PATH_STYLE` | | `true` | Set `true` for MinIO; `false` for AWS S3 |
| `DB_MAX_OPEN_CONNS` | | `25` | Max open DB connections |
| `DB_MAX_IDLE_CONNS` | | `5` | Max idle DB connections |
| `DB_CONN_MAX_LIFETIME` | | `5m` | Connection max lifetime |
| `SMTP_HOST` | | `mailpit` | SMTP server host — use `mailpit` in the local override |
| `SMTP_PORT` | | `1025` | SMTP port |
| `SMTP_FROM` | | `noreply@mansooba.local` | Sender address for outbound email |

**Connection string formats:**
```
# PostgreSQL
host=<hostname> port=5432 user=<user> password=<pass> dbname=<db> sslmode=disable

# MySQL / MariaDB
<user>:<pass>@tcp(<host>:3306)/<db>?charset=utf8mb4&parseTime=True&loc=Local
```

---

## Troubleshooting

**`docker pull` returns 401 Unauthorized**  
The GHCR packages are private. Log in with a `read:packages` PAT (see Step 1 in Option B).

**Backend container exits immediately**  
`JWT_SECRET` is missing or empty. The backend refuses to start without it:
```bash
docker logs <container-name>
```

**Health endpoint shows `"storage":"error"`**  
The backend can't reach the object storage endpoint. Verify:
- The `minio` container is running and healthy: `docker compose ps minio`
- `STORAGE_ENDPOINT` matches the hostname Docker can resolve (use the service name `minio` inside Compose networks, not `localhost`)
- The bucket was created by `minio-init`: `docker compose logs minio-init`

**Health endpoint shows `"db":"error"`**  
Database connection failed. Check that the `db` container is healthy before the backend starts:
```bash
docker compose ps db
docker compose logs db
```

**Frontend shows API errors / blank page**  
`CORS_ORIGINS` must exactly match the origin you're accessing from (scheme + host + port). If accessing via `http://localhost`, set `CORS_ORIGINS=http://localhost`. If via `http://localhost:3000`, set accordingly.

**Port 80 already in use**  
Change the frontend host port in `compose.override.yml`:
```yaml
services:
  frontend:
    ports:
      - "8081:80"   # access app at http://localhost:8081
```
