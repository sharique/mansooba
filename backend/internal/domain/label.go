package domain

import (
	"context"
	"time"
)

// Label is a project-scoped tag that can be attached to issues.
// Color must be one of the 12 palette values defined in LabelPalette.
type Label struct {
	ID        uint      `gorm:"primaryKey"`
	ProjectID uint      `gorm:"not null;index"`
	Name      string    `gorm:"not null"`
	Color     string    `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

// IssueLabel is the join table between issues and labels.
type IssueLabel struct {
	IssueID uint `gorm:"primaryKey"`
	LabelID uint `gorm:"primaryKey"`
}

// LabelPalette contains the 12 allowed hex color values for labels.
var LabelPalette = map[string]bool{
	"#e11d48": true, "#f97316": true, "#eab308": true, "#22c55e": true,
	"#06b6d4": true, "#3b82f6": true, "#8b5cf6": true, "#ec4899": true,
	"#6b7280": true, "#78716c": true, "#0f172a": true, "#ffffff": true,
}

// LabelRepository defines persistence for Label and IssueLabel.
type LabelRepository interface {
	Create(ctx context.Context, label *Label) error
	FindByProjectID(ctx context.Context, projectID uint) ([]*Label, error)
	FindByID(ctx context.Context, id uint) (*Label, error)
	Delete(ctx context.Context, id uint) error
	// AttachToIssue creates an IssueLabel row (idempotent — ignore duplicate key).
	AttachToIssue(ctx context.Context, issueID, labelID uint) error
	DetachFromIssue(ctx context.Context, issueID, labelID uint) error
	FindByIssueID(ctx context.Context, issueID uint) ([]*Label, error)
}
