package domain

import (
	"context"
	"time"
)

const (
	RelationTypeBlocks      = "blocks"
	RelationTypeIsBlockedBy = "is_blocked_by"
	RelationTypeRelatesTo   = "relates_to"
	RelationTypeDuplicates  = "duplicates"
)

// IssueRelation records a directional or symmetric relationship between two issues.
// Directional ("blocks"): two rows are stored — (A→B, "blocks") and (B→A, "is_blocked_by").
// Symmetric ("relates_to", "duplicates"): one row; FindByIssueID queries both sides.
type IssueRelation struct {
	ID              uint      `gorm:"primaryKey"`
	IssueID         uint      `gorm:"not null;uniqueIndex:idx_relation_unique"`
	RelatedIssueID  uint      `gorm:"not null;uniqueIndex:idx_relation_unique"`
	RelationType    string    `gorm:"not null;uniqueIndex:idx_relation_unique"`
	CreatedByID     uint
	CreatedAt       time.Time
}

// IssueRelationRepository defines the persistence contract for IssueRelation.
type IssueRelationRepository interface {
	// Create inserts a new relation row.
	Create(ctx context.Context, rel *IssueRelation) error
	// FindByIssueID returns all relations visible from issueID's perspective.
	FindByIssueID(ctx context.Context, issueID uint) ([]*IssueRelation, error)
	// FindByID returns a single relation by primary key, or ErrNotFound.
	FindByID(ctx context.Context, id uint) (*IssueRelation, error)
	// Delete removes the relation with the given primary key.
	Delete(ctx context.Context, id uint) error
	// DeleteByIssueID removes all rows where issue_id = issueID OR related_issue_id = issueID.
	DeleteByIssueID(ctx context.Context, issueID uint) error
	// ExistsBlock returns true if a "blocks" row with (fromID → toID) exists.
	ExistsBlock(ctx context.Context, fromID, toID uint) (bool, error)
}
