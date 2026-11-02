package repository

import (
	"context"
	"errors"

	"github.com/sharique/mansooba/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	// key is a reserved word in MySQL, so the column is quoted by the dialect, not written by hand.
	if err := r.db.WithContext(ctx).Where("LOWER(?) = LOWER(?)", clause.Column{Name: "key"}, key).First(&project).Error; err != nil {
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

// CreateWithAdmin inserts the project and its first admin membership in one
// transaction, so a project never exists without the admin who made it.
func (r *projectRepo) CreateWithAdmin(ctx context.Context, project *domain.Project, adminUserID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(project).Error; err != nil {
			return err
		}
		return tx.Create(&domain.ProjectMember{ProjectID: project.ID, UserID: adminUserID, Role: domain.RoleAdmin}).Error
	})
}

// FindWithoutActiveAdmin returns the projects with no admin member whose account
// is active, oldest first. A project with no members at all is included.
func (r *projectRepo) FindWithoutActiveAdmin(ctx context.Context) ([]*domain.Project, error) {
	var projects []*domain.Project
	err := r.db.WithContext(ctx).
		Where(`NOT EXISTS (
			SELECT 1 FROM project_members pm
			JOIN users u ON u.id = pm.user_id
			WHERE pm.project_id = projects.id AND pm.role = ? AND u.is_active = ?)`, domain.RoleAdmin, true).
		Order("id ASC").Find(&projects).Error
	return projects, err
}

// DeleteIfEmpty removes a project and everything that belongs only to it, in one
// transaction. The project row is locked first, then the tasks are counted: if
// any remain the transaction rolls back untouched. A failure part-way leaves the
// project exactly as it was.
func (r *projectRepo) DeleteIfEmpty(ctx context.Context, projectID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockProject(tx, projectID); err != nil {
			return err
		}
		var tasks int64
		if err := tx.Model(&domain.Issue{}).Where("project_id = ?", projectID).Count(&tasks).Error; err != nil {
			return err
		}
		if tasks > 0 {
			return &domain.ProjectNotEmptyError{TaskCount: int(tasks)}
		}
		labelIDs := tx.Model(&domain.Label{}).Select("id").Where("project_id = ?", projectID)
		if err := tx.Where("label_id IN (?)", labelIDs).Delete(&domain.IssueLabel{}).Error; err != nil {
			return err
		}
		for _, model := range []any{&domain.Label{}, &domain.Sprint{}, &domain.ProjectMember{}} {
			if err := tx.Where("project_id = ?", projectID).Delete(model).Error; err != nil {
				return err
			}
		}
		return tx.Delete(&domain.Project{}, projectID).Error
	})
}
