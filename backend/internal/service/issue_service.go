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

var validStatuses = map[string]bool{
	domain.IssueStatusBacklog:    true,
	domain.IssueStatusTodo:       true,
	domain.IssueStatusInProgress: true,
	domain.IssueStatusInReview:   true,
	domain.IssueStatusDone:       true,
}

func (s *issueService) Create(ctx context.Context, projectKey string, callerID uint, req dto.CreateIssueRequest) (*dto.IssueResponse, error) {
	project, err := s.projectRepo.FindByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}
	if err := s.requireMember(ctx, project.ID, callerID); err != nil {
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
		AssigneeID:  req.AssigneeID,
		ReporterID:  callerID,
		StoryPoints: req.StoryPoints,
	}

	if err := s.issueRepo.Create(ctx, issue); err != nil {
		return nil, err
	}
	return s.enrichIssueResponse(ctx, toIssueResponse(issue), issue), nil
}

func (s *issueService) ListByProject(ctx context.Context, projectKey string, callerID uint, q dto.IssueListQuery) ([]*dto.IssueResponse, error) {
	project, err := s.projectRepo.FindByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}
	if err := s.requireMember(ctx, project.ID, callerID); err != nil {
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
	project, err := s.projectRepo.FindByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}
	if err := s.requireMember(ctx, project.ID, callerID); err != nil {
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
	project, err := s.projectRepo.FindByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}
	if err := s.requireMember(ctx, project.ID, callerID); err != nil {
		return nil, err
	}

	issue, err := s.issueRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if issue.ProjectID != project.ID {
		return nil, domain.ErrNotFound
	}

	if req.Status != nil && !validStatuses[*req.Status] {
		return nil, fmt.Errorf("invalid status: %s", *req.Status)
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
		issue.AssigneeID = req.AssigneeID
	}
	if req.StoryPoints != nil {
		issue.StoryPoints = req.StoryPoints
	}
	if req.SprintID != nil {
		if *req.SprintID == 0 {
			issue.SprintID = nil
		} else {
			issue.SprintID = req.SprintID
		}
	}

	if err := s.issueRepo.Update(ctx, issue); err != nil {
		return nil, err
	}

	s.recordFieldChanges(ctx, issue.ID, callerID, oldStatus, oldPriority, oldAssigneeID, oldSprintID, oldPoints, issue)

	return s.enrichIssueResponse(ctx, toIssueResponse(issue), issue), nil
}

func (s *issueService) Delete(ctx context.Context, projectKey string, id uint, callerID uint) error {
	project, err := s.projectRepo.FindByKey(ctx, projectKey)
	if err != nil {
		return err
	}
	if err := s.requireMember(ctx, project.ID, callerID); err != nil {
		return err
	}

	issue, err := s.issueRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if issue.ProjectID != project.ID {
		return domain.ErrNotFound
	}

	// Reporter can always delete their own issue; project admin can delete any.
	if issue.ReporterID != callerID {
		membership, err := s.memberRepo.FindByProjectAndUser(ctx, project.ID, callerID)
		if err != nil || membership.Role != "admin" {
			return domain.ErrForbidden
		}
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

	return s.issueRepo.Delete(ctx, id)
}

// GetMyIssues returns all issues assigned to callerID across all projects.
// If q.Status is non-empty only issues with that status are returned.
func (s *issueService) GetMyIssues(ctx context.Context, callerID uint, q dto.IssueListQuery) ([]*dto.IssueResponse, error) {
	issues, err := s.issueRepo.FindByAssignee(ctx, callerID)
	if err != nil {
		return nil, err
	}

	var resp []*dto.IssueResponse
	for _, issue := range issues {
		if q.Status != "" && issue.Status != q.Status {
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

func (s *issueService) requireMember(ctx context.Context, projectID, userID uint) error {
	if _, err := s.memberRepo.FindByProjectAndUser(ctx, projectID, userID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrForbidden
		}
		return err
	}
	return nil
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
