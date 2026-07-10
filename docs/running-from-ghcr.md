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

## Option A — Quick local run (SQLite + LocalStack)

The fastest way to run the full stack locally. Uses SQLite for the database and a local LocalStack container for S3-compatible object storage (issue attachments) — no Postgres setup required.

Create a `compose.quickstart.yml` anywhere:

```yaml
services:
  localstack:
    image: localstack/localstack:3.8
    ports:
      - "4566:4566"
    environment:
      SERVICES: s3
      DEFAULT_REGION: us-east-1
      ACTIVATE_PRO: "0"
    volumes:
      - localstack_data:/var/lib/localstack
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:4566/_localstack/health"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 10s

  localstack-init:
    image: amazon/aws-cli:latest
    depends_on:
      localstack:
        condition: service_healthy
    environment:
      AWS_ACCESS_KEY_ID: test
      AWS_SECRET_ACCESS_KEY: test
      AWS_DEFAULT_REGION: us-east-1
    entrypoint: >
      /bin/sh -c "
      aws --endpoint-url http://localstack:4566 s3api head-bucket --bucket mansooba-attachments 2>/dev/null ||
      aws --endpoint-url http://localstack:4566 s3 mb s3://mansooba-attachments --region us-east-1 &&
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
      STORAGE_ENDPOINT: http://localstack:4566
      # PresignEndpoint overrides the host baked into presigned download URLs — the
      # backend reaches LocalStack via the Docker-internal hostname above, but the
      # browser that follows those URLs needs "localhost" instead.
      STORAGE_PRESIGN_ENDPOINT: http://localhost:4566
      STORAGE_BUCKET: mansooba-attachments
      STORAGE_ACCESS_KEY_ID: test
      STORAGE_SECRET_ACCESS_KEY: test
      STORAGE_USE_PATH_STYLE: "true"
    volumes:
      - sqlite_data:/data
    depends_on:
      localstack-init:
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
  localstack_data:
```

Then:

```bash
docker compose -f compose.quickstart.yml pull
docker compose -f compose.quickstart.yml up -d
```

App is at **http://localhost:3000** · API at **http://localhost:8080** · LocalStack health check at **http://localhost:4566/_localstack/health**

> SQLite data persists in the `sqlite_data` volume. Stop with `docker compose -f compose.quickstart.yml down` (add `-v` to wipe data too).

---

## Option B — Full stack with PostgreSQL (compose.prod.yml)

Uses `compose.prod.yml` from the repo with a local Postgres and LocalStack container alongside it. Closest to the real production setup.

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
STORAGE_ENDPOINT=http://localstack:4566
STORAGE_PRESIGN_ENDPOINT=http://localhost:4566
STORAGE_BUCKET=mansooba-attachments
STORAGE_ACCESS_KEY_ID=test
STORAGE_SECRET_ACCESS_KEY=test
STORAGE_REGION=us-east-1
STORAGE_USE_PATH_STYLE=true
STORAGE_PRESIGN_TTL=1h
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m
```

> The backend supports `DB_DRIVER=sqlite`, `postgres`, or `mysql` / `mariadb`. For local use, any of these work fine — swap the `DB_DRIVER` and `DB_DSN` values and add the matching database container to `compose.override.yml` if using MariaDB. For production (e.g. EC2 + RDS), set `DB_DSN` to the RDS endpoint with `sslmode=require` and replace the LocalStack vars with real AWS S3 access: unset `STORAGE_ENDPOINT`, `STORAGE_PRESIGN_ENDPOINT`, `STORAGE_ACCESS_KEY_ID`, and `STORAGE_SECRET_ACCESS_KEY` entirely, set `STORAGE_USE_PATH_STYLE=false`, and rely on the EC2 instance's IAM role for credentials (see ADR-029) — never a static key in production.

### Step 3 — Create a local override file

`compose.prod.yml` connects to an external database and storage. For local use, add Postgres and LocalStack via an override file.

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

  localstack:
    image: localstack/localstack:3.8
    restart: unless-stopped
    ports:
      - "4566:4566"
    environment:
      SERVICES: s3
      DEFAULT_REGION: us-east-1
      ACTIVATE_PRO: "0"
    volumes:
      - localstack_data:/var/lib/localstack
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:4566/_localstack/health"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 10s

  localstack-init:
    image: amazon/aws-cli:latest
    depends_on:
      localstack:
        condition: service_healthy
    environment:
      AWS_ACCESS_KEY_ID: test
      AWS_SECRET_ACCESS_KEY: test
      AWS_DEFAULT_REGION: us-east-1
    entrypoint: >
      /bin/sh -c "
      aws --endpoint-url http://localstack:4566 s3api head-bucket --bucket mansooba-attachments 2>/dev/null ||
      aws --endpoint-url http://localstack:4566 s3 mb s3://mansooba-attachments --region us-east-1 &&
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
      localstack-init:
        condition: service_completed_successfully

volumes:
  pg_data:
  localstack_data:
```

