package domain

import (
	"context"
	"time"
)

// ActivityEvent is an immutable record of a change to an issue.
// OldValue and NewValue are human-readable strings (e.g. "todo", "in_progress").
type ActivityEvent struct {
	ID        uint   `gorm:"primaryKey"`
	IssueID   uint   `gorm:"not null;index"`
	ActorID   uint   `gorm:"not null"`
	Kind      string `gorm:"not null"`
	OldValue  string
	NewValue  string
	CreatedAt time.Time
}

// Activity kind constants.
const (
	ActivityStatusChanged      = "status_changed"
	ActivityAssigneeChanged    = "assignee_changed"
	ActivityPriorityChanged    = "priority_changed"
	ActivitySprintChanged      = "sprint_changed"
	ActivityStoryPointsChanged = "story_points_changed"
	ActivityCommentAdded       = "comment_added"
	ActivityLabelAdded         = "label_added"
	ActivityLabelRemoved       = "label_removed"
	ActivityAttachmentAdded    = "attachment_added"
	ActivityAttachmentRemoved  = "attachment_removed"
)

// ActivityRepository defines persistence for ActivityEvent.
type ActivityRepository interface {
	Create(ctx context.Context, event *ActivityEvent) error
	FindByIssueID(ctx context.Context, issueID uint) ([]*ActivityEvent, error)
	// FindByActorID returns events where the actor is actorID, ordered by created_at DESC,
	// with pagination. Used by the "my activity" endpoint.
	FindByActorID(ctx context.Context, actorID uint, limit, offset int) ([]*ActivityEvent, error)
}
