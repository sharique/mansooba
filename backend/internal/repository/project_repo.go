package repository

import (
	"context"
	"errors"

	"github.com/sharique/mansooba/internal/domain"
	"gorm.io/gorm"
)

type projectRepo struct {
	db *gorm.DB
}

// NewProjectRepository returns a GORM-backed implementation of domain.ProjectRepository.
func NewProjectRepository(db *gorm.DB) domain.ProjectRepository {
	return &projectRepo{db: db}
}

// Create inserts a new project record and populates the ID field on success.
func (r *projectRepo) Create(ctx context.Context, project *domain.Project) error {
	return r.db.WithContext(ctx).Create(project).Error
}

// FindByID retrieves a project by primary key.
// Returns domain.ErrNotFound when no row matches.
func (r *projectRepo) FindByID(ctx context.Context, id uint) (*domain.Project, error) {
	var project domain.Project
	if err := r.db.WithContext(ctx).First(&project, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &project, nil
}

// FindByKey retrieves a project by its unique key (e.g. "PROJ").
// Returns domain.ErrNotFound when no row matches.
func (r *projectRepo) FindByKey(ctx context.Context, key string) (*domain.Project, error) {
	var project domain.Project
	if err := r.db.WithContext(ctx).Where("LOWER(key) = LOWER(?)", key).First(&project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &project, nil
}

// FindByUserID returns all projects whose owner_id matches the given user.
func (r *projectRepo) FindByUserID(ctx context.Context, userID uint) ([]*domain.Project, error) {
	var projects []*domain.Project
	if err := r.db.WithContext(ctx).Where("owner_id = ?", userID).Find(&projects).Error; err != nil {
		return nil, err
	}
	return projects, nil
}

// Update saves all fields of an existing project record.
func (r *projectRepo) Update(ctx context.Context, project *domain.Project) error {
	return r.db.WithContext(ctx).Save(project).Error
}

// Delete removes a project by primary key.
func (r *projectRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&domain.Project{}, id).Error
}
