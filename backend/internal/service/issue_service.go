package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
)

// IssueService defines the issues business-logic contract.
type IssueService interface {
	Create(ctx context.Context, projectKey string, callerID uint, req dto.CreateIssueRequest) (*dto.IssueResponse, error)
	ListByProject(ctx context.Context, projectKey string, callerID uint, q dto.IssueListQuery) ([]*dto.IssueResponse, error)
	// GetMyIssues returns all issues assigned to callerID across all projects,
	// optionally filtered by status via q.Status.
	GetMyIssues(ctx context.Context, callerID uint, q dto.IssueListQuery) ([]*dto.IssueResponse, error)
	FindByID(ctx context.Context, projectKey string, id uint, callerID uint) (*dto.IssueResponse, error)
	Update(ctx context.Context, projectKey string, id uint, callerID uint, req dto.UpdateIssueRequest) (*dto.IssueResponse, error)
	Delete(ctx context.Context, projectKey string, id uint, callerID uint) error
}

type issueService struct {
	issueRepo   domain.IssueRepository
	projectRepo domain.ProjectRepository
	memberRepo  domain.ProjectMemberRepository
	activitySvc ActivityService
	userRepo    domain.UserRepository
	sprintRepo  domain.SprintRepository

	// attachmentRepo/attachmentStorage are optional (nil unless WithAttachments
	// is called). When set: enrichIssueResponse populates AttachmentCount, and
	// Delete cascades — removing the issue's attachments (S3 objects then DB
	// rows, research.md Decision 9) before deleting the issue itself.
	attachmentRepo    domain.AttachmentRepository
	attachmentStorage AttachmentStorage

	// notifier is optional (nil unless WithNotifier is called).
	notifier NotificationService

	access Access
}

// NewIssueService returns an IssueService backed by the given repositories.
// The concrete *issueService type is returned (not the IssueService
// interface) so callers may optionally chain WithAttachments(...); it still
// satisfies IssueService for existing callers that only need the interface.
func NewIssueService(
	issueRepo domain.IssueRepository,
	projectRepo domain.ProjectRepository,
	memberRepo domain.ProjectMemberRepository,
	activitySvc ActivityService,
	userRepo domain.UserRepository,
	sprintRepo domain.SprintRepository,
) *issueService {
	return &issueService{
		issueRepo:   issueRepo,
		projectRepo: projectRepo,
		memberRepo:  memberRepo,
		activitySvc: activitySvc,
		userRepo:    userRepo,
		sprintRepo:  sprintRepo,
		access:      NewAccess(projectRepo, memberRepo, userRepo),
	}
}

// WithAttachments attaches attachment support (count enrichment on read,
// cascade delete on issue removal) — feature 009, added after issues
// already existed without any notion of attachments.
func (s *issueService) WithAttachments(repo domain.AttachmentRepository, storage AttachmentStorage) *issueService {
	s.attachmentRepo = repo
	s.attachmentStorage = storage
	return s
}

// WithNotifier enables assignment notifications (feature 017). Optional: without
// it, assignment changes simply notify nobody.
func (s *issueService) WithNotifier(n NotificationService) *issueService {
	s.notifier = n
	return s
}

var validStatuses = map[string]bool{
	domain.IssueStatusBacklog:    true,
	domain.IssueStatusTodo:       true,
	domain.IssueStatusInProgress: true,
	domain.IssueStatusInReview:   true,
	domain.IssueStatusDone:       true,
}

