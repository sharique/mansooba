package dto

import "time"

// SystemLogEntryDTO is the public representation of a System Log entry
// (011-system-logs, contracts/system-logs-api.md). ActorLabel is never
// null (FR-004) — it's "system" for db_lifecycle entries. TargetLabel is
// nullable, since not every category has a distinct target.
type SystemLogEntryDTO struct {
	ID            string    `json:"id"`
	EventCategory string    `json:"event_category"`
	Action        string    `json:"action"`
	Outcome       string    `json:"outcome"`
	ActorLabel    string    `json:"actor_label"`
	TargetLabel   *string   `json:"target_label"`
	Detail        string    `json:"detail"`
	CreatedAt     time.Time `json:"created_at"`
}

// SystemLogListResponse is returned by GET /api/v1/admin/system-logs.
type SystemLogListResponse struct {
	Entries []SystemLogEntryDTO `json:"entries"`
	Total   int                 `json:"total"`
	Page    int                 `json:"page"`
	Size    int                 `json:"size"`
}

// SystemLogListQuery is the parsed/validated form of GET /admin/system-logs's
// query parameters (contracts/system-logs-api.md).
type SystemLogListQuery struct {
	Category string
	From     *time.Time
	To       *time.Time
	Actor    string
	Q        string
	Page     int
	Size     int
}
