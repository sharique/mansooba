# Architecture Overview

## Tech stack

| Layer | Technology |
|-------|-----------|
| Backend language | Go 1.25 |
| Backend framework | Echo v4 |
| ORM | GORM |
| Databases | SQLite (dev default), PostgreSQL, MySQL / MariaDB |
| Object storage | AWS S3 / MinIO (aws-sdk-go-v2) |
| Auth | JWT (golang-jwt/jwt) |
| Logging | Zap |
| Config | Viper (env vars + `.env` file) |
| Frontend framework | Nuxt 4 + Vue 3 |
| State management | Pinia |
| Testing (backend) | Go stdlib `testing` + `testify` |
| Testing (frontend) | Vitest 4 |

---

## Project structure

```
code/
├── backend/
│   ├── cmd/server/            # Entry point — route wiring and server startup
│   ├── internal/
│   │   ├── domain/            # Entities + repository interfaces (zero external imports)
│   │   ├── dto/               # Request/response structs shared between handlers and services
│   │   ├── repository/        # GORM implementations of all repository interfaces
│   │   ├── service/           # Business logic; works only with domain types and DTOs
│   │   ├── handler/           # Echo HTTP handlers; HTTP ↔ service translation only
│   │   ├── middleware/        # JWT auth middleware
│   │   └── pkg/
│   │       └── avatarstorage/ # Local-disk storage for user avatar images
│   └── pkg/
│       ├── apierror/          # Centralised HTTP error mapping
│       ├── config/            # Env config loader (Viper)
│       ├── database/          # GORM connection + auto-migrations
│       ├── logger/            # Zap logger initialisation
│       └── storage/           # S3/MinIO client + FakeStorage for tests
│
├── frontend/
│   └── app/
│       ├── assets/css/        # OKLCH design tokens, global styles
│       ├── components/        # Vue SFCs grouped by domain
│       ├── composables/       # Shared composition functions
│       ├── layouts/           # default.vue — Sidebar + TopBar shell
│       ├── middleware/        # Route guards (auth, setup redirect)
│       ├── pages/             # Nuxt file-based routes
│       ├── plugins/           # $fetch proxy, auth init, theme init
│       ├── services/          # API layer — one file per backend resource
│       ├── stores/            # Pinia stores — one file per domain
│       ├── types/             # TypeScript domain, API, auth, setup types
│       └── utils/             # chart helpers, issue style maps
│
├── compose.yml                # Dev stack: SQLite + MinIO + backend + frontend
├── compose.prod.yml           # Prod stack: pulls GHCR images, connects to external DB
├── terraform/                 # AWS infra (EC2 + RDS)
└── docs/                      # Architecture and operational guides
```

---

## How the layers connect

```
HTTP request
     │
     ▼  handler/       parse request → call service → map errors → HTTP response
     │
     ▼  service/       business logic; orchestrates repositories; enforces access rules
     │
     ▼  repository/    GORM queries; wraps DB errors into domain errors
     │
     ▼  domain/        pure Go structs (entities) + repository interfaces
```

The frontend mirrors this with its own layered stack:

```
Browser
   │
   ▼  nginx           serves static Nuxt bundle; proxies /api/* to backend
   │
   ▼  pages/          Composition API; trigger store actions
   │
   ▼  stores/         Pinia; hold fetched state; call services for mutations
   │
   ▼  services/       thin $fetch wrappers — one file per backend resource
   │
   ▼  backend API
```

---

## Storage architecture

The app uses two storage subsystems:

| What | Where | Access pattern |
|------|-------|---------------|
| Attachments (files, images, docs) | S3 / MinIO | Pre-signed URLs (1 h TTL); never proxied through the backend |
| User avatars | Local disk (`uploads/`) | Publicly served at `/uploads/*` without auth (ADR-026) |

Attachment keys follow the pattern:
```
projects/<projectID>/tasks/<issueID>/attachments/<attachmentID>-<sanitised-filename>
```

---

## CI/CD pipeline

GitHub Actions (`ci.yml`) runs three jobs on every push to `main`, `develop`, and `feature/**`:

| Job | Trigger | What |
|-----|---------|------|
| `test` | every push / PR | `go vet ./...` + `go test -race -count=1 ./...` (Go 1.25) |
| `frontend` | every push / PR | `npm run typecheck` + `npm test` (Node 22) |
| `build-and-push` | merge to `main` only | Docker Buildx → GHCR; tags `sha-<short>` + `latest` |

Images published:
- `ghcr.io/sharique/mansooba-backend`
- `ghcr.io/sharique/mansooba-frontend`

---

## Further reading

- [Backend detail](arch-backend.md) — layers, entities, services
- [Frontend detail](arch-frontend.md) — pages, stores, components, routing
- [API reference](arch-api.md) — all endpoints with methods and descriptions
- [Running locally with Docker](running-locally-using-docker.md)
- [Running from GHCR images](running-from-ghcr.md)
