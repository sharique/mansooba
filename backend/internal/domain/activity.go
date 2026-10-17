package domain

import (
	"context"
	"time"
)

// ActivityEvent is an immutable record of a change to an issue.
// OldValue and NewValue are human-readable strings (e.g. "todo", "in_progress").
type ActivityEvent struct {
	ID        uint   `gorm:"primaryKey"`
	IssueID   uint   `gorm:"not null;index;index:idx_activity_issue_created,priority:1"`
	ActorID   uint   `gorm:"not null"`
	Kind      string `gorm:"not null"`
	OldValue  string
	NewValue  string
	CreatedAt time.Time `gorm:"index:idx_activity_issue_created,priority:2"`
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
	// FindRecentByProject returns, newest first and at most limit, the events since
	// the given time on issues of the project that show progress or conversation: a
	// status change whose new status is in progress, in review or done (old status
	// different), and every added comment. When sprintID is set only issues that are
	// currently in that sprint are considered.
	FindRecentByProject(ctx context.Context, projectID uint, since time.Time, sprintID *uint, limit int) ([]*ActivityEvent, error)
}
