package repository

import (
	"context"
	"errors"

	"github.com/sharique/mansooba/internal/domain"
	"gorm.io/gorm"
)

type sprintRepo struct {
	db *gorm.DB
}

// NewSprintRepository returns a GORM-backed domain.SprintRepository.
func NewSprintRepository(db *gorm.DB) domain.SprintRepository {
	return &sprintRepo{db: db}
}

// Create inserts a new sprint and populates its ID.
func (r *sprintRepo) Create(ctx context.Context, sprint *domain.Sprint) error {
	return r.db.WithContext(ctx).Create(sprint).Error
}

// FindByID retrieves a sprint by primary key. Returns domain.ErrNotFound when absent.
func (r *sprintRepo) FindByID(ctx context.Context, id uint) (*domain.Sprint, error) {
	var sprint domain.Sprint
	if err := r.db.WithContext(ctx).First(&sprint, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &sprint, nil
}

// FindByProject returns all sprints for a project, ordered by created_at ASC.
func (r *sprintRepo) FindByProject(ctx context.Context, projectID uint) ([]*domain.Sprint, error) {
	var sprints []*domain.Sprint
	if err := r.db.WithContext(ctx).
		Where("project_id = ?", projectID).
		Order("created_at ASC").
		Find(&sprints).Error; err != nil {
		return nil, err
	}
	return sprints, nil
}

// Update saves all fields of an existing sprint.
func (r *sprintRepo) Update(ctx context.Context, sprint *domain.Sprint) error {
	return r.db.WithContext(ctx).Save(sprint).Error
}

// Delete removes a sprint by primary key.
func (r *sprintRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&domain.Sprint{}, id).Error
}

// FindActiveByProject returns the active sprint for a project, or nil when none exists.
// A nil return with a nil error is not an error — it means no sprint is currently active.
func (r *sprintRepo) FindActiveByProject(ctx context.Context, projectID uint) (*domain.Sprint, error) {
	var sprint domain.Sprint
	err := r.db.WithContext(ctx).
		Where("project_id = ? AND status = 'active'", projectID).
		First(&sprint).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &sprint, nil
}

// FindWithIssues returns a sprint with its Issues slice preloaded.
func (r *sprintRepo) FindWithIssues(ctx context.Context, id uint) (*domain.Sprint, error) {
	var sprint domain.Sprint
	if err := r.db.WithContext(ctx).Preload("Issues").First(&sprint, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &sprint, nil
}

// FindCompletedWithIssuesByProject returns completed sprints for a project with their Issues
// preloaded in a single query (WHERE project_id AND status='completed' + Preload("Issues")).
func (r *sprintRepo) FindCompletedWithIssuesByProject(ctx context.Context, projectID uint) ([]*domain.Sprint, error) {
	var sprints []*domain.Sprint
	if err := r.db.WithContext(ctx).
		Preload("Issues").
		Where("project_id = ? AND status = ?", projectID, domain.SprintStatusCompleted).
		Order("created_at ASC").
		Find(&sprints).Error; err != nil {
		return nil, err
	}
	return sprints, nil
}

// CompleteWithMigration atomically marks a sprint completed and migrates unfinished issues.
// The sprint update and issue bulk-update run in a single DB transaction.
func (r *sprintRepo) CompleteWithMigration(ctx context.Context, sprint *domain.Sprint, unfinishedIDs []uint, nextSprintID *uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(sprint).Error; err != nil {
			return err
		}
		if len(unfinishedIDs) == 0 {
			return nil
		}
		q := tx.Model(&domain.Issue{}).Where("id IN ?", unfinishedIDs)
		if nextSprintID != nil {
			return q.Update("sprint_id", nextSprintID).Error
		}
		return q.Update("sprint_id", nil).Error
	})
}
