# Backend Architecture

See [arch-overview.md](arch-overview.md) for the project-level overview and layer diagram.

## Layer responsibilities

### `domain/`

Pure Go structs with zero external imports. Every other layer depends on this one; it depends on nothing.

- Defines all **entities** (Go structs that map to database tables)
- Defines all **repository interfaces** (e.g. `IssueRepository`, `AttachmentRepository`)
- Defines shared **error sentinels** (`ErrNotFound`, `ErrForbidden`, `ErrConflict`, etc.)

### `dto/`

Request/response structs that flow between `handler/` and `service/`. Keeps GORM model fields out of the HTTP layer and prevents over-posting.

One file per domain area: `issue_dto.go`, `attachment_dto.go`, `relation_dto.go`, etc.

### `repository/`

GORM implementations of every repository interface declared in `domain/`. Contains no business logic — only query construction and error wrapping.

All repository tests (`*_repo_test.go`) run against a real in-memory SQLite database to catch query-level issues that mocks would hide.

### `service/`

All business logic lives here. Services:
- Accept and return domain types or DTOs (never GORM models)
- Enforce access control (membership checks, ownership guards, cross-project IDOR prevention)
- Orchestrate multiple repositories when needed (e.g. `IssueService.Delete` cascades to `attachmentRepo.DeleteByIssueID` then `relationRepo.DeleteByTaskID`)

### `handler/`

Echo HTTP handlers that do three things only:
1. Parse and validate the request (binding + `echo.Validate`)
2. Call the appropriate service method
3. Map the result or error to an HTTP response via `apierror.HTTPErrorHandler`

### `middleware/`

