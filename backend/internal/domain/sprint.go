package domain

import (
	"context"
	"time"
)

// Sprint is a time-boxed iteration within a project.
// Status is one of: "planning", "active", "completed".
// Issues are associated via Issue.SprintID; nil SprintID means the issue is in the backlog.
type Sprint struct {
	ID        uint       `gorm:"primaryKey"`
	ProjectID uint       `gorm:"not null;index"`
	Name      string     `gorm:"not null"`
	Goal      string
	Status    string     `gorm:"not null;default:'planning'"` // planning | active | completed
	StartDate *time.Time
	EndDate   *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time

	// Issues is populated only when explicitly preloaded (e.g. FindWithIssues).
	Issues []Issue `gorm:"foreignKey:SprintID"`
}

// SprintRepository defines all persistence operations for sprints.
// Implementations live in internal/repository/sprint_repo.go.
type SprintRepository interface {
	// Create persists a new sprint and sets its ID on success.
	Create(ctx context.Context, sprint *Sprint) error

	// FindByID returns the sprint with the given ID, or domain.ErrNotFound.
	FindByID(ctx context.Context, id uint) (*Sprint, error)

	// FindByProject returns all sprints for a project, ordered by created_at ASC.
	FindByProject(ctx context.Context, projectID uint) ([]*Sprint, error)

	// Update persists all fields of an existing sprint.
	Update(ctx context.Context, sprint *Sprint) error

	// Delete removes a sprint by ID.
	Delete(ctx context.Context, id uint) error

	// FindActiveByProject returns the currently Active sprint for a project,
	// or nil (no error) when no sprint is active.
	FindActiveByProject(ctx context.Context, projectID uint) (*Sprint, error)

	// FindWithIssues returns a sprint with its Issues slice preloaded.
	// Used by Complete and Burndown.
	FindWithIssues(ctx context.Context, id uint) (*Sprint, error)

	// CompleteWithMigration atomically marks the sprint completed and migrates
	// unfinished issues. If nextSprintID is nil, issues move to the backlog (sprint_id = NULL).
	// Executes inside a single DB transaction.
	CompleteWithMigration(ctx context.Context, sprint *Sprint, unfinishedIDs []uint, nextSprintID *uint) error
}
