package repository

import (
	"context"
	"errors"

	"github.com/sharique/mansooba/internal/domain"
	"gorm.io/gorm"
)

type issueRelationRepo struct{ db *gorm.DB }

// NewIssueRelationRepository returns an IssueRelationRepository backed by GORM.
func NewIssueRelationRepository(db *gorm.DB) domain.IssueRelationRepository {
	return &issueRelationRepo{db: db}
}

func (r *issueRelationRepo) Create(ctx context.Context, rel *domain.IssueRelation) error {
	return r.db.WithContext(ctx).Create(rel).Error
}

// FindByIssueID returns all relations visible from issueID's perspective.
// Directional rows already have issueID as issue_id. Symmetric rows can have
// issueID on either side, so we also query via related_issue_id for those types.
func (r *issueRelationRepo) FindByIssueID(ctx context.Context, issueID uint) ([]*domain.IssueRelation, error) {
	var direct []*domain.IssueRelation
	if err := r.db.WithContext(ctx).
		Where("issue_id = ?", issueID).
		Order("created_at DESC").
		Find(&direct).Error; err != nil {
		return nil, err
	}

	// For symmetric types stored the other way round, fetch the mirror rows.
	var mirrored []*domain.IssueRelation
	if err := r.db.WithContext(ctx).
		Where("related_issue_id = ? AND relation_type IN (?, ?)", issueID,
			domain.RelationTypeRelatesTo, domain.RelationTypeDuplicates).
		Order("created_at DESC").
		Find(&mirrored).Error; err != nil {
		return nil, err
	}

	seen := make(map[uint]bool, len(direct))
	result := make([]*domain.IssueRelation, 0, len(direct)+len(mirrored))
	for _, r := range direct {
		seen[r.ID] = true
		result = append(result, r)
	}
	for _, r := range mirrored {
		if !seen[r.ID] {
			result = append(result, r)
		}
	}
	return result, nil
}

func (r *issueRelationRepo) FindByID(ctx context.Context, id uint) (*domain.IssueRelation, error) {
	var rel domain.IssueRelation
	if err := r.db.WithContext(ctx).First(&rel, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &rel, nil
}

func (r *issueRelationRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&domain.IssueRelation{}, id).Error
}

func (r *issueRelationRepo) DeleteByIssueID(ctx context.Context, issueID uint) error {
	return r.db.WithContext(ctx).
		Where("issue_id = ? OR related_issue_id = ?", issueID, issueID).
		Delete(&domain.IssueRelation{}).Error
}

func (r *issueRelationRepo) ExistsBlock(ctx context.Context, fromID, toID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.IssueRelation{}).
		Where("issue_id = ? AND related_issue_id = ? AND relation_type = ?",
			fromID, toID, domain.RelationTypeBlocks).
		Count(&count).Error
	return count > 0, err
}
