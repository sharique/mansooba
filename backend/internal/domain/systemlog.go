package domain

import (
	"context"
	"time"
)

// System Logs event categories (FR-011). General application error/exception
// events are explicitly out of scope for this feature (spec.md Assumptions).
const (
	SystemLogCategoryAuthentication = "authentication"
	SystemLogCategoryAdminAction    = "admin_action"
	SystemLogCategorySettingsChange = "settings_change"
	SystemLogCategoryDBLifecycle    = "db_lifecycle"
)

// SystemLogActorSystem is the fixed actor value for entries with no human
// actor (FR-004, FR-013) — currently only db_lifecycle events. Never null or
// absent: every SystemLogEntry carries a non-empty Actor.
const SystemLogActorSystem = "system"

// SystemLogEntry is a single durable, immutable record of a security-relevant
// or operational event (011-system-logs). Persisted to Grafana Loki
// (ADR-031), not the application database — see SystemLogRepository. ID is
// populated only on read (synthesized by the repository); it is empty when
// constructing an entry for Create.
type SystemLogEntry struct {
	ID        string
	Category  string
	Action    string
	Outcome   string
	Actor     string
	ActorID   uint
	Target    string
	TargetID  uint
	Detail    string
	CreatedAt time.Time
}

// SystemLogListFilter narrows FindPaginated's results (FR-006, FR-007). All
// fields are optional — the zero value (empty string / nil time / zero int)
// means "unfiltered" for that dimension. From/To are pointers so "not set" is
// distinguishable from the zero time.Time value.
type SystemLogListFilter struct {
	Category string
	From     *time.Time
	To       *time.Time
	Actor    string
	Q        string
	Page     int
	Size     int
}

// SystemLogListResult is FindPaginated's return shape.
type SystemLogListResult struct {
	Entries []SystemLogEntry
	Total   int
}

// SystemLogRepository defines the persistence contract for SystemLogEntry.
// Implemented by a Grafana Loki-backed repository (ADR-031) — there is no
// GORM implementation; System Logs never writes to the application database.
type SystemLogRepository interface {
	// Create durably records entry. Callers needing best-effort semantics
	// (FR-012) do not call this directly — see service.SystemLogService.Record.
	Create(ctx context.Context, entry SystemLogEntry) error
	// FindPaginated returns entries matching filter, newest first.
	FindPaginated(ctx context.Context, filter SystemLogListFilter) (SystemLogListResult, error)
}
