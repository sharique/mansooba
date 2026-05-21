package service

import (
	"context"

	"github.com/sharique/jira-go/internal/domain"
	"github.com/sharique/jira-go/internal/dto"
)

// ActivityService records and retrieves issue state-change events.
type ActivityService interface {
	Record(ctx context.Context, event *domain.ActivityEvent) error
	// ListByIssue returns enriched activity events for an issue (actor name + issue key/title resolved).
	ListByIssue(ctx context.Context, issueID uint) ([]*dto.ActivityEventResponse, error)
	// GetMyActivity returns enriched events where the caller was the actor (paginated).
	GetMyActivity(ctx context.Context, actorID uint, limit, offset int) ([]*dto.ActivityEventResponse, error)
}

type activityServiceImpl struct {
	repo      domain.ActivityRepository
	userRepo  domain.UserRepository
	issueRepo domain.IssueRepository
}

// NewActivityService returns an ActivityService backed by the given repositories.
func NewActivityService(repo domain.ActivityRepository, userRepo domain.UserRepository, issueRepo domain.IssueRepository) ActivityService {
	return &activityServiceImpl{repo: repo, userRepo: userRepo, issueRepo: issueRepo}
}

func (s *activityServiceImpl) Record(ctx context.Context, e *domain.ActivityEvent) error {
	return s.repo.Create(ctx, e)
}

func (s *activityServiceImpl) ListByIssue(ctx context.Context, issueID uint) ([]*dto.ActivityEventResponse, error) {
	events, err := s.repo.FindByIssueID(ctx, issueID)
	if err != nil {
		return nil, err
	}
	return s.enrich(ctx, events), nil
}

func (s *activityServiceImpl) GetMyActivity(ctx context.Context, actorID uint, limit, offset int) ([]*dto.ActivityEventResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	events, err := s.repo.FindByActorID(ctx, actorID, limit, offset)
	if err != nil {
		return nil, err
	}
	return s.enrich(ctx, events), nil
}

// enrich resolves actor names and issue key/title for a slice of activity events.
func (s *activityServiceImpl) enrich(ctx context.Context, events []*domain.ActivityEvent) []*dto.ActivityEventResponse {
	// Resolve unique actor IDs → names (one repo call per distinct actor).
	actorNames := make(map[uint]string)
	issueKeys := make(map[uint]string)
	issueTitles := make(map[uint]string)

	for _, e := range events {
		actorNames[e.ActorID] = ""
		issueKeys[e.IssueID] = ""
	}
	for id := range actorNames {
		if u, err := s.userRepo.FindByID(ctx, id); err == nil {
			actorNames[id] = u.Name
		}
	}
	// Resolve unique issue IDs → keys and titles (one repo call per distinct issue).
	for id := range issueKeys {
		if issue, err := s.issueRepo.FindByID(ctx, id); err == nil {
			issueKeys[id] = issue.Key
			issueTitles[id] = issue.Title
		}
	}

	result := make([]*dto.ActivityEventResponse, 0, len(events))
	for _, e := range events {
		result = append(result, &dto.ActivityEventResponse{
			ID:         e.ID,
			IssueID:    e.IssueID,
			ActorID:    e.ActorID,
			ActorName:  actorNames[e.ActorID],
			IssueKey:   issueKeys[e.IssueID],
			IssueTitle: issueTitles[e.IssueID],
			Kind:       e.Kind,
			OldValue:   e.OldValue,
			NewValue:   e.NewValue,
			CreatedAt:  e.CreatedAt,
		})
	}
	return result
}
