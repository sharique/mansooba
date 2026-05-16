package service

import (
	"context"
	"time"

	"github.com/sharique/jira-go/internal/domain"
	"github.com/sharique/jira-go/internal/dto"
)

// SprintService defines all business operations for sprints, backlog, and burndown.
type SprintService interface {
	Create(ctx context.Context, projectKey string, callerID uint, req dto.CreateSprintRequest) (*dto.SprintResponse, error)
	List(ctx context.Context, projectKey string, callerID uint) ([]*dto.SprintResponse, error)
	Get(ctx context.Context, projectKey string, id uint, callerID uint) (*dto.SprintResponse, error)
	Update(ctx context.Context, projectKey string, id uint, callerID uint, req dto.UpdateSprintRequest) (*dto.SprintResponse, error)
	Delete(ctx context.Context, projectKey string, id uint, callerID uint) error
	Start(ctx context.Context, projectKey string, id uint, callerID uint) (*dto.SprintResponse, error)
	Complete(ctx context.Context, projectKey string, id uint, callerID uint, req dto.CompleteSprintRequest) (*dto.SprintResponse, error)
	Backlog(ctx context.Context, projectKey string, callerID uint) ([]*domain.Issue, error)
	Burndown(ctx context.Context, projectKey string, id uint, callerID uint) (*dto.BurndownResponse, error)
}

type sprintService struct {
	sprintRepo  domain.SprintRepository
	issueRepo   domain.IssueRepository
	projectRepo domain.ProjectRepository
	memberRepo  domain.ProjectMemberRepository
}

