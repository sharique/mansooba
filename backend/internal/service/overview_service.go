package service

import (
	"context"
	"fmt"
	"time"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
)

const (
	// overviewListLimit is how many open tasks and activity entries the overview shows.
	overviewListLimit = 10
	// overviewActivityWindow is how far back recent activity reaches.
	overviewActivityWindow = 5 * 24 * time.Hour

	overviewScopeAll    = "all"
	overviewScopeSprint = "sprint"
)

// OverviewService builds the read-only project overview: the active sprint's
// progress, its open tasks and recent activity. All methods require the caller to
// be a member of the project (domain.ErrForbidden otherwise).
type OverviewService interface {
	// Sprint returns the active sprint's progress; Sprint is nil when none is active.
	Sprint(ctx context.Context, projectKey string, callerID uint) (*dto.OverviewSprintResponse, error)
	// Tasks returns up to 10 open tasks of the active sprint, most urgent first.
	Tasks(ctx context.Context, projectKey string, callerID uint) (*dto.OverviewTasksResponse, error)
	// Activity returns up to 10 recent progress and comment events. scope is "all"
	// (default when empty) or "sprint"; anything else is domain.ErrInvalidInput.
	Activity(ctx context.Context, projectKey string, callerID uint, scope string) (*dto.OverviewActivityResponse, error)
}

type overviewService struct {
	projectRepo  domain.ProjectRepository
	memberRepo   domain.ProjectMemberRepository
	sprintRepo   domain.SprintRepository
	issueRepo    domain.IssueRepository
	activityRepo domain.ActivityRepository
	userRepo     domain.UserRepository
	now          func() time.Time
}

// NewOverviewService returns an OverviewService that reads the current time from
// the system clock.
func NewOverviewService(
	projectRepo domain.ProjectRepository,
	memberRepo domain.ProjectMemberRepository,
	sprintRepo domain.SprintRepository,
	issueRepo domain.IssueRepository,
	activityRepo domain.ActivityRepository,
	userRepo domain.UserRepository,
) OverviewService {
	return NewOverviewServiceWithClock(projectRepo, memberRepo, sprintRepo, issueRepo, activityRepo, userRepo, time.Now)
}

// NewOverviewServiceWithClock is NewOverviewService with an injectable clock, so
// the five-day window and the overdue rule can be tested at a fixed moment.
func NewOverviewServiceWithClock(
	projectRepo domain.ProjectRepository,
	memberRepo domain.ProjectMemberRepository,
	sprintRepo domain.SprintRepository,
	issueRepo domain.IssueRepository,
	activityRepo domain.ActivityRepository,
	userRepo domain.UserRepository,
	now func() time.Time,
) OverviewService {
	return &overviewService{
		projectRepo: projectRepo, memberRepo: memberRepo, sprintRepo: sprintRepo,
		issueRepo: issueRepo, activityRepo: activityRepo, userRepo: userRepo, now: now,
	}
}

// project resolves the key and checks membership, in that order, so an unknown
// key is 404 and a known project is 403 to a non-member.
func (s *overviewService) project(ctx context.Context, key string, callerID uint) (*domain.Project, error) {
	p, err := s.projectRepo.FindByKey(ctx, key)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	if _, err := s.memberRepo.FindByProjectAndUser(ctx, p.ID, callerID); err != nil {
		return nil, domain.ErrForbidden
	}
	return p, nil
}

