package dto

import "time"

type CreateLabelRequest struct {
	Name  string `json:"name"  validate:"required"`
	Color string `json:"color" validate:"required"`
}

type LabelResponse struct {
	ID        uint      `json:"id"`
	ProjectID uint      `json:"project_id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
}
