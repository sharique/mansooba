package domain

import (
	"context"
	"time"
)

// Project is a top-level container for issues (analogous to a Jira project).
// Key is a short uppercase identifier, e.g. "PROJ", used to prefix issue keys.
type Project struct {
	ID          uint      `gorm:"primaryKey"`
	Key         string    `gorm:"uniqueIndex;not null"`
	Name        string    `gorm:"not null"`
	Description string
	OwnerID     uint      `gorm:"not null;index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ProjectMember records a user's membership and role within a project.
// Role is one of: "admin", "member", "viewer".
type ProjectMember struct {
	ID        uint      `gorm:"primaryKey"`
	ProjectID uint      `gorm:"not null;index"`
	UserID    uint      `gorm:"not null;index"`
	Role      string    `gorm:"not null"`
	CreatedAt time.Time
}

// ProjectRepository defines the persistence contract for Project.
type ProjectRepository interface {
	// Create persists a new project and sets its ID on success.
	Create(ctx context.Context, project *Project) error
	// FindByKey returns a project by its unique key, or ErrNotFound if absent.
	FindByKey(ctx context.Context, key string) (*Project, error)
	// FindByUserID returns all projects owned by the given user.
	FindByUserID(ctx context.Context, userID uint) ([]*Project, error)
	// Update persists all fields of an existing project.
	Update(ctx context.Context, project *Project) error
	// Delete removes a project by primary key.
	Delete(ctx context.Context, id uint) error
}

// ProjectMemberRepository defines the persistence contract for ProjectMember.
type ProjectMemberRepository interface {
	// Create adds a user as a member of a project with the specified role.
	Create(ctx context.Context, member *ProjectMember) error
	// FindByProjectID returns all members of a project.
	FindByProjectID(ctx context.Context, projectID uint) ([]*ProjectMember, error)
	// FindByProjectAndUser returns the membership record for a specific user in a project,
	// or ErrNotFound if the user is not a member.
	FindByProjectAndUser(ctx context.Context, projectID, userID uint) (*ProjectMember, error)
	// Delete removes a membership record by primary key.
	Delete(ctx context.Context, id uint) error
}
