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

// NotificationRepository defines persistence for Notification.
type NotificationRepository interface {
	Create(ctx context.Context, n *Notification) error
	FindByRecipientID(ctx context.Context, recipientID uint) ([]*Notification, error)
	MarkRead(ctx context.Context, id uint) error
}