- `JWTAuth` validates the `Authorization: Bearer` header and injects the caller's user ID into the Echo context. All protected route groups use it.
- `DBActivity` / `DBWake` (spec 010, ADR-030) — only registered when the database idle auto-stop feature is active (AWS RDS demo deployment only; see [Database idle auto-stop](#database-idle-auto-stop) below). `DBActivity` records every request as activity and tracks it as an in-flight operation; `DBWake` intercepts requests while the database is stopped/starting and returns the `waking_up` signal instead of letting them reach a handler.

### `internal/pkg/avatarstorage/`

Local-disk storage for avatar images. Avatars are saved under `uploads/` and served as unauthenticated static files (ADR-026). Separate from `pkg/storage/` because avatars do not need pre-signed URLs or S3.

### `internal/pkg/rdsclient/`

Thin `aws-sdk-go-v2/service/rds` wrapper (spec 010) — `StartDBInstance`/`StopDBInstance`/`DescribeDBInstances`, authenticated via the EC2 instance's IAM role. Same shape as `pkg/storage`'s AWS SDK v2 usage below. Only ever constructed when the auto-stop feature is active; never touched in local dev.

### `pkg/storage/`

S3-compatible storage backed by AWS SDK v2. Exposes a `Storage` interface:

```go
type Storage interface {
    Upload(ctx, key, reader, size, contentType) error
    PreSignedURL(ctx, key, ttl) (string, error)
    Delete(ctx, key) error
    DeleteByPrefix(ctx, prefix) error
    Ping(ctx) error
}
```

`FakeStorage` (in-memory) is used in all unit tests. `s3Storage` is the production implementation. `Ping` calls `HeadBucket` and is used by the health endpoint to report storage reachability.

---

## Domain entities

| Entity | Key fields | Notes |
|--------|-----------|-------|
| `User` | `Name`, `Email`, `PasswordHash`, `AvatarURL`, `IsSuperAdmin` | |
| `Project` | `Key`, `Name`, `Description` | Key is used as the URL slug |
| `ProjectMember` | `UserID`, `ProjectID`, `Role` | Role: `member` or `admin` |
| `Issue` | `Key`, `Title`, `Description`, `Type`, `Status`, `Priority`, `AssigneeID`, `ReporterID`, `SprintID`, `StoryPoints`, `CompletedAt` | |
| `Sprint` | `Name`, `ProjectID`, `Status`, `StartDate`, `EndDate` | Status: `planned`, `active`, `completed` |
| `Comment` | `IssueID`, `AuthorID`, `Body` | |
| `Activity` | `IssueID`, `ActorID`, `Kind`, `OldValue`, `NewValue` | Audit log for issue field changes |
| `Label` | `ProjectID`, `Name`, `Color` | |
| `Notification` | `UserID`, `IssueID`, `Kind`, `ReadAt` | |
| `Attachment` | `IssueID`, `UploaderID`, `Filename`, `ObjectKey`, `ContentType`, `SizeBytes` | Binary in S3 (LocalStack in dev) |
| `TaskRelation` | `TaskAID`, `TaskBID`, `CreatedBy` | Stored once; always `TaskAID < TaskBID` (canonical ordering) |
| `GlobalSetting` | `Key`, `Value` | Org-wide config: `org.timezone`, `org.locale` |

---

## Services

| Service | Key responsibilities |
|---------|---------------------|
| `AuthService` | Register, login, JWT issue and refresh |
| `UserService` | Profile get/update, avatar upload/delete |
| `ProjectService` | Project CRUD, member management |
| `IssueService` | Issue CRUD; cascade delete (attachments + relations); `GetMyIssues` across projects |
| `BoardService` | Kanban view — issues grouped by status for a project |
| `SprintService` | Sprint lifecycle (planned → active → completed); burndown data; velocity; backlog |
| `CommentService` | Comment CRUD; notification fan-out to assignee and reporter |
| `ActivityService` | Record field-change events; list by issue or user |
| `LabelService` | Label CRUD; attach/detach to issues |
| `AttachmentService` | Upload to S3; generate pre-signed download URL; delete with storage key cleanup |
| `RelationService` | Create/list/delete symmetric task relations; enforces canonical ordering (`taskAID < taskBID`); guards against self-relations and cross-project links |
| `SettingService` | Get/update org-wide settings; write requires `IsSuperAdmin` |
| `SetupService` | First-run wizard: check if any users exist; create the superadmin account |

---

## Background jobs

Two goroutines run for the lifetime of the process, both started in `cmd/server/main.go` and stopped cleanly via `signal.NotifyContext` on SIGTERM/SIGINT:

| Job | Purpose | Interval |
|---|---|---|
| Revoked-token cleanup | Purges expired rows from the `revoked_tokens` table | `REVOKED_TOKEN_CLEANUP_INTERVAL` (default `15m`) |
| DB idle-check (spec 010) | Stops the database when idle; polls for a pending start to complete | `RDS_IDLE_CHECK_INTERVAL` (default `1m`) — AWS demo deployment only, see below |

## Database idle auto-stop

Cost-saving feature for the AWS demo deployment (spec 010; full spec/plan/ADR in the docs repo's `specs/010-db-idle-autostop/` and `docs/decisions/ADR-030-db-idle-autostop.md`). Stops the RDS instance after 10 minutes of no application activity, and starts it again the moment a request needs it — no new cloud infrastructure, no change to local development.

- **`DBLifecycleTracker`** (`internal/service/dbinstance_service.go`) — the only place lifecycle state lives: last-activity timestamp, in-flight operation count, start dedupe flag, and start-failure count, all behind one mutex.
- **Wake-on-hit**: a request hitting a stopped database gets an immediate `503 {"status":"waking_up",...}` instead of blocking (the `DBWake` middleware); the frontend (`app/plugins/api.ts`'s `fetchWithWakeRetry`) retries automatically with a loading indicator until it succeeds or 5 minutes elapse.
- **Never affects local dev**: `Config.RDSAutoStopApplies()` (`pkg/config/rds_hostname.go`) requires the driver to be postgres/postgresql/mysql/mariadb, the flag not explicitly disabled, an identifier configured, *and* `DB_DSN`'s hostname confirmed as that exact AWS RDS instance's endpoint (ends in `.rds.amazonaws.com`, starts with `<identifier>.`) — local Postgres/MySQL/MariaDB via docker-compose can never satisfy that last check, so it never engages this path, regardless of what the other config values happen to be set to.
- **Fails fast, not silently**: if the feature is enabled but the configured instance can't be described (bad identifier, missing IAM permissions), the backend refuses to start rather than pretending to work.

---

## Error handling

`domain/errors.go` defines sentinel errors:

```go
var (
    ErrNotFound  = errors.New("not found")
    ErrForbidden = errors.New("forbidden")
    ErrConflict  = errors.New("conflict")
    // domain-specific
    ErrSelfRelation        = errors.New("cannot relate a task to itself")
    ErrCrossProjectRelation = errors.New("cannot relate tasks across projects")
)
```

Handlers call `apierror.HTTPErrorHandler` which maps these to HTTP status codes:

| Error | Status |
|-------|--------|
| `ErrNotFound` | 404 |
| `ErrForbidden` | 403 |
| `ErrConflict` | 409 |
| `ErrSelfRelation` / `ErrCrossProjectRelation` | 422 |
| Validation error | 400 |
| Other | 500 |

---

## Testing approach

- **Repository tests**: real SQLite in-memory DB via `database.Open` in `main_test.go`; tests verify actual SQL behaviour
- **Service tests**: mock repositories (stub structs implementing the interface); test business logic in isolation
- **Handler tests**: Echo `httptest` with mock services; test HTTP binding, status codes, and response shape
- All tests run with `-race` flag in CI to catch data races