// NewSprintService constructs a SprintService backed by the given repositories.
func NewSprintService(
	sprintRepo domain.SprintRepository,
	issueRepo domain.IssueRepository,
	projectRepo domain.ProjectRepository,
	memberRepo domain.ProjectMemberRepository,
) SprintService {
	return &sprintService{
		sprintRepo:  sprintRepo,
		issueRepo:   issueRepo,
		projectRepo: projectRepo,
		memberRepo:  memberRepo,
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (s *sprintService) resolveProject(ctx context.Context, key string) (*domain.Project, error) {
	p, err := s.projectRepo.FindByKey(ctx, key)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	return p, nil
}

func (s *sprintService) requireMember(ctx context.Context, projectID, callerID uint) error {
	if _, err := s.memberRepo.FindByProjectAndUser(ctx, projectID, callerID); err != nil {
		return domain.ErrForbidden
	}
	return nil
}

func (s *sprintService) requireAdminOrOwner(ctx context.Context, project *domain.Project, callerID uint) error {
	membership, err := s.memberRepo.FindByProjectAndUser(ctx, project.ID, callerID)
	if err != nil {
		return domain.ErrForbidden
	}
	if membership.Role != "admin" && project.OwnerID != callerID {
		return domain.ErrForbidden
	}
	return nil
}

func (s *sprintService) resolveSprint(ctx context.Context, id, projectID uint) (*domain.Sprint, error) {
	sprint, err := s.sprintRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if sprint.ProjectID != projectID {
		return nil, domain.ErrNotFound
	}
	return sprint, nil
}

func toSprintResponse(s *domain.Sprint) *dto.SprintResponse {
	return &dto.SprintResponse{
		ID:        s.ID,
		ProjectID: s.ProjectID,
		Name:      s.Name,
		Goal:      s.Goal,
		Status:    s.Status,
		StartDate: s.StartDate,
		EndDate:   s.EndDate,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

// ── CRUD ─────────────────────────────────────────────────────────────────────

func (s *sprintService) Create(ctx context.Context, projectKey string, callerID uint, req dto.CreateSprintRequest) (*dto.SprintResponse, error) {
	p, err := s.resolveProject(ctx, projectKey)
	if err != nil {
		return nil, err
	}
	if err := s.requireAdminOrOwner(ctx, p, callerID); err != nil {
		return nil, err
	}
	sprint := &domain.Sprint{
		ProjectID: p.ID,
		Name:      req.Name,
		Goal:      req.Goal,
		Status:    domain.SprintStatusPlanning,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
	}
	if err := s.sprintRepo.Create(ctx, sprint); err != nil {
		return nil, err
	}
	return toSprintResponse(sprint), nil
}

func (s *sprintService) List(ctx context.Context, projectKey string, callerID uint) ([]*dto.SprintResponse, error) {
	p, err := s.resolveProject(ctx, projectKey)
	if err != nil {
		return nil, err
	}
	if err := s.requireMember(ctx, p.ID, callerID); err != nil {
		return nil, err
	}
	sprints, err := s.sprintRepo.FindByProject(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	resp := make([]*dto.SprintResponse, len(sprints))
	for i, sp := range sprints {
		resp[i] = toSprintResponse(sp)
	}
	return resp, nil
}

func (s *sprintService) Get(ctx context.Context, projectKey string, id uint, callerID uint) (*dto.SprintResponse, error) {
	p, err := s.resolveProject(ctx, projectKey)
	if err != nil {
		return nil, err
	}
	if err := s.requireMember(ctx, p.ID, callerID); err != nil {
		return nil, err
	}
	sprint, err := s.resolveSprint(ctx, id, p.ID)
	if err != nil {
		return nil, err
	}
	return toSprintResponse(sprint), nil
}

func (s *sprintService) Update(ctx context.Context, projectKey string, id uint, callerID uint, req dto.UpdateSprintRequest) (*dto.SprintResponse, error) {
	p, err := s.resolveProject(ctx, projectKey)
	if err != nil {
		return nil, err
	}
	if err := s.requireAdminOrOwner(ctx, p, callerID); err != nil {
		return nil, err
	}
	sprint, err := s.resolveSprint(ctx, id, p.ID)
	if err != nil {
		return nil, err
	}
	if sprint.Status == domain.SprintStatusCompleted {
		return nil, domain.ErrSprintNotEditable
	}
	if req.Name != nil {
		sprint.Name = *req.Name
	}
	if req.Goal != nil {
		sprint.Goal = *req.Goal
	}
	if req.StartDate != nil {
		sprint.StartDate = req.StartDate
	}
	if req.EndDate != nil {
		sprint.EndDate = req.EndDate
	}
	if err := s.sprintRepo.Update(ctx, sprint); err != nil {
		return nil, err
	}
	return toSprintResponse(sprint), nil
}

func (s *sprintService) Delete(ctx context.Context, projectKey string, id uint, callerID uint) error {
	p, err := s.resolveProject(ctx, projectKey)
	if err != nil {
		return err
	}
	if err := s.requireAdminOrOwner(ctx, p, callerID); err != nil {
		return err
	}
	sprint, err := s.resolveSprint(ctx, id, p.ID)
	if err != nil {
		return err
	}
	if sprint.Status != domain.SprintStatusPlanning {
		return domain.ErrSprintNotDeletable
	}
	return s.sprintRepo.Delete(ctx, id)
}

// ── Lifecycle ─────────────────────────────────────────────────────────────────

// Start transitions a sprint from Planning to Active.
// Enforces that only one sprint per project can be Active at a time.
// Sets StartDate to now if not already provided.
func (s *sprintService) Start(ctx context.Context, projectKey string, id uint, callerID uint) (*dto.SprintResponse, error) {
	p, err := s.resolveProject(ctx, projectKey)
	if err != nil {
		return nil, err
	}
	if err := s.requireAdminOrOwner(ctx, p, callerID); err != nil {
		return nil, err
	}
	sprint, err := s.resolveSprint(ctx, id, p.ID)
	if err != nil {
		return nil, err
	}
	if sprint.Status != domain.SprintStatusPlanning {
		return nil, domain.ErrSprintInvalidTransition
	}
	active, err := s.sprintRepo.FindActiveByProject(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	if active != nil {
		return nil, domain.ErrSprintAlreadyActive
	}
	now := time.Now()
	sprint.Status = domain.SprintStatusActive
	if sprint.StartDate == nil {
		sprint.StartDate = &now
	}
	if err := s.sprintRepo.Update(ctx, sprint); err != nil {
		return nil, err
	}
	return toSprintResponse(sprint), nil
}

// Complete transitions an Active sprint to Completed.
// Issues with status != "done" are unfinished and get moved to req.NextSprintID,
// or to the backlog (sprint_id = NULL) if NextSprintID is nil.
// The sprint update and issue migration execute in a single DB transaction.
func (s *sprintService) Complete(ctx context.Context, projectKey string, id uint, callerID uint, req dto.CompleteSprintRequest) (*dto.SprintResponse, error) {
	p, err := s.resolveProject(ctx, projectKey)
	if err != nil {
		return nil, err
	}
	if err := s.requireAdminOrOwner(ctx, p, callerID); err != nil {
		return nil, err
	}

	// Single fetch — owns the sprint and loads its issues atomically.
	sprint, err := s.sprintRepo.FindWithIssues(ctx, id)
	if err != nil {
		return nil, err
	}
	if sprint.ProjectID != p.ID {
		return nil, domain.ErrNotFound
	}
	if sprint.Status != domain.SprintStatusActive {
		return nil, domain.ErrSprintInvalidTransition
	}

	// If a next sprint is provided, verify it belongs to the same project and is still in planning.
	if req.NextSprintID != nil {
		next, err := s.sprintRepo.FindByID(ctx, *req.NextSprintID)
		if err != nil || next.ProjectID != p.ID {
			return nil, domain.ErrNotFound
		}
		if next.Status != domain.SprintStatusPlanning {
			return nil, domain.ErrSprintInvalidTransition
		}
	}

	// Collect IDs of issues that are not done.
	var unfinishedIDs []uint
	for _, issue := range sprint.Issues {
		if issue.Status != "done" {
			unfinishedIDs = append(unfinishedIDs, issue.ID)
		}
	}

	now := time.Now()
	sprint.Status = domain.SprintStatusCompleted
	if sprint.EndDate == nil {
		sprint.EndDate = &now
	}

	if err := s.sprintRepo.CompleteWithMigration(ctx, sprint, unfinishedIDs, req.NextSprintID); err != nil {
		return nil, err
	}
	return toSprintResponse(sprint), nil
}

func (s *sprintService) Backlog(ctx context.Context, projectKey string, callerID uint) ([]*domain.Issue, error) {
	panic("Backlog implemented in Task 16")
}

func (s *sprintService) Burndown(ctx context.Context, projectKey string, id uint, callerID uint) (*dto.BurndownResponse, error) {
	panic("Burndown implemented in Task 16")
}
