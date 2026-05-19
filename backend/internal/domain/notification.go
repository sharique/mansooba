package domain

import (
	"context"
	"time"
)

// Notification is created when a user is @mentioned in a comment.
type Notification struct {
	ID          uint      `gorm:"primaryKey"`
	RecipientID uint      `gorm:"not null;index"`
	ActorID     uint      `gorm:"not null"`
	IssueID     uint      `gorm:"not null"`
	CommentID   uint      `gorm:"not null"`
	Read        bool      `gorm:"default:false"`
	CreatedAt   time.Time
}

// NotificationDetail is a Notification enriched with project context from a JOIN.
type NotificationDetail struct {
	Notification
	ProjectKey string
	IssueKey   string
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
