package service

import (
	"context"
	"errors"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
)

// ActivityService records and retrieves issue state-change events.
type ActivityService interface {
	Record(ctx context.Context, event *domain.ActivityEvent) error
	// ListByIssue returns enriched activity events for an issue (actor name + issue key/title resolved).
	// The caller must be a member of the issue's project (domain.ErrForbidden otherwise).
	ListByIssue(ctx context.Context, issueID, callerID uint) ([]*dto.ActivityEventResponse, error)
	// GetMyActivity returns enriched events where the caller was the actor (paginated).
	GetMyActivity(ctx context.Context, actorID uint, limit, offset int) ([]*dto.ActivityEventResponse, error)
}

type activityServiceImpl struct {
	repo       domain.ActivityRepository
	userRepo   domain.UserRepository
	issueRepo  domain.IssueRepository
	memberRepo domain.ProjectMemberRepository
}

// NewActivityService returns an ActivityService backed by the given repositories.
func NewActivityService(repo domain.ActivityRepository, userRepo domain.UserRepository, issueRepo domain.IssueRepository, memberRepo domain.ProjectMemberRepository) ActivityService {
	return &activityServiceImpl{repo: repo, userRepo: userRepo, issueRepo: issueRepo, memberRepo: memberRepo}
}

func (s *activityServiceImpl) Record(ctx context.Context, e *domain.ActivityEvent) error {
	return s.repo.Create(ctx, e)
}

func (s *activityServiceImpl) ListByIssue(ctx context.Context, issueID, callerID uint) ([]*dto.ActivityEventResponse, error) {
	issue, err := s.issueRepo.FindByID(ctx, issueID)
	if err != nil {
		return nil, err
	}
	// Activity names the people involved and quotes the issue's key and title,
	// so it is as private as the issue itself: members only.
	if _, err := s.memberRepo.FindByProjectAndUser(ctx, issue.ProjectID, callerID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrForbidden
		}
		return nil, err
	}
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

type actorInfo struct {
	Name      string
	AvatarURL string
}

// enrich resolves actor names/avatars and issue key/title for a slice of activity events.
func (s *activityServiceImpl) enrich(ctx context.Context, events []*domain.ActivityEvent) []*dto.ActivityEventResponse {
	// Resolve unique actor IDs → name + avatar (one repo call per distinct actor).
	actors := make(map[uint]actorInfo)
	issueKeys := make(map[uint]string)
	issueTitles := make(map[uint]string)

	for _, e := range events {
		actors[e.ActorID] = actorInfo{}
		issueKeys[e.IssueID] = ""
	}
	for id := range actors {
		if u, err := s.userRepo.FindByID(ctx, id); err == nil {
			actors[id] = actorInfo{Name: u.Name, AvatarURL: u.AvatarURL}
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
		actor := actors[e.ActorID]
		result = append(result, &dto.ActivityEventResponse{
			ID:             e.ID,
			IssueID:        e.IssueID,
			ActorID:        e.ActorID,
			ActorName:      actor.Name,
			ActorAvatarURL: actor.AvatarURL,
			IssueKey:       issueKeys[e.IssueID],
			IssueTitle:     issueTitles[e.IssueID],
			Kind:           e.Kind,
			OldValue:       e.OldValue,
			NewValue:       e.NewValue,
			CreatedAt:      e.CreatedAt,
		})
	}
	return result
}
