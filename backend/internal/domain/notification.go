package domain

import (
	"context"
	"time"
)

// Notification kinds. A client must treat an unknown kind as a generic notification.
const (
	NotificationKindMention              = "mention"
	NotificationKindTaskAssigned         = "task_assigned"
	NotificationKindTaskUnassigned       = "task_unassigned"
	NotificationKindProjectMemberAdded   = "project_member_added"
	NotificationKindProjectMemberRemoved = "project_member_removed"
)

// Notification tells one recipient about an event another user caused. IssueID,
// CommentID and ProjectID use 0 for "not applicable to this kind" (ADR-036):
// the first two columns are NOT NULL, and altering them is avoided.
type Notification struct {
	ID          uint      `gorm:"primaryKey"`
	RecipientID uint      `gorm:"not null;index"`
	ActorID     uint      `gorm:"not null"`
	Kind        string    `gorm:"not null;default:mention"`
	ProjectID   uint      `gorm:"not null;default:0"`
	IssueID     uint      `gorm:"not null"`
	CommentID   uint      `gorm:"not null"`
	Read        bool      `gorm:"default:false"`
	CreatedAt   time.Time
}

// NotificationDetail is a Notification enriched with names from LEFT JOINs.
type NotificationDetail struct {
	Notification
	ProjectKey  string
	ProjectName string
	IssueKey    string
	ActorName   string
}

// NotificationRepository defines persistence for Notification.
type NotificationRepository interface {
	Create(ctx context.Context, n *Notification) error
	// FindUnreadByRecipientID returns all unread notifications for the given recipient,
	// enriched with project_key and issue_key via JOIN.
	FindUnreadByRecipientID(ctx context.Context, recipientID uint) ([]*NotificationDetail, error)
	// MarkRead sets Read=true for the given notification ID, scoped to the recipient.
	// Returns ErrNotFound if the notification does not exist or belongs to a different user.
	MarkRead(ctx context.Context, id, recipientID uint) error
}