### Step 4 — Pull and start

```bash
# Pull the latest images from GHCR
docker compose -f compose.prod.yml -f compose.override.yml pull

# Start all services
docker compose -f compose.prod.yml -f compose.override.yml up -d
```

App is at **http://localhost** (port 80) · API at **http://localhost:8080** · LocalStack health check at **http://localhost:4566/_localstack/health** · Mail inbox at **http://localhost:8025**

### Step 5 — Verify

```bash
# Check all containers are running
docker compose -f compose.prod.yml -f compose.override.yml ps

# Check backend health (expect all green)
curl http://localhost:8080/health
# Expected: {"status":"ok","db":"ok","db_latency_ms":1}
# Note: this only reports database connectivity, not storage — the backend doesn't
# check storage at startup, only when an attachment is actually uploaded/downloaded/deleted.
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
| `JWT_SECRET` | ✅ | — | Secret for signing JWTs. Use `openssl rand -hex 32`. The only var the backend requires at startup. |
| `DB_DRIVER` | ✅ | — | `sqlite`, `postgres` / `postgresql`, or `mysql` / `mariadb` |
| `DB_DSN` | ✅ | — | SQLite path, Postgres DSN, or MySQL/MariaDB DSN |
| `CORS_ORIGINS` | ✅ | — | Comma-separated allowed origins (e.g. `http://localhost`) |
| `LOG_LEVEL` | | `info` | `debug`, `info`, `warn`, `error` |
| `SERVER_PORT` | | `8080` | Port the backend listens on |
| `BODY_SIZE_LIMIT` | | `4M` | Max request body size (rejects with 413 when exceeded) |
| `REQUEST_TIMEOUT` | | `30s` | Per-request timeout |
| `SHUTDOWN_TIMEOUT` | | `30s` | Graceful shutdown window |
| `STORAGE_ENDPOINT` | | *(unset — AWS default)* | Leave unset for real AWS S3; set to `http://localstack:4566` for LocalStack. Only checked when an attachment is actually uploaded/downloaded/deleted, not at startup. |
| `STORAGE_PRESIGN_ENDPOINT` | | *(unset, falls back to `STORAGE_ENDPOINT`)* | Overrides the host baked into presigned download URLs — needed when `STORAGE_ENDPOINT` is a Docker-internal hostname the browser can't resolve |
| `STORAGE_BUCKET` | | `mansooba-attachments` | S3/LocalStack bucket name |
| `STORAGE_ACCESS_KEY_ID` / `STORAGE_SECRET_ACCESS_KEY` | | *(unset)* | LocalStack only (`test`/`test`); leave unset in production — the EC2 instance's IAM role is used instead (ADR-029) |
| `STORAGE_REGION` | | `us-east-1` | AWS region (LocalStack ignores this) |
| `STORAGE_PRESIGN_TTL` | | `1h` | Pre-signed download URL expiry |
| `STORAGE_USE_PATH_STYLE` | | `false` | Set `true` for LocalStack; `false` for AWS S3 |
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

**Attachment upload/download/delete returns 502 "storage temporarily unavailable"**  
The backend can't reach the object storage endpoint (checked lazily, only on these requests — not at startup). Verify:
- The `localstack` container is running and healthy: `docker compose ps localstack`
- `STORAGE_ENDPOINT` matches a hostname the *backend container* can resolve (use the service name `localstack` inside Compose networks, not `localhost`) — and separately, `STORAGE_PRESIGN_ENDPOINT` matches a hostname your *browser* can resolve (`localhost`, not `localstack`)
- The bucket was created by `localstack-init`: `docker compose logs localstack-init`

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
