# go-jira

A project management application (simialr to Jira) build in Go and Nuxt.js as a learning and portfolio project.

It implemented using spec driven approach, where I worked as architect/manager and AI/Claude worked as enginneer.

**Backend:** Go + Echo v4 · GORM · SQLite (local) / PostgreSQL (prod) · JWT auth  
**Frontend:** Nuxt 4 (SPA) · Pinia · Tailwind CSS v4 · DaisyUI

> Design spec, ADRs, and task plans live in the docs repo: [sharique/jira-go](https://github.com/sharique/jira-go)

## Prerequisites

- Go 1.21+
- Node 22 + npm 10
- SQLite development headers (`sqlite-devel` or `libsqlite3-dev`)

## Running locally

### 1. Environment

```sh
cp backend/.env.example backend/.env
# Edit backend/.env if needed — defaults work out of the box
```

### 2. Backend

```sh
cd backend
go run ./cmd/server
# Listening on http://localhost:8080
# Health check: GET http://localhost:8080/health
```

### 3. Frontend

```sh
cd frontend
npm install
npm run dev
# Listening on http://localhost:3000
```

## Project structure

```
backend/              # Go API
  cmd/server/         # Entry point
  internal/domain/    # Domain entities and repository interfaces
  internal/repository/# GORM repository implementations
  pkg/config/         # Env config (Viper)
  pkg/database/       # GORM connection + migrations
  pkg/logger/         # Zap logger
frontend/             # Nuxt 4 SPA
  app/                # Source root (pages, components, stores, ...)
```

## Architecture

Clean Architecture layers (backend): `domain` → `repository` → `service` → `handler`  
`domain/` has zero external imports; GORM is confined to `repository/`.
