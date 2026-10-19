package dto

import "time"

// StatusCount is the number of tasks and the story points in one status.
type StatusCount struct {
	Tasks  int `json:"tasks"`
	Points int `json:"points"`
}

// SprintSummaryResponse is the active sprint's progress, counted in the board's
// four columns.
type SprintSummaryResponse struct {
	ID         uint        `json:"id"`
	Name       string      `json:"name"`
	StartDate  *time.Time  `json:"start_date,omitempty"`
	EndDate    *time.Time  `json:"end_date,omitempty"`
	IsOverdue  bool        `json:"is_overdue"`
	Todo       StatusCount `json:"todo"`
	InProgress StatusCount `json:"in_progress"`
	InReview   StatusCount `json:"in_review"`
	Done       StatusCount `json:"done"`
	Total      StatusCount `json:"total"`
}

// OverviewSprintResponse is returned by GET /projects/{key}/overview/sprint.
// Sprint is null when no sprint is active.
type OverviewSprintResponse struct {
	Sprint *SprintSummaryResponse `json:"sprint"`
}

// PersonRef names a user in an overview list. Name is empty for a removed user.
type PersonRef struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

// OpenTaskResponse is one open task of the active sprint.
type OpenTaskResponse struct {
	IssueID  uint       `json:"issue_id"`
	Key      string     `json:"key"`
	Title    string     `json:"title"`
	Status   string     `json:"status"`
	Priority string     `json:"priority"`
	Assignee *PersonRef `json:"assignee"`
}

// OverviewTasksResponse is returned by GET /projects/{key}/overview/tasks.
type OverviewTasksResponse struct {
	SprintActive bool               `json:"sprint_active"`
	Items        []OpenTaskResponse `json:"items"`
}

// RecentActivityResponse is one progress or comment event. It never carries the
// text of a comment.
type RecentActivityResponse struct {
	ID         uint      `json:"id"`
	Kind       string    `json:"kind"` // status_changed | comment_added
	Actor      PersonRef `json:"actor"`
	IssueID    uint      `json:"issue_id"`
	IssueKey   string    `json:"issue_key"`
	IssueTitle string    `json:"issue_title"`
	NewStatus  string    `json:"new_status,omitempty"` // status changes only
	CreatedAt  time.Time `json:"created_at"`
}

// OverviewActivityResponse is returned by GET /projects/{key}/overview/activity.
type OverviewActivityResponse struct {
	SprintActive bool                     `json:"sprint_active"`
	Items        []RecentActivityResponse `json:"items"`
}
