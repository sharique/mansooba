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
// Role is one of RoleAdmin, RoleMember, RoleViewer (see access.go); a person
// has at most one membership per project, enforced by the unique index.
type ProjectMember struct {
	ID        uint      `gorm:"primaryKey"`
	ProjectID uint      `gorm:"not null;uniqueIndex:idx_project_user"`
	UserID    uint      `gorm:"not null;index;uniqueIndex:idx_project_user"`
	Role      string    `gorm:"not null"`
	CreatedAt time.Time
}

// ProjectRepository defines the persistence contract for Project.
type ProjectRepository interface {
	// Create persists a new project and sets its ID on success.
	Create(ctx context.Context, project *Project) error
	// FindByID returns a project by primary key, or ErrNotFound if absent.
	FindByID(ctx context.Context, id uint) (*Project, error)
	// FindByKey returns a project by its unique key, or ErrNotFound if absent.
	FindByKey(ctx context.Context, key string) (*Project, error)
	// FindByUserID returns all projects owned by the given user.
	FindByUserID(ctx context.Context, userID uint) ([]*Project, error)
	// Update persists all fields of an existing project.
	Update(ctx context.Context, project *Project) error
	// Delete removes a project by primary key.
	Delete(ctx context.Context, id uint) error
	// DeleteIfEmpty removes a project with its members, sprints and labels in one
	// transaction, taking the project lock first and counting tasks under it. If
	// any task remains nothing is deleted and *ProjectNotEmptyError carries the
	// count; ErrNotFound when the project does not exist.
	DeleteIfEmpty(ctx context.Context, projectID uint) error
	// CreateWithAdmin persists a new project together with its first admin
	// membership in one transaction: either both exist afterwards or neither.
	CreateWithAdmin(ctx context.Context, project *Project, adminUserID uint) error
	// FindWithoutActiveAdmin returns the projects that have no admin member whose
	// account is active (an abandoned project), ordered by id.
	FindWithoutActiveAdmin(ctx context.Context) ([]*Project, error)
}

// ProjectMemberRepository defines the persistence contract for ProjectMember.
type ProjectMemberRepository interface {
	// Create adds a user as a member of a project with the specified role.
	Create(ctx context.Context, member *ProjectMember) error
	// FindByProjectID returns all members of a project.
	FindByProjectID(ctx context.Context, projectID uint) ([]*ProjectMember, error)
	// FindByUserID returns all membership records for a given user (across all projects).
	FindByUserID(ctx context.Context, userID uint) ([]*ProjectMember, error)
	// FindByProjectAndUser returns the membership record for a specific user in a project,
	// or ErrNotFound if the user is not a member.
	FindByProjectAndUser(ctx context.Context, projectID, userID uint) (*ProjectMember, error)
	// Delete removes a membership record by primary key.
	Delete(ctx context.Context, id uint) error
	// DeleteByProjectID removes all membership records for a project (used during project deletion).
	DeleteByProjectID(ctx context.Context, projectID uint) error
	// ChangeRole sets a member's role. It runs under a lock on the project and
	// returns ErrLastProjectAdmin, changing nothing, when the change would leave
	// a project that had an active admin without one, and ErrNotFound when the
	// user is not a member.
	ChangeRole(ctx context.Context, projectID, userID uint, role string) error
	// Remove deletes a member under the same lock and with the same
	// ErrLastProjectAdmin guard as ChangeRole.
	Remove(ctx context.Context, projectID, userID uint) error
	// CountActiveAdmins counts the project's admins whose account is active.
	CountActiveAdmins(ctx context.Context, projectID uint) (int64, error)
	// AddAdminIfNone makes userID an admin of the project, but only when the
	// project has no active admin (the rescue of an abandoned project). It
	// returns ErrConflict, changing nothing, when one exists, and ErrNotFound
	// when the project does not.
	AddAdminIfNone(ctx context.Context, projectID, userID uint) (*ProjectMember, error)
}