func (s *issueService) Create(ctx context.Context, projectKey string, callerID uint, req dto.CreateIssueRequest) (*dto.IssueResponse, error) {
	project, _, err := s.access.RequireProjectByKey(ctx, projectKey, callerID, domain.CapTaskWrite)
	if err != nil {
		return nil, err
	}

	if err := s.requireAssignable(ctx, project.ID, req.AssigneeID); err != nil {
		return nil, err
	}
	sprintID, err := s.resolveNewSprint(ctx, project.ID, req.SprintID)
	if err != nil {
		return nil, err
	}

	// Sequential key: count existing issues for this project.
	existing, err := s.issueRepo.FindByProjectID(ctx, project.ID)
	if err != nil {
		return nil, err
	}

	issue := &domain.Issue{
		Key:         fmt.Sprintf("%s-%d", project.Key, len(existing)+1),
		ProjectID:   project.ID,
		Title:       req.Title,
		Description: req.Description,
		Type:        req.Type,
		Status:      domain.IssueStatusTodo,
		Priority:    req.Priority,
		AssigneeID:  assigneeOrNil(req.AssigneeID),
		ReporterID:  callerID,
		StoryPoints: req.StoryPoints,
		SprintID:    sprintID,
	}

	if err := s.issueRepo.Create(ctx, issue); err != nil {
		return nil, err
	}
	if s.notifier != nil && issue.AssigneeID != nil {
		s.notifier.NotifyAssignmentChange(ctx, project.ID, issue.ID, callerID, nil, issue.AssigneeID)
	}
	return s.enrichIssueResponse(ctx, toIssueResponse(issue), issue), nil
}

func (s *issueService) ListByProject(ctx context.Context, projectKey string, callerID uint, q dto.IssueListQuery) ([]*dto.IssueResponse, error) {
	project, _, err := s.access.RequireProjectByKey(ctx, projectKey, callerID, domain.CapProjectView)
	if err != nil {
		return nil, err
	}

	all, err := s.issueRepo.FindByProjectID(ctx, project.ID)
	if err != nil {
		return nil, err
	}

	// Filter in-memory — acceptable for MVP scale.
	var filtered []*domain.Issue

	// Resolve label filter IDs once (avoid repeated repo calls).
	labelIssueSet := make(map[uint]bool)
	if q.LabelID != 0 {
		ids, err := s.issueRepo.FindIssueIDsByLabelID(ctx, q.LabelID)
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			labelIssueSet[id] = true
		}
	}

	searchQ := strings.ToLower(q.Q)

	for _, i := range all {
		if q.Type != "" && i.Type != q.Type {
			continue
		}
		if q.Status != "" && i.Status != q.Status {
			continue
		}
		if q.Priority != "" && i.Priority != q.Priority {
			continue
		}
		if q.AssigneeID != 0 && (i.AssigneeID == nil || *i.AssigneeID != q.AssigneeID) {
			continue
		}
		if q.LabelID != 0 && !labelIssueSet[i.ID] {
			continue
		}
		if searchQ != "" {
			titleMatch := strings.Contains(strings.ToLower(i.Title), searchQ)
			descMatch := strings.Contains(strings.ToLower(i.Description), searchQ)
			if !titleMatch && !descMatch {
				continue
			}
		}
		filtered = append(filtered, i)
	}

	// Pagination defaults.
	page, limit := q.Page, q.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	start := (page - 1) * limit
	if start >= len(filtered) {
		return []*dto.IssueResponse{}, nil
	}
	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	result := make([]*dto.IssueResponse, 0, end-start)
	for _, i := range filtered[start:end] {
		result = append(result, s.enrichIssueResponse(ctx, toIssueResponse(i), i))
	}
	return result, nil
}

func (s *issueService) FindByID(ctx context.Context, projectKey string, id uint, callerID uint) (*dto.IssueResponse, error) {
	project, _, err := s.access.RequireProjectByKey(ctx, projectKey, callerID, domain.CapProjectView)
	if err != nil {
		return nil, err
	}

	issue, err := s.issueRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// Guard against cross-project ID access.
	if issue.ProjectID != project.ID {
		return nil, domain.ErrNotFound
	}
	return s.enrichIssueResponse(ctx, toIssueResponse(issue), issue), nil
}

