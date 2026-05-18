package domain

import (
	"context"
	"time"
)

// Comment is a user-authored markdown message attached to an issue.
type Comment struct {
	ID        uint      `gorm:"primaryKey"`
	IssueID   uint      `gorm:"not null;index"`
	AuthorID  uint      `gorm:"not null"`
	Body      string    `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// CommentRepository defines persistence for Comment.
type CommentRepository interface {
	Create(ctx context.Context, comment *Comment) error
	FindByIssueID(ctx context.Context, issueID uint) ([]*Comment, error)
	FindByID(ctx context.Context, id uint) (*Comment, error)
	Update(ctx context.Context, comment *Comment) error
	Delete(ctx context.Context, id uint) error
}
