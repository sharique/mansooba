package repository

import (
	"context"
	"errors"

	"github.com/sharique/mansooba/internal/domain"
	"gorm.io/gorm"
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
	return r.db.WithContext(ctx).Create(member).Error
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
