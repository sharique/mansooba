package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/sharique/mansooba/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type projectMemberRepo struct {
	db *gorm.DB
}

// NewProjectMemberRepository returns a GORM-backed implementation of domain.ProjectMemberRepository.
func NewProjectMemberRepository(db *gorm.DB) domain.ProjectMemberRepository {
	return &projectMemberRepo{db: db}
}

// Create inserts a new project membership record and populates the ID field on success.
func (r *projectMemberRepo) Create(ctx context.Context, member *domain.ProjectMember) error {
	if err := r.db.WithContext(ctx).Create(member).Error; err != nil {
		return translateMembershipConflict(err)
	}
	return nil
}

// translateMembershipConflict turns the unique-index violation on (project, user)
// into domain.ErrConflict. The drivers word it differently, so match the words.
func translateMembershipConflict(err error) error {
	msg := strings.ToLower(err.Error())
	if (strings.Contains(msg, "unique") || strings.Contains(msg, "duplicate")) &&
		(strings.Contains(msg, "idx_project_user") || strings.Contains(msg, "project_members")) {
		return domain.ErrConflict
	}
	return err
}

// FindByProjectID returns all membership records for a given project.
func (r *projectMemberRepo) FindByProjectID(ctx context.Context, projectID uint) ([]*domain.ProjectMember, error) {
	var members []*domain.ProjectMember
	if err := r.db.WithContext(ctx).Where("project_id = ?", projectID).Find(&members).Error; err != nil {
		return nil, err
	}
	return members, nil
}

// FindByProjectAndUser returns the membership record for a specific user in a project.
// Returns domain.ErrNotFound when the user is not a member.
func (r *projectMemberRepo) FindByProjectAndUser(ctx context.Context, projectID, userID uint) (*domain.ProjectMember, error) {
	var member domain.ProjectMember
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &member, nil
}

// FindByUserID returns all membership records for a given user across all projects.
func (r *projectMemberRepo) FindByUserID(ctx context.Context, userID uint) ([]*domain.ProjectMember, error) {
	var members []*domain.ProjectMember
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&members).Error; err != nil {
		return nil, err
	}
	return members, nil
}

// Delete removes a membership record by primary key.
func (r *projectMemberRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&domain.ProjectMember{}, id).Error
}

// DeleteByProjectID removes all membership records belonging to a project.
func (r *projectMemberRepo) DeleteByProjectID(ctx context.Context, projectID uint) error {
	return r.db.WithContext(ctx).Where("project_id = ?", projectID).Delete(&domain.ProjectMember{}).Error
}

// lockProject takes a row lock on the project for the rest of the transaction,
// so two requests that change who administers it run one after the other. The
// SQLite driver ignores the clause: its single connection already serialises
// transactions. It returns ErrNotFound for a project that does not exist.
func lockProject(tx *gorm.DB, projectID uint) error {
	var p domain.Project
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&p, projectID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.ErrNotFound
	}
	return err
}

// countActiveAdmins counts a project's admin members whose account is active.
func countActiveAdmins(tx *gorm.DB, projectID uint) (int64, error) {
	var n int64
	err := tx.Table("project_members").
		Joins("JOIN users ON users.id = project_members.user_id").
		Where("project_members.project_id = ? AND project_members.role = ? AND users.is_active = ?", projectID, domain.RoleAdmin, true).
		Count(&n).Error
	return n, err
}

// CountActiveAdmins counts the project's admins whose account is active.
func (r *projectMemberRepo) CountActiveAdmins(ctx context.Context, projectID uint) (int64, error) {
	return countActiveAdmins(r.db.WithContext(ctx), projectID)
}

// guardedChange runs change under the project lock and rolls it back when it
// would leave a project that had an active admin with none.
func (r *projectMemberRepo) guardedChange(ctx context.Context, projectID uint, change func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockProject(tx, projectID); err != nil {
			return err
		}
		before, err := countActiveAdmins(tx, projectID)
		if err != nil {
			return err
		}
		if err := change(tx); err != nil {
			return err
		}
		after, err := countActiveAdmins(tx, projectID)
		if err != nil {
			return err
		}
		if before > 0 && after == 0 {
			return domain.ErrLastProjectAdmin
		}
		return nil
	})
}

// ChangeRole sets a member's role under the project lock; see the interface.
func (r *projectMemberRepo) ChangeRole(ctx context.Context, projectID, userID uint, role string) error {
	return r.guardedChange(ctx, projectID, func(tx *gorm.DB) error {
		res := tx.Model(&domain.ProjectMember{}).
			Where("project_id = ? AND user_id = ?", projectID, userID).
			Update("role", role)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

// Remove deletes a member under the project lock; see the interface.
func (r *projectMemberRepo) Remove(ctx context.Context, projectID, userID uint) error {
	return r.guardedChange(ctx, projectID, func(tx *gorm.DB) error {
		res := tx.Where("project_id = ? AND user_id = ?", projectID, userID).Delete(&domain.ProjectMember{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

// AddAdminIfNone makes userID admin of a project that has no active admin.
func (r *projectMemberRepo) AddAdminIfNone(ctx context.Context, projectID, userID uint) (*domain.ProjectMember, error) {
	var out domain.ProjectMember
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockProject(tx, projectID); err != nil {
			return err
		}
		n, err := countActiveAdmins(tx, projectID)
		if err != nil {
			return err
		}
		if n > 0 {
			return domain.ErrConflict
		}
		res := tx.Where("project_id = ? AND user_id = ?", projectID, userID).Limit(1).Find(&out)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected > 0 {
			out.Role = domain.RoleAdmin
			return tx.Model(&domain.ProjectMember{}).Where("id = ?", out.ID).Update("role", domain.RoleAdmin).Error
		}
		out = domain.ProjectMember{ProjectID: projectID, UserID: userID, Role: domain.RoleAdmin}
		return tx.Create(&out).Error
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}
