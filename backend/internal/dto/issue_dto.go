package dto

// Note: oneof validator tags intentionally duplicate domain.IssueType*/IssuePriority*/IssueStatus*
// constants — Go struct tags are compile-time string literals and cannot reference variables.
type CreateIssueRequest struct {
	Title       string `json:"title"       validate:"required"`
	Description string `json:"description"`
	Type        string `json:"type"        validate:"required,oneof=task story bug epic"`
	Priority    string `json:"priority"    validate:"required,oneof=low medium high critical"`
	AssigneeID  *uint  `json:"assignee_id"`
	StoryPoints *int   `json:"story_points"`
}

type UpdateIssueRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Type        *string `json:"type"     validate:"omitempty,oneof=task story bug epic"`
	Status      *string `json:"status"   validate:"omitempty,oneof=backlog todo in_progress in_review done"`
	Priority    *string `json:"priority" validate:"omitempty,oneof=low medium high critical"`
	AssigneeID  *uint   `json:"assignee_id"`
	StoryPoints *int    `json:"story_points"`
	// SprintID assigns the issue to a sprint. Send 0 to move the issue back to the backlog.
	SprintID    *uint   `json:"sprint_id"`
}

type IssueListQuery struct {
	Type       string `query:"type"`
	Status     string `query:"status"`
	AssigneeID uint   `query:"assignee_id"`
	Page       int    `query:"page"`
	Limit      int    `query:"limit"`
}

type IssueResponse struct {
	ID          uint   `json:"id"`
	Key         string `json:"key"`
	ProjectID   uint   `json:"project_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
	AssigneeID  *uint  `json:"assignee_id,omitempty"`
	ReporterID  uint   `json:"reporter_id"`
	SprintID    *uint  `json:"sprint_id,omitempty"`
	StoryPoints *int   `json:"story_points,omitempty"`
}
