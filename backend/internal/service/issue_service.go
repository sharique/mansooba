package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/sharique/jira-go/internal/domain"
	"github.com/sharique/jira-go/internal/dto"
)

// IssueService defines the issues business-logic contract.
type IssueService interface {
	Create(ctx context.Context, projectKey string, callerID uint, req dto.CreateIssueRequest) (*dto.IssueResponse, error)
	ListByProject(ctx context.Context, projectKey string, callerID uint, q dto.IssueListQuery) ([]*dto.IssueResponse, error)
	FindByID(ctx context.Context, projectKey string, id uint, callerID uint) (*dto.IssueResponse, error)
	Update(ctx context.Context, projectKey string, id uint, callerID uint, req dto.UpdateIssueRequest) (*dto.IssueResponse, error)
	Delete(ctx context.Context, projectKey string, id uint, callerID uint) error
}

type issueService struct {
	issueRepo   domain.IssueRepository
	projectRepo domain.ProjectRepository
	memberRepo  domain.ProjectMemberRepository
}

// NewIssueService returns an IssueService backed by the given repositories.
func NewIssueService(
	issueRepo domain.IssueRepository,
	projectRepo domain.ProjectRepository,
	memberRepo domain.ProjectMemberRepository,
) IssueService {
	return &issueService{
		issueRepo:   issueRepo,
		projectRepo: projectRepo,
		memberRepo:  memberRepo,
	}
}

var validStatuses = map[string]bool{
	"backlog":     true,
	"todo":        true,
	"in_progress": true,
	"in_review":   true,
	"done":        true,
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
		Status:      "todo",
		Priority:    req.Priority,
		AssigneeID:  req.AssigneeID,
		ReporterID:  callerID,
	}

	if err := s.issueRepo.Create(ctx, issue); err != nil {
		return nil, err
	}
	return toIssueResponse(issue), nil
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
	for _, i := range all {
		if q.Type != "" && i.Type != q.Type {
			continue
		}
		if q.Status != "" && i.Status != q.Status {
			continue
		}
		if q.AssigneeID != 0 && (i.AssigneeID == nil || *i.AssigneeID != q.AssigneeID) {
			continue
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
		result = append(result, toIssueResponse(i))
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
	return toIssueResponse(issue), nil
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
	}
	if req.Priority != nil {
		issue.Priority = *req.Priority
	}
	if req.AssigneeID != nil {
		issue.AssigneeID = req.AssigneeID
	}

	if err := s.issueRepo.Update(ctx, issue); err != nil {
		return nil, err
	}
	return toIssueResponse(issue), nil
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

	return s.issueRepo.Delete(ctx, id)
}

// ── helpers ───────────────────────────────────────────────────────────────────

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
	}
}
