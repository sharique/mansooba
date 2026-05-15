package dto

type CreateIssueRequest struct {
	Title       string `json:"title"       validate:"required"`
	Description string `json:"description"`
	Type        string `json:"type"        validate:"required,oneof=task story bug epic"`
	Priority    string `json:"priority"    validate:"required,oneof=low medium high critical"`
	AssigneeID  *uint  `json:"assignee_id"`
}

type UpdateIssueRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Type        *string `json:"type"     validate:"omitempty,oneof=task story bug epic"`
	Status      *string `json:"status"   validate:"omitempty,oneof=backlog todo in_progress in_review done"`
	Priority    *string `json:"priority" validate:"omitempty,oneof=low medium high critical"`
	AssigneeID  *uint   `json:"assignee_id"`
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
}
