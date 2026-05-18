package service

import (
	"context"

	"github.com/sharique/jira-go/internal/domain"
)

// ActivityService records and retrieves issue state-change events.
type ActivityService interface {
	Record(ctx context.Context, event *domain.ActivityEvent) error
	ListByIssue(ctx context.Context, issueID uint) ([]*domain.ActivityEvent, error)
}

type activityServiceImpl struct {
	repo domain.ActivityRepository
}

func NewActivityService(repo domain.ActivityRepository) ActivityService {
	return &activityServiceImpl{repo: repo}
}

func (s *activityServiceImpl) Record(ctx context.Context, e *domain.ActivityEvent) error {
	return s.repo.Create(ctx, e)
}

func (s *activityServiceImpl) ListByIssue(ctx context.Context, issueID uint) ([]*domain.ActivityEvent, error) {
	return s.repo.FindByIssueID(ctx, issueID)
}