func (s *issueService) Update(ctx context.Context, projectKey string, id uint, callerID uint, req dto.UpdateIssueRequest) (*dto.IssueResponse, error) {
	project, _, err := s.access.RequireProjectByKey(ctx, projectKey, callerID, domain.CapTaskWrite)
	if err != nil {
		return nil, err
	}

	issue, err := s.issueRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if issue.ProjectID != project.ID {
		return nil, domain.ErrNotFound
	}

	if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
		return nil, domain.NewInputError("title must not be empty")
	}
	if req.Status != nil && !validStatuses[*req.Status] {
		return nil, domain.NewInputError("invalid status: %s", *req.Status)
	}

	// Capture old values before patching so we can record what changed.
	oldStatus := issue.Status
	oldPriority := issue.Priority
	oldAssigneeID := issue.AssigneeID
	oldSprintID := issue.SprintID
	oldPoints := issue.StoryPoints

	if req.Title != nil {
		issue.Title = *req.Title
	}
	if req.Description != nil {
		issue.Description = *req.Description
	}
	if req.Type != nil {
		issue.Type = *req.Type
	}
	if req.Status != nil {
		issue.Status = *req.Status
		if *req.Status == domain.IssueStatusDone && issue.CompletedAt == nil {
			now := time.Now().UTC()
			issue.CompletedAt = &now
		} else if *req.Status != domain.IssueStatusDone {
			issue.CompletedAt = nil
		}
	}
	if req.Priority != nil {
		issue.Priority = *req.Priority
	}
	if req.AssigneeID != nil {
		if err := s.requireAssignable(ctx, project.ID, req.AssigneeID); err != nil {
			return nil, err
		}
		// 0 means "clear" (ADR-036); JSON null is indistinguishable from absent.
		issue.AssigneeID = assigneeOrNil(req.AssigneeID)
	}
	if req.StoryPoints != nil {
		issue.StoryPoints = req.StoryPoints
	}
	if req.SprintID != nil {
		if *req.SprintID == 0 {
			issue.SprintID = nil
		} else {
			// A sprint of another project must not be reachable by guessing its id.
			sprint, err := s.sprintRepo.FindByID(ctx, *req.SprintID)
			if err != nil {
				return nil, err
			}
			if sprint.ProjectID != project.ID {
				return nil, domain.ErrNotFound
			}
			issue.SprintID = req.SprintID
		}
	}

	if err := s.issueRepo.Update(ctx, issue); err != nil {
		return nil, err
	}

	s.recordFieldChanges(ctx, issue.ID, callerID, oldStatus, oldPriority, oldAssigneeID, oldSprintID, oldPoints, issue)

	if s.notifier != nil && ptrUintChanged(oldAssigneeID, issue.AssigneeID) {
		s.notifier.NotifyAssignmentChange(ctx, project.ID, issue.ID, callerID, oldAssigneeID, issue.AssigneeID)
	}

	return s.enrichIssueResponse(ctx, toIssueResponse(issue), issue), nil
}

func (s *issueService) Delete(ctx context.Context, projectKey string, id uint, callerID uint) error {
	project, _, err := s.access.RequireProjectByKey(ctx, projectKey, callerID, domain.CapProjectView)
	if err != nil {
		return err
	}

	issue, err := s.issueRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if issue.ProjectID != project.ID {
		return domain.ErrNotFound
	}

	// The reporter may delete their own task, but only while they can still
	// edit tasks (a reporter demoted to viewer cannot); deleting anyone else's
	// takes the delete-any capability.
	need := domain.CapTaskDeleteAny
	if issue.ReporterID == callerID {
		need = domain.CapTaskWrite
	}
	if _, err := s.access.RequireProject(ctx, project.ID, callerID, need); err != nil {
		return err
	}

	// Cascade attachments only after authorization has passed — never destroy
	// data before confirming the issue delete itself will proceed (research.md
	// Decision 9). S3 objects go before their DB rows, same ordering as a
	// single attachment delete.
	if s.attachmentRepo != nil && s.attachmentStorage != nil {
		attachments, err := s.attachmentRepo.FindByIssueID(ctx, id)
		if err != nil {
			return err
		}
		if len(attachments) > 0 {
			keys := make([]string, len(attachments))
			for i, a := range attachments {
				keys[i] = a.ObjectKey
			}
			if err := s.attachmentStorage.DeleteAll(ctx, keys); err != nil {
				return fmt.Errorf("%w: %v", domain.ErrAttachmentStorageUnavailable, err)
			}
			if err := s.attachmentRepo.DeleteByIssueID(ctx, id); err != nil {
				return err
			}
		}
	}

	if err := s.issueRepo.Delete(ctx, id); err != nil {
		return err
	}

	// The task's row is gone now, so the event carries what names and places it. Like the
	// other activity records, a failure to write it never undoes the delete.
	_ = s.activitySvc.Record(ctx, &domain.ActivityEvent{
		IssueID: id, ActorID: callerID, Kind: domain.ActivityIssueDeleted,
		ProjectID: project.ID, IssueKey: issue.Key, IssueTitle: issue.Title, SprintID: issue.SprintID,
	})
	return nil
}