func (s *overviewService) Sprint(ctx context.Context, projectKey string, callerID uint) (*dto.OverviewSprintResponse, error) {
	p, err := s.project(ctx, projectKey, callerID)
	if err != nil {
		return nil, err
	}
	sprint, err := s.sprintRepo.FindActiveByProject(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	if sprint == nil {
		return &dto.OverviewSprintResponse{}, nil
	}
	totals, err := s.issueRepo.CountByStatusForSprint(ctx, sprint.ID)
	if err != nil {
		return nil, err
	}

	sum := &dto.SprintSummaryResponse{
		ID: sprint.ID, Name: sprint.Name, StartDate: sprint.StartDate, EndDate: sprint.EndDate,
		IsOverdue: isOverdue(sprint.EndDate, s.now()),
	}
	for _, t := range totals {
		c := dto.StatusCount{Tasks: t.Tasks, Points: t.Points}
		switch t.Status {
		case domain.IssueStatusInProgress:
			add(&sum.InProgress, c)
		case domain.IssueStatusInReview:
			add(&sum.InReview, c)
		case domain.IssueStatusDone:
			add(&sum.Done, c)
		default: // todo, and backlog-status tasks that sit in a sprint
			add(&sum.Todo, c)
		}
		add(&sum.Total, c)
	}
	return &dto.OverviewSprintResponse{Sprint: sum}, nil
}

func add(dst *dto.StatusCount, c dto.StatusCount) {
	dst.Tasks += c.Tasks
	dst.Points += c.Points
}

// isOverdue is true from the day after the end date; the end day itself is not
// overdue. Days are UTC calendar days, like the dates the sprint form stores.
func isOverdue(end *time.Time, now time.Time) bool {
	if end == nil {
		return false
	}
	y, m, d := end.UTC().Date()
	ny, nm, nd := now.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Before(time.Date(ny, nm, nd, 0, 0, 0, 0, time.UTC))
}

func (s *overviewService) Tasks(ctx context.Context, projectKey string, callerID uint) (*dto.OverviewTasksResponse, error) {
	p, err := s.project(ctx, projectKey, callerID)
	if err != nil {
		return nil, err
	}
	resp := &dto.OverviewTasksResponse{Items: []dto.OpenTaskResponse{}}
	sprint, err := s.sprintRepo.FindActiveByProject(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	if sprint == nil {
		return resp, nil
	}
	resp.SprintActive = true
	issues, err := s.issueRepo.FindOpenBySprint(ctx, sprint.ID, overviewListLimit)
	if err != nil {
		return nil, err
	}
	people := s.personLookup(ctx)
	for _, i := range issues {
		item := dto.OpenTaskResponse{
			IssueID: i.ID, Key: i.Key, Title: i.Title, Status: i.Status, Priority: i.Priority,
		}
		if i.AssigneeID != nil {
			ref := people(*i.AssigneeID)
			item.Assignee = &ref
		}
		resp.Items = append(resp.Items, item)
	}
	return resp, nil
}

func (s *overviewService) Activity(ctx context.Context, projectKey string, callerID uint, scope string) (*dto.OverviewActivityResponse, error) {
	if scope == "" {
		scope = overviewScopeAll
	}
	if scope != overviewScopeAll && scope != overviewScopeSprint {
		return nil, fmt.Errorf("%w: scope must be %q or %q", domain.ErrInvalidInput, overviewScopeAll, overviewScopeSprint)
	}
	p, err := s.project(ctx, projectKey, callerID)
	if err != nil {
		return nil, err
	}
	sprint, err := s.sprintRepo.FindActiveByProject(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	resp := &dto.OverviewActivityResponse{SprintActive: sprint != nil, Items: []dto.RecentActivityResponse{}}

	var sprintID *uint
	if scope == overviewScopeSprint {
		if sprint == nil {
			return resp, nil
		}
		sprintID = &sprint.ID
	}
	events, err := s.activityRepo.FindRecentByProject(ctx, p.ID, s.now().Add(-overviewActivityWindow), sprintID, overviewListLimit)
	if err != nil {
		return nil, err
	}

	people := s.personLookup(ctx)
	issues := map[uint]*domain.Issue{}
	for _, e := range events {
		issue, ok := issues[e.IssueID]
		if !ok {
			// The repository joined on issues, so the issue exists; a failure here
			// only means a title is blank, never a failed overview.
			issue, _ = s.issueRepo.FindByID(ctx, e.IssueID)
			issues[e.IssueID] = issue
		}
		item := dto.RecentActivityResponse{
			ID: e.ID, Kind: e.Kind, Actor: people(e.ActorID), IssueID: e.IssueID, CreatedAt: e.CreatedAt,
		}
		if issue != nil {
			item.IssueKey, item.IssueTitle = issue.Key, issue.Title
		}
		if e.Kind == domain.ActivityStatusChanged {
			item.NewStatus = e.NewValue
		}
		resp.Items = append(resp.Items, item)
	}
	return resp, nil
}

// personLookup returns a function that resolves user ids to names, asking the
// repository once per distinct user. A user that cannot be found (removed) gets an
// empty name; the entry is still shown.
func (s *overviewService) personLookup(ctx context.Context) func(id uint) dto.PersonRef {
	cache := map[uint]dto.PersonRef{}
	return func(id uint) dto.PersonRef {
		if p, ok := cache[id]; ok {
			return p
		}
		p := dto.PersonRef{ID: id}
		if u, err := s.userRepo.FindByID(ctx, id); err == nil {
			p.Name, p.AvatarURL = u.Name, u.AvatarURL
		}
		cache[id] = p
		return p
	}
}
