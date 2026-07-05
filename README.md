# Mansooba

*Mansooba (منصوبہ) — Urdu for "plan" or "project"*

A project management app for teams — track issues, run sprints, and collaborate, all in
one place. Built as a learning and portfolio project using a spec-driven approach.

**Backend:** Go + Echo v4 · GORM · SQLite / PostgreSQL  
**Frontend:** Nuxt 4 · Pinia · Tailwind CSS v4 · DaisyUI

### Demo

[![Initial Setup Walkthrough](https://cdn.loom.com/sessions/thumbnails/61ce0f47d5f14ecbb51c7a314fa0a107-93b2fe3c8eaa3262-full-play.gif)](https://www.loom.com/share/61ce0f47d5f14ecbb51c7a314fa0a107)

---

## Features

| Feature | What you can do |
|---|---|
| **Authentication** | Sign in, reset your password by email, manage your profile and avatar |
| **Projects** | Create projects, invite teammates, and control who can manage them |
| **Issues** | Track tasks, bugs, stories, and epics with labels, priorities, and related-issue links |
| **Sprints** | Plan and run time-boxed sprints with a kanban board, burndown chart, and velocity chart |
| **Backlog** | Triage unscheduled work and pull it into upcoming sprints |
| **Collaboration** | Comment on issues, @mention teammates, and follow per-issue activity feeds |
| **My Desk** | See your assigned issues, notifications, and pinned projects at a glance |
| **Reports** | Visualise sprint velocity across your project history |
| **System Admin** | Manage users, configure platform settings, and onboard new team members |
| **First-Run Wizard** | Six-step guided setup gets a fresh install ready in minutes |

---

## Getting Started

```sh
docker compose up --build
```

| What | URL |
|---|---|
| App | http://localhost:3000 |
| API | http://localhost:8080 |
| Email (dev) | http://localhost:8025 |

On first visit the setup wizard walks you through creating the admin account.

Other ways to run:
- [Docker with PostgreSQL or hot-reload dev mode](docs/running-locally-using-docker.md)
- [Pre-built GHCR images](docs/running-from-ghcr.md) — no Go toolchain needed
- [From source](docs/running-from-source.md) — Go + Node, no Docker

---

## Documentation

### Architecture

| | |
|---|---|
| [Overview](docs/arch-overview.md) | Tech stack, project structure, CI/CD pipeline |
| [Backend](docs/arch-backend.md) | Layers, entities, services |
| [Frontend](docs/arch-frontend.md) | Pages, stores, components, routing |
| [API reference](docs/arch-api.md) | All endpoints with methods and descriptions |

### Feature deep-dives

| | |
|---|---|
| [Authentication & security](docs/feature-auth.md) | JWT strategy, token revocation, password reset flow |
| [Issues](docs/feature-issues.md) | Relations, labels, and how issues connect |
| [Sprints & board](docs/feature-sprints.md) | Lifecycle states, completion migration, burndown & velocity |
| [Reports](docs/feature-reports.md) | Velocity chart data source and rendering approach |
| [Collaboration](docs/feature-collaboration.md) | @mention parsing, notification model, activity feeds |
| [System admin](docs/feature-admin.md) | User management, platform settings, and safety guardrails |
| [Setup wizard](docs/feature-setup.md) | First-run wizard flow, sample data import, and seed CLI |
| [First-run wizard guide](docs/first-run-wizard.md) | Step-by-step wizard reference with DEMO conflict, retry behaviour, and CLI output |