// resolveNewSprint checks the sprint a new task is to be created in: it must
// belong to the project and still be open for work (planning or active). nil or 0
// means the backlog.
func (s *issueService) resolveNewSprint(ctx context.Context, projectID uint, sprintID *uint) (*uint, error) {
	if sprintID == nil || *sprintID == 0 {
		return nil, nil
	}
	sprint, err := s.sprintRepo.FindByID(ctx, *sprintID)
	if err != nil {
		return nil, err
	}
	if sprint.ProjectID != projectID {
		return nil, domain.ErrNotFound
	}
	if sprint.Status == domain.SprintStatusCompleted {
		return nil, domain.NewInputError("a completed sprint cannot take new tasks")
	}
	id := *sprintID
	return &id, nil
}

// requireAssignable checks that the person a task is being assigned to is a member
// of the project (any role, a viewer included). Without it a task could be handed
// to an outsider, who would then be able to read it through their own task list.
// A nil or zero assignee means "unassigned" and is always fine.
func (s *issueService) requireAssignable(ctx context.Context, projectID uint, assignee *uint) error {
	if assignee == nil || *assignee == 0 {
		return nil
	}
	if _, err := s.access.RequireProject(ctx, projectID, *assignee, domain.CapProjectView); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.NewInputError("assignee must be a member of the project")
		}
		return err
	}
	return nil
}

