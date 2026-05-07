package domain

import (
	"context"
	"time"
)

// Issue is a unit of work within a project (task, story, bug, or epic).
// Key combines the project key and a sequential number, e.g. "PROJ-3".
// AssigneeID is a pointer because an issue may be unassigned.
type Issue struct {
	ID          uint      `gorm:"primaryKey"`
	Key         string    `gorm:"uniqueIndex;not null"`
	ProjectID   uint      `gorm:"not null;index"`
	Title       string    `gorm:"not null"`
	Description string
	Type        string    `gorm:"not null"` // task | story | bug | epic
	Status      string    `gorm:"not null"` // todo | in_progress | done
	Priority    string    `gorm:"not null"` // low | medium | high | critical
	AssigneeID  *uint
	ReporterID  uint      `gorm:"not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// IssueRepository defines the persistence contract for Issue.
type IssueRepository interface {
	// Create persists a new issue and sets its ID on success.
	Create(ctx context.Context, issue *Issue) error
	// FindByID returns an issue by primary key, or ErrNotFound if absent.
	FindByID(ctx context.Context, id uint) (*Issue, error)
	// FindByProjectID returns all issues belonging to a project.
	FindByProjectID(ctx context.Context, projectID uint) ([]*Issue, error)
	// Update persists all fields of an existing issue.
	Update(ctx context.Context, issue *Issue) error
	// Delete removes an issue by primary key.
	Delete(ctx context.Context, id uint) error
}
