package repository

import (
	"context"

	"github.com/sharique/jira-go/internal/domain"
	"gorm.io/gorm"
)

type activityRepo struct{ db *gorm.DB }

func NewActivityRepository(db *gorm.DB) domain.ActivityRepository {
	return &activityRepo{db: db}
}

func (r *activityRepo) Create(ctx context.Context, e *domain.ActivityEvent) error {
	return r.db.WithContext(ctx).Create(e).Error
}

func (r *activityRepo) FindByIssueID(ctx context.Context, issueID uint) ([]*domain.ActivityEvent, error) {
	var events []*domain.ActivityEvent
	if err := r.db.WithContext(ctx).
		Where("issue_id = ?", issueID).
		Order("created_at ASC").
		Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

func (r *activityRepo) FindByActorID(ctx context.Context, actorID uint, limit, offset int) ([]*domain.ActivityEvent, error) {
	var events []*domain.ActivityEvent
	if err := r.db.WithContext(ctx).
		Where("actor_id = ?", actorID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}
