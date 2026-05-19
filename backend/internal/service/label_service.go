package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/sharique/jira-go/internal/domain"
	"github.com/sharique/jira-go/internal/dto"
)

// LabelService manages project labels and their attachment to issues.
type LabelService interface {
	Create(ctx context.Context, projectKey string, callerID uint, req dto.CreateLabelRequest) (*dto.LabelResponse, error)
	ListByProject(ctx context.Context, projectKey string, callerID uint) ([]*dto.LabelResponse, error)
	Delete(ctx context.Context, projectKey string, labelID, callerID uint) error
	AttachToIssue(ctx context.Context, issueID, labelID, callerID uint) error
	DetachFromIssue(ctx context.Context, issueID, labelID, callerID uint) error
}

type labelService struct {
	labelRepo   domain.LabelRepository
	issueRepo   domain.IssueRepository
	projectRepo domain.ProjectRepository
	memberRepo  domain.ProjectMemberRepository
	activitySvc ActivityService
}

func NewLabelService(
	labelRepo domain.LabelRepository,
	issueRepo domain.IssueRepository,
	projectRepo domain.ProjectRepository,
	memberRepo domain.ProjectMemberRepository,
	activitySvc ActivityService,
) LabelService {
	return &labelService{
		labelRepo:   labelRepo,
		issueRepo:   issueRepo,
		projectRepo: projectRepo,
		memberRepo:  memberRepo,
		activitySvc: activitySvc,
	}
}

func (s *labelService) Create(ctx context.Context, projectKey string, callerID uint, req dto.CreateLabelRequest) (*dto.LabelResponse, error) {
	if !domain.LabelPalette[req.Color] {
		return nil, fmt.Errorf("color %q is not in the allowed palette", req.Color)
	}
	project, err := s.projectRepo.FindByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}
	if err := s.requireMember(ctx, project.ID, callerID); err != nil {
		return nil, err
	}
	label := &domain.Label{ProjectID: project.ID, Name: req.Name, Color: req.Color}
	if err := s.labelRepo.Create(ctx, label); err != nil {
		return nil, err
	}
	return toLabelResponse(label), nil
}

func (s *labelService) ListByProject(ctx context.Context, projectKey string, callerID uint) ([]*dto.LabelResponse, error) {
	project, err := s.projectRepo.FindByKey(ctx, projectKey)
	if err != nil {
		return nil, err
	}
	if err := s.requireMember(ctx, project.ID, callerID); err != nil {
		return nil, err
	}
	labels, err := s.labelRepo.FindByProjectID(ctx, project.ID)
	if err != nil {
		return nil, err
	}
	result := make([]*dto.LabelResponse, 0, len(labels))
	for _, l := range labels {
		result = append(result, toLabelResponse(l))
	}
	return result, nil
}

func (s *labelService) Delete(ctx context.Context, projectKey string, labelID, callerID uint) error {
	project, err := s.projectRepo.FindByKey(ctx, projectKey)
	if err != nil {
		return err
	}
	if err := s.requireMember(ctx, project.ID, callerID); err != nil {
		return err
	}
	label, err := s.labelRepo.FindByID(ctx, labelID)
	if err != nil {
		return err
	}
	if label.ProjectID != project.ID {
		return domain.ErrNotFound
	}
	return s.labelRepo.Delete(ctx, labelID)
}

func (s *labelService) AttachToIssue(ctx context.Context, issueID, labelID, callerID uint) error {
	issue, err := s.issueRepo.FindByID(ctx, issueID)
	if err != nil {
		return err
	}
	if err := s.requireMember(ctx, issue.ProjectID, callerID); err != nil {
		return err
	}
	label, err := s.labelRepo.FindByID(ctx, labelID)
	if err != nil {
		return err
	}
	if label.ProjectID != issue.ProjectID {
		return domain.ErrNotFound // treat as not-found to avoid leaking label existence
	}
	if err := s.labelRepo.AttachToIssue(ctx, issueID, labelID); err != nil {
		return err
	}
	_ = s.activitySvc.Record(ctx, &domain.ActivityEvent{
		IssueID: issueID, ActorID: callerID,
		Kind: domain.ActivityLabelAdded, NewValue: label.Name,
	})
	return nil
}

func (s *labelService) DetachFromIssue(ctx context.Context, issueID, labelID, callerID uint) error {
	issue, err := s.issueRepo.FindByID(ctx, issueID)
	if err != nil {
		return err
	}
	if err := s.requireMember(ctx, issue.ProjectID, callerID); err != nil {
		return err
	}
	label, err := s.labelRepo.FindByID(ctx, labelID)
	if err != nil {
		return err
	}
	if label.ProjectID != issue.ProjectID {
		return domain.ErrNotFound // treat as not-found to avoid leaking label existence
	}
	if err := s.labelRepo.DetachFromIssue(ctx, issueID, labelID); err != nil {
		return err
	}
	_ = s.activitySvc.Record(ctx, &domain.ActivityEvent{
		IssueID: issueID, ActorID: callerID,
		Kind: domain.ActivityLabelRemoved, OldValue: label.Name,
	})
	return nil
}

func (s *labelService) requireMember(ctx context.Context, projectID, userID uint) error {
	if _, err := s.memberRepo.FindByProjectAndUser(ctx, projectID, userID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrForbidden
		}
		return err
	}
	return nil
}

func toLabelResponse(l *domain.Label) *dto.LabelResponse {
	return &dto.LabelResponse{
		ID: l.ID, ProjectID: l.ProjectID,
		Name: l.Name, Color: l.Color, CreatedAt: l.CreatedAt,
	}
}
