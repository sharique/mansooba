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
	access     Access
}

// NewActivityService returns an ActivityService backed by the given repositories.
func NewActivityService(repo domain.ActivityRepository, userRepo domain.UserRepository, issueRepo domain.IssueRepository, memberRepo domain.ProjectMemberRepository) ActivityService {
	return &activityServiceImpl{repo: repo, userRepo: userRepo, issueRepo: issueRepo, memberRepo: memberRepo, access: NewAccess(nil, memberRepo, userRepo)}
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
	if _, err := s.access.RequireProject(ctx, issue.ProjectID, callerID, domain.CapProjectView); err != nil {
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
	// Only events in projects the person still belongs to: an event quotes a
	// task's key and title, which someone who has been removed must not keep reading.
	visible, err := s.visibleToMember(ctx, actorID, events)
	if err != nil {
		return nil, err
	}
	return s.enrich(ctx, visible), nil
}

// visibleToMember keeps the events whose project the person is currently a
// member of. An event whose task is gone (other than a deletion event, which
// carries its project) has nothing to show and is dropped.
func (s *activityServiceImpl) visibleToMember(ctx context.Context, userID uint, events []*domain.ActivityEvent) ([]*domain.ActivityEvent, error) {
	member := map[uint]bool{}
	allowed := func(projectID uint) (bool, error) {
		if ok, seen := member[projectID]; seen {
			return ok, nil
		}
		_, err := s.access.RequireProject(ctx, projectID, userID, domain.CapProjectView)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return false, err
		}
		member[projectID] = err == nil
		return err == nil, nil
	}
	out := make([]*domain.ActivityEvent, 0, len(events))
	for _, e := range events {
		projectID := e.ProjectID
		if e.Kind != domain.ActivityIssueDeleted {
			issue, err := s.issueRepo.FindByID(ctx, e.IssueID)
			if errors.Is(err, domain.ErrNotFound) {
				continue
			}
			if err != nil {
				return nil, err
			}
			projectID = issue.ProjectID
		}
		ok, err := allowed(projectID)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, e)
		}
	}
	return out, nil
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
		key, title := issueKeys[e.IssueID], issueTitles[e.IssueID]
		if e.Kind == domain.ActivityIssueDeleted {
			// The task is gone; the event carries its key and title.
			key, title = e.IssueKey, e.IssueTitle
		}
		result = append(result, &dto.ActivityEventResponse{
			ID:             e.ID,
			IssueID:        e.IssueID,
			ActorID:        e.ActorID,
			ActorName:      actor.Name,
			ActorAvatarURL: actor.AvatarURL,
			IssueKey:       key,
			IssueTitle:     title,
			Kind:           e.Kind,
			OldValue:       e.OldValue,
			NewValue:       e.NewValue,
			CreatedAt:      e.CreatedAt,
		})
	}
	return result
}
