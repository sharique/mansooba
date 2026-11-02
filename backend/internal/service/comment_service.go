package service

import (
	"context"
	"regexp"
	"strings"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
)

// A mention is "@" plus a username (letters, digits, underscore). The "@" must not
// follow a letter, digit, underscore, dot or another "@", so "(@sam)", "**@sam**"
// and "> @sam" are mentions while "a@b.com" and "a.b@c" are not (ADR-038). RE2 has
// no lookbehind, so the preceding character is part of the match.
var mentionRe = regexp.MustCompile(`(?:^|[^A-Za-z0-9_.@])@([A-Za-z0-9_]+)`)

// CommentService manages comments on issues.
type CommentService interface {
	Create(ctx context.Context, issueID, callerID uint, req dto.CreateCommentRequest) (*dto.CommentResponse, error)
	List(ctx context.Context, issueID, callerID uint) ([]*dto.CommentResponse, error)
	// Update and Delete address a comment through its own task: a comment that
	// belongs to a different task than issueID is ErrNotFound.
	Update(ctx context.Context, issueID, commentID, callerID uint, req dto.UpdateCommentRequest) (*dto.CommentResponse, error)
	Delete(ctx context.Context, issueID, commentID, callerID uint) error
}

type commentService struct {
	commentRepo domain.CommentRepository
	issueRepo   domain.IssueRepository
	memberRepo  domain.ProjectMemberRepository
	activitySvc ActivityService
	notifRepo   domain.NotificationRepository
	userRepo    domain.UserRepository
	access      Access
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
		access:      NewAccess(nil, memberRepo, userRepo),
	}
}

func (s *commentService) Create(ctx context.Context, issueID, callerID uint, req dto.CreateCommentRequest) (*dto.CommentResponse, error) {
	issue, err := s.issueRepo.FindByID(ctx, issueID)
	if err != nil {
		return nil, err
	}
	if _, err := s.access.RequireProject(ctx, issue.ProjectID, callerID, domain.CapCommentWrite); err != nil {
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

	s.sendMentionNotifications(ctx, comment, issue.ProjectID)

	return toCommentResponse(comment), nil
}

func (s *commentService) List(ctx context.Context, issueID, callerID uint) ([]*dto.CommentResponse, error) {
	issue, err := s.issueRepo.FindByID(ctx, issueID)
	if err != nil {
		return nil, err
	}
	if _, err := s.access.RequireProject(ctx, issue.ProjectID, callerID, domain.CapProjectView); err != nil {
		return nil, err
	}

	comments, err := s.commentRepo.FindByIssueID(ctx, issueID)
	if err != nil {
		return nil, err
	}

	type authorInfo struct{ Name, AvatarURL string }

	// Collect unique author IDs to reduce repeated lookups (one call per unique ID).
	idSet := make(map[uint]authorInfo)
	for _, c := range comments {
		idSet[c.AuthorID] = authorInfo{}
	}
	for id := range idSet {
		if u, err := s.userRepo.FindByID(ctx, id); err == nil {
			idSet[id] = authorInfo{Name: u.Name, AvatarURL: u.AvatarURL}
		}
	}

	result := make([]*dto.CommentResponse, 0, len(comments))
	for _, c := range comments {
		r := toCommentResponse(c)
		info := idSet[c.AuthorID]
		r.AuthorName = info.Name
		r.AuthorAvatarURL = info.AvatarURL
		result = append(result, r)
	}
	return result, nil
}

func (s *commentService) Update(ctx context.Context, issueID, commentID, callerID uint, req dto.UpdateCommentRequest) (*dto.CommentResponse, error) {
	comment, err := s.commentRepo.FindByID(ctx, commentID)
	if err != nil {
		return nil, err
	}
	if comment.IssueID != issueID {
		return nil, domain.ErrNotFound
	}
	issue, err := s.issueRepo.FindByID(ctx, comment.IssueID)
	if err != nil {
		return nil, err
	}
	// Editing needs permission to comment, and only the author may edit.
	if _, err := s.access.RequireProject(ctx, issue.ProjectID, callerID, domain.CapCommentWrite); err != nil {
		return nil, err
	}
	if comment.AuthorID != callerID {
		return nil, &domain.ForbiddenError{Reason: "Only the author can edit a comment"}
	}
	comment.Body = req.Body
	if err := s.commentRepo.Update(ctx, comment); err != nil {
		return nil, err
	}
	return toCommentResponse(comment), nil
}

func (s *commentService) Delete(ctx context.Context, issueID, commentID, callerID uint) error {
	comment, err := s.commentRepo.FindByID(ctx, commentID)
	if err != nil {
		return err
	}
	if comment.IssueID != issueID {
		return domain.ErrNotFound
	}
	issue, err := s.issueRepo.FindByID(ctx, comment.IssueID)
	if err != nil {
		return err
	}
	// The author may delete their own comment while they can still comment (a
	// viewer cannot); deleting anyone else's takes the delete-any capability.
	need := domain.CapCommentDeleteAny
	if comment.AuthorID == callerID {
		need = domain.CapCommentWrite
	}
	if _, err := s.access.RequireProject(ctx, issue.ProjectID, callerID, need); err != nil {
		return err
	}
	return s.commentRepo.Delete(ctx, commentID)
}

// sendMentionNotifications tells mentioned people about a comment, but only those
// who belong to the project: a notification names the task and project, so
// mentioning an outsider must not reveal either.
func (s *commentService) sendMentionNotifications(ctx context.Context, comment *domain.Comment, projectID uint) {
	matches := mentionRe.FindAllStringSubmatch(comment.Body, -1)
	seen := make(map[uint]bool)
	for _, m := range matches {
		handle := m[1]
		user, err := s.userRepo.FindByUsername(ctx, strings.ToLower(handle))
		if err != nil {
			continue
		}
		if seen[user.ID] {
			continue
		}
		seen[user.ID] = true
		if _, err := s.access.RequireProject(ctx, projectID, user.ID, domain.CapProjectView); err != nil {
			continue
		}
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
