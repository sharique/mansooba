package service

import (
	"context"
	"errors"
	"regexp"

	"github.com/sharique/jira-go/internal/domain"
	"github.com/sharique/jira-go/internal/dto"
)

var mentionRe = regexp.MustCompile(`@([\w.]+)`)

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
	notifRepo   domain.NotificationRepository
	userRepo    domain.UserRepository
}

func NewCommentService(
	commentRepo domain.CommentRepository,
	issueRepo domain.IssueRepository,
	memberRepo domain.ProjectMemberRepository,
	activitySvc ActivityService,
	notifRepo domain.NotificationRepository,
	userRepo domain.UserRepository,
) CommentService {
	return &commentService{
		commentRepo: commentRepo,
		issueRepo:   issueRepo,
		memberRepo:  memberRepo,
		activitySvc: activitySvc,
		notifRepo:   notifRepo,
		userRepo:    userRepo,
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

	s.sendMentionNotifications(ctx, comment)

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

	// Collect unique author IDs to batch-resolve names.
	idSet := make(map[uint]string)
	for _, c := range comments {
		idSet[c.AuthorID] = ""
	}
	for id := range idSet {
		if u, err := s.userRepo.FindByID(ctx, id); err == nil {
			idSet[id] = u.Name
		}
	}

	result := make([]*dto.CommentResponse, 0, len(comments))
	for _, c := range comments {
		r := toCommentResponse(c)
		r.AuthorName = idSet[c.AuthorID]
		result = append(result, r)
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

func (s *commentService) sendMentionNotifications(ctx context.Context, comment *domain.Comment) {
	matches := mentionRe.FindAllStringSubmatch(comment.Body, -1)
	seen := make(map[uint]bool)
	for _, m := range matches {
		handle := m[1]
		user, err := s.userRepo.FindByEmailPrefix(ctx, handle)
		if err != nil {
			continue
		}
		if seen[user.ID] {
			continue
		}
		seen[user.ID] = true
		_ = s.notifRepo.Create(ctx, &domain.Notification{
			RecipientID: user.ID,
			ActorID:     comment.AuthorID,
			IssueID:     comment.IssueID,
			CommentID:   comment.ID,
		})
	}
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
