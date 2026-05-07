# jira-go

A mini Jira clone built as a learning and portfolio project.

**Backend:** Go + Echo v4 · GORM · SQLite (local) / PostgreSQL (prod) · JWT auth  
**Frontend:** Nuxt 4 (SPA) · Pinia · Tailwind CSS v4 · DaisyUI

## Prerequisites

- Go 1.21+
- Node 22 + npm 10
- SQLite development headers (`sqlite-devel` or `libsqlite3-dev`)

## Running locally

### 1. Environment

```sh
cp code/backend/.env.example code/backend/.env
# Edit code/backend/.env if needed — defaults work out of the box
```

### 2. Backend

```sh
cd code/backend
go run ./cmd/server
# Listening on http://localhost:8080
# Health check: GET http://localhost:8080/health
```

### 3. Frontend

```sh
cd code/frontend
npm install
npm run dev
# Listening on http://localhost:3000
```

## Project structure

```
code/
  backend/          # Go API
    cmd/server/     # Entry point
    pkg/config/     # Env config (Viper)
    pkg/database/   # GORM connection
    pkg/logger/     # Zap logger
  frontend/         # Nuxt 4 SPA
    app/            # Source root (pages, components, stores, ...)
docs/
  decisions/        # Architecture Decision Records
  plan/             # MVP task plans
infrastructure/     # Terraform (MVP 5)
```

## Architecture

Clean Architecture layers (backend): `domain` → `repository` → `service` → `handler`  
`domain/` has zero external imports; GORM is confined to `repository/`.
