package repository

import (
	"context"
	"errors"

	"github.com/sharique/mansooba/internal/domain"
	"gorm.io/gorm"
)

type attachmentRepo struct {
	db *gorm.DB
}

// NewAttachmentRepository returns a GORM-backed implementation of domain.AttachmentRepository.
func NewAttachmentRepository(db *gorm.DB) domain.AttachmentRepository {
	return &attachmentRepo{db: db}
}

// Create inserts a new attachment record and populates the ID field on success.
func (r *attachmentRepo) Create(ctx context.Context, a *domain.Attachment) error {
	return r.db.WithContext(ctx).Create(a).Error
}

// FindByID retrieves an attachment by primary key.
// Returns domain.ErrNotFound when no row matches.
func (r *attachmentRepo) FindByID(ctx context.Context, id uint) (*domain.Attachment, error) {
	var a domain.Attachment
	if err := r.db.WithContext(ctx).First(&a, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &a, nil
}

// FindByIssueID returns all attachments for the given issue, most-recent-first.
func (r *attachmentRepo) FindByIssueID(ctx context.Context, issueID uint) ([]*domain.Attachment, error) {
	var attachments []*domain.Attachment
	if err := r.db.WithContext(ctx).
		Where("issue_id = ?", issueID).
		Order("created_at DESC").
		Find(&attachments).Error; err != nil {
		return nil, err
	}
	return attachments, nil
}

// CountByIssueID returns the number of attachments on the given issue.
func (r *attachmentRepo) CountByIssueID(ctx context.Context, issueID uint) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&domain.Attachment{}).
		Where("issue_id = ?", issueID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// Delete removes an attachment by primary key. Attachment has no soft-delete
// column, so this is a hard delete — there's no product requirement to
// recover a deleted attachment (spec.md Assumptions: no file versioning).
func (r *attachmentRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&domain.Attachment{}, id).Error
}

// DeleteByIssueID removes all attachments belonging to an issue.
func (r *attachmentRepo) DeleteByIssueID(ctx context.Context, issueID uint) error {
	return r.db.WithContext(ctx).
		Where("issue_id = ?", issueID).
		Delete(&domain.Attachment{}).Error
}
