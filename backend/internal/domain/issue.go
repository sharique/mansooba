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
	Status      string    `gorm:"not null"` // backlog | todo | in_progress | in_review | done
	Priority    string    `gorm:"not null"` // low | medium | high | critical
	AssigneeID  *uint
	ReporterID  uint      `gorm:"not null"`
	// SprintID is the sprint this issue belongs to. nil means the issue is in the backlog.
	SprintID    *uint `gorm:"index"`
	// StoryPoints is the effort estimate for burndown calculations. nil if not estimated.
	StoryPoints *int
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
	// DeleteByProjectID removes all issues belonging to a project (used during project deletion).
	DeleteByProjectID(ctx context.Context, projectID uint) error
	// FindBacklog returns issues with sprint_id IS NULL for a project,
	// ordered by priority (critical first) then created_at ASC.
	FindBacklog(ctx context.Context, projectID uint) ([]*Issue, error)
	// FindBySprint returns all issues assigned to the given sprint.
	FindBySprint(ctx context.Context, sprintID uint) ([]*Issue, error)
}
