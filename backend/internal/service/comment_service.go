package service

import (
	"context"
	"errors"

	"github.com/sharique/jira-go/internal/domain"
	"github.com/sharique/jira-go/internal/dto"
)

// CommentService manages comments on issues.
type CommentService interface {
	Create(ctx context.Context, issueID, callerID uint, req dto.CreateCommentRequest) (*dto.CommentResponse, error)
	List(ctx context.Context, issueID, callerID uint) ([]*dto.CommentResponse, error)
	Update(ctx context.Context, commentID, callerID uint, req dto.UpdateCommentRequest) (*dto.CommentResponse, error)
	Delete(ctx context.Context, commentID, callerID uint) error
}

type commentService struct {
	commentRepo domain.CommentRepository
	issueRepo   domain.IssueRepository
	memberRepo  domain.ProjectMemberRepository
	activitySvc ActivityService
}

func NewCommentService(
	commentRepo domain.CommentRepository,
	issueRepo domain.IssueRepository,
	memberRepo domain.ProjectMemberRepository,
	activitySvc ActivityService,
) CommentService {
	return &commentService{
		commentRepo: commentRepo,
		issueRepo:   issueRepo,
		memberRepo:  memberRepo,
		activitySvc: activitySvc,
	}
}

func (s *commentService) Create(ctx context.Context, issueID, callerID uint, req dto.CreateCommentRequest) (*dto.CommentResponse, error) {
	issue, err := s.issueRepo.FindByID(ctx, issueID)
	if err != nil {
		return nil, err
	}
	if err := s.requireMemberOfProject(ctx, issue.ProjectID, callerID); err != nil {
		return nil, err
	}

	comment := &domain.Comment{
		IssueID:  issueID,
		AuthorID: callerID,
		Body:     req.Body,
	}
	if err := s.commentRepo.Create(ctx, comment); err != nil {
		return nil, err
	}

	_ = s.activitySvc.Record(ctx, &domain.ActivityEvent{
		IssueID: issueID,
		ActorID: callerID,
		Kind:    domain.ActivityCommentAdded,
	})

	return toCommentResponse(comment), nil
}

func (s *commentService) List(ctx context.Context, issueID, callerID uint) ([]*dto.CommentResponse, error) {
	issue, err := s.issueRepo.FindByID(ctx, issueID)
	if err != nil {
		return nil, err
	}
	if err := s.requireMemberOfProject(ctx, issue.ProjectID, callerID); err != nil {
		return nil, err
	}

	comments, err := s.commentRepo.FindByIssueID(ctx, issueID)
	if err != nil {
		return nil, err
	}
	result := make([]*dto.CommentResponse, 0, len(comments))
	for _, c := range comments {
		result = append(result, toCommentResponse(c))
	}
	return result, nil
}

func (s *commentService) Update(ctx context.Context, commentID, callerID uint, req dto.UpdateCommentRequest) (*dto.CommentResponse, error) {
	comment, err := s.commentRepo.FindByID(ctx, commentID)
	if err != nil {
		return nil, err
	}
	if comment.AuthorID != callerID {
		return nil, domain.ErrForbidden
	}
	comment.Body = req.Body
	if err := s.commentRepo.Update(ctx, comment); err != nil {
		return nil, err
	}
	return toCommentResponse(comment), nil
}

func (s *commentService) Delete(ctx context.Context, commentID, callerID uint) error {
	comment, err := s.commentRepo.FindByID(ctx, commentID)
	if err != nil {
		return err
	}
	if comment.AuthorID != callerID {
		issue, err := s.issueRepo.FindByID(ctx, comment.IssueID)
		if err != nil {
			return err
		}
		membership, err := s.memberRepo.FindByProjectAndUser(ctx, issue.ProjectID, callerID)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.ErrForbidden
			}
			return err
		}
		if membership.Role != "admin" && membership.Role != "owner" {
			return domain.ErrForbidden
		}
	}
	return s.commentRepo.Delete(ctx, commentID)
}

func (s *commentService) requireMemberOfProject(ctx context.Context, projectID, userID uint) error {
	if _, err := s.memberRepo.FindByProjectAndUser(ctx, projectID, userID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrForbidden
		}
		return err
	}
	return nil
}

func toCommentResponse(c *domain.Comment) *dto.CommentResponse {
	return &dto.CommentResponse{
		ID:        c.ID,
		IssueID:   c.IssueID,
		AuthorID:  c.AuthorID,
		Body:      c.Body,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
