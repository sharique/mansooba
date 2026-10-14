package service

import (
	"context"

	"github.com/sharique/mansooba/internal/domain"
	"go.uber.org/zap"
)

// NotificationService owns the rules for "who is told, and when" when an
// assignment or a project membership changes (feature 017, ADR-036).
//
// Rules: never notify the person who made the change; task notifications go to
// current project members only (so an issue key is never disclosed to someone
// without access); failures are logged and never returned, because a missing
// notification must not fail the change that caused it.
//
// Mention notifications are created by commentService and are unchanged.
type NotificationService interface {
	// NotifyAssignmentChange tells the new assignee they were assigned and the
	// previous assignee they no longer are. Either may be nil.
	NotifyAssignmentChange(ctx context.Context, projectID, issueID, actorID uint, oldAssignee, newAssignee *uint)
	// NotifyMembershipChange tells targetID they were added to or removed from a project.
	NotifyMembershipChange(ctx context.Context, projectID, actorID, targetID uint, added bool)
}

type notificationService struct {
	notifRepo  domain.NotificationRepository
	memberRepo domain.ProjectMemberRepository
	log        *zap.Logger
}

func NewNotificationService(
	notifRepo domain.NotificationRepository,
	memberRepo domain.ProjectMemberRepository,
	log *zap.Logger,
) NotificationService {
	return &notificationService{notifRepo: notifRepo, memberRepo: memberRepo, log: log}
}

func (s *notificationService) NotifyAssignmentChange(ctx context.Context, projectID, issueID, actorID uint, oldAssignee, newAssignee *uint) {
	if sameUser(oldAssignee, newAssignee) {
		return
	}
	if newAssignee != nil {
		s.notifyTask(ctx, projectID, issueID, actorID, *newAssignee, domain.NotificationKindTaskAssigned)
	}
	if oldAssignee != nil {
		s.notifyTask(ctx, projectID, issueID, actorID, *oldAssignee, domain.NotificationKindTaskUnassigned)
	}
}

func (s *notificationService) NotifyMembershipChange(ctx context.Context, projectID, actorID, targetID uint, added bool) {
	if targetID == actorID {
		return
	}
	kind := domain.NotificationKindProjectMemberRemoved
	if added {
		kind = domain.NotificationKindProjectMemberAdded
	}
	s.create(ctx, &domain.Notification{
		RecipientID: targetID, ActorID: actorID, Kind: kind, ProjectID: projectID,
	})
}

func (s *notificationService) notifyTask(ctx context.Context, projectID, issueID, actorID, recipientID uint, kind string) {
	if recipientID == actorID {
		return
	}
	if _, err := s.memberRepo.FindByProjectAndUser(ctx, projectID, recipientID); err != nil {
		// Not a member (or the lookup failed): skip rather than disclose the issue key.
		return
	}
	s.create(ctx, &domain.Notification{
		RecipientID: recipientID, ActorID: actorID, Kind: kind, IssueID: issueID,
	})
}

func (s *notificationService) create(ctx context.Context, n *domain.Notification) {
	if err := s.notifRepo.Create(ctx, n); err != nil {
		s.log.Warn("failed to create notification",
			zap.String("kind", n.Kind),
			zap.Uint("actor_id", n.ActorID),
			zap.Uint("recipient_id", n.RecipientID),
			zap.Error(err))
	}
}

func sameUser(a, b *uint) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
