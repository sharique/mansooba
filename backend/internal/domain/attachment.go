package domain

import (
	"context"
	"time"
)

// Attachment is a file uploaded to an issue, stored as an S3 object.
// ObjectKey is a generated, collision-proof S3 key — never the original
// filename — so two attachments on the same issue can share a display name
// without colliding on storage.
type Attachment struct {
	ID          uint   `gorm:"primaryKey"`
	IssueID     uint   `gorm:"not null;index"`
	UploaderID  uint   `gorm:"not null"`
	Filename    string `gorm:"not null"`
	ObjectKey   string `gorm:"not null"`
	ContentType string `gorm:"not null"`
	SizeBytes   int64  `gorm:"not null"`
	CreatedAt   time.Time
}

// AttachmentRepository defines persistence for Attachment.
type AttachmentRepository interface {
	Create(ctx context.Context, a *Attachment) error
	FindByIssueID(ctx context.Context, issueID uint) ([]*Attachment, error)
	FindByID(ctx context.Context, id uint) (*Attachment, error)
	CountByIssueID(ctx context.Context, issueID uint) (int64, error)
	Delete(ctx context.Context, id uint) error
	DeleteByIssueID(ctx context.Context, issueID uint) error
}