// GetMyIssues returns all issues assigned to callerID across all projects.
// If q.Status is non-empty only issues with that status are returned.
func (s *issueService) GetMyIssues(ctx context.Context, callerID uint, q dto.IssueListQuery) ([]*dto.IssueResponse, error) {
	issues, err := s.issueRepo.FindByAssignee(ctx, callerID)
	if err != nil {
		return nil, err
	}

	// Only tasks in projects the caller still belongs to: someone who was removed
	// from a project must not keep reading its tasks through their own list.
	member := map[uint]bool{}
	var resp []*dto.IssueResponse
	for _, issue := range issues {
		if q.Status != "" && issue.Status != q.Status {
			continue
		}
		ok, seen := member[issue.ProjectID]
		if !seen {
			_, err := s.access.RequireProject(ctx, issue.ProjectID, callerID, domain.CapProjectView)
			if err != nil && !errors.Is(err, domain.ErrNotFound) {
				return nil, err
			}
			ok = err == nil
			member[issue.ProjectID] = ok
		}
		if !ok {
			continue
		}
		resp = append(resp, s.enrichIssueResponse(ctx, toIssueResponse(issue), issue))
	}
	return resp, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

// enrichIssueResponse populates AssigneeName and AssigneeAvatarURL from userRepo when
// the issue has an assignee.
func (s *issueService) enrichIssueResponse(ctx context.Context, r *dto.IssueResponse, i *domain.Issue) *dto.IssueResponse {
	if s.attachmentRepo != nil {
		if count, err := s.attachmentRepo.CountByIssueID(ctx, i.ID); err == nil {
			r.AttachmentCount = int(count)
		}
	}

	if i.AssigneeID == nil {
		return r
	}
	u, err := s.userRepo.FindByID(ctx, *i.AssigneeID)
	if err != nil {
		return r
	}
	name := u.Name
	r.AssigneeName = &name
	if u.AvatarURL != "" {
		url := u.AvatarURL
		r.AssigneeAvatarURL = &url
	}
	return r
}

func (s *issueService) recordFieldChanges(
	ctx context.Context,
	issueID, actorID uint,
	oldStatus, oldPriority string,
	oldAssigneeID, oldSprintID *uint,
	oldPoints *int,
	issue *domain.Issue,
) {
	if issue.Status != oldStatus {
		_ = s.activitySvc.Record(ctx, &domain.ActivityEvent{
			IssueID: issueID, ActorID: actorID,
			Kind: domain.ActivityStatusChanged, OldValue: oldStatus, NewValue: issue.Status,
		})
	}
	if issue.Priority != oldPriority {
		_ = s.activitySvc.Record(ctx, &domain.ActivityEvent{
			IssueID: issueID, ActorID: actorID,
			Kind: domain.ActivityPriorityChanged, OldValue: oldPriority, NewValue: issue.Priority,
		})
	}
	if ptrUintChanged(oldAssigneeID, issue.AssigneeID) {
		_ = s.activitySvc.Record(ctx, &domain.ActivityEvent{
			IssueID: issueID, ActorID: actorID,
			Kind:     domain.ActivityAssigneeChanged,
			OldValue: s.resolveUserName(ctx, oldAssigneeID),
			NewValue: s.resolveUserName(ctx, issue.AssigneeID),
		})
	}
	if ptrUintChanged(oldSprintID, issue.SprintID) {
		_ = s.activitySvc.Record(ctx, &domain.ActivityEvent{
			IssueID: issueID, ActorID: actorID,
			Kind:     domain.ActivitySprintChanged,
			OldValue: s.resolveSprintName(ctx, oldSprintID),
			NewValue: s.resolveSprintName(ctx, issue.SprintID),
		})
	}
	if ptrIntChanged(oldPoints, issue.StoryPoints) {
		_ = s.activitySvc.Record(ctx, &domain.ActivityEvent{
			IssueID: issueID, ActorID: actorID,
			Kind:     domain.ActivityStoryPointsChanged,
			OldValue: pointsLabel(oldPoints),
			NewValue: pointsLabel(issue.StoryPoints),
		})
	}
}

func (s *issueService) resolveUserName(ctx context.Context, userID *uint) string {
	if userID == nil {
		return "unassigned"
	}
	u, err := s.userRepo.FindByID(ctx, *userID)
	if err != nil {
		return "unknown"
	}
	return u.Name
}

// assigneeOrNil maps the API sentinel 0 ("no assignee") to nil.
func assigneeOrNil(id *uint) *uint {
	if id == nil || *id == 0 {
		return nil
	}
	return id
}

func ptrUintChanged(a, b *uint) bool {
	if a == nil && b == nil {
		return false
	}
	if a == nil || b == nil {
		return true
	}
	return *a != *b
}

func ptrIntChanged(a, b *int) bool {
	if a == nil && b == nil {
		return false
	}
	if a == nil || b == nil {
		return true
	}
	return *a != *b
}

func (s *issueService) resolveSprintName(ctx context.Context, id *uint) string {
	if id == nil {
		return "backlog"
	}
	sprint, err := s.sprintRepo.FindByID(ctx, *id)
	if err != nil {
		return fmt.Sprintf("sprint %d", *id)
	}
	return sprint.Name
}

func pointsLabel(pts *int) string {
	if pts == nil {
		return "none"
	}
	return fmt.Sprintf("%d", *pts)
}

func toIssueResponse(i *domain.Issue) *dto.IssueResponse {
	return &dto.IssueResponse{
		ID:          i.ID,
		Key:         i.Key,
		ProjectID:   i.ProjectID,
		Title:       i.Title,
		Description: i.Description,
		Type:        i.Type,
		Status:      i.Status,
		Priority:    i.Priority,
		AssigneeID:  i.AssigneeID,
		ReporterID:  i.ReporterID,
		SprintID:    i.SprintID,
		StoryPoints: i.StoryPoints,
		CreatedAt:   i.CreatedAt,
		CompletedAt: i.CompletedAt,
	}
}
