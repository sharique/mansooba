package dto

import "time"

type CreateSprintRequest struct {
	Name      string     `json:"name"       validate:"required,min=1,max=255"`
	Goal      string     `json:"goal"`
	StartDate *time.Time `json:"start_date"`
	EndDate   *time.Time `json:"end_date"`
}

type UpdateSprintRequest struct {
	Name      *string    `json:"name"       validate:"omitempty,min=1,max=255"`
	Goal      *string    `json:"goal"`
	StartDate *time.Time `json:"start_date"`
	EndDate   *time.Time `json:"end_date"`
}

type CompleteSprintRequest struct {
	NextSprintID *uint `json:"next_sprint_id"`
}

type SprintResponse struct {
	ID        uint       `json:"id"`
	ProjectID uint       `json:"project_id"`
	Name      string     `json:"name"`
	Goal      string     `json:"goal"`
	Status    string     `json:"status"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type BurndownPoint struct {
	Date            string `json:"date"`
	RemainingPoints int    `json:"remaining_points"`
}

type BurndownResponse struct {
	SprintID    uint            `json:"sprint_id"`
	SprintName  string          `json:"sprint_name"`
	StartDate   string          `json:"start_date"`
	EndDate     string          `json:"end_date"`
	TotalPoints int             `json:"total_points"`
	Data        []BurndownPoint `json:"data"`
}
