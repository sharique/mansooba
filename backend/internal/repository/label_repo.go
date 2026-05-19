package repository

import (
	"context"
	"errors"

	"github.com/sharique/jira-go/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type labelRepo struct{ db *gorm.DB }

// NewLabelRepository returns a GORM-backed domain.LabelRepository.
func NewLabelRepository(db *gorm.DB) domain.LabelRepository {
	return &labelRepo{db: db}
}

func (r *labelRepo) Create(ctx context.Context, l *domain.Label) error {
	return r.db.WithContext(ctx).Create(l).Error
}

func (r *labelRepo) FindByProjectID(ctx context.Context, projectID uint) ([]*domain.Label, error) {
	var labels []*domain.Label
	if err := r.db.WithContext(ctx).Where("project_id = ?", projectID).Find(&labels).Error; err != nil {
		return nil, err
	}
	return labels, nil
}

func (r *labelRepo) FindByID(ctx context.Context, id uint) (*domain.Label, error) {
	var l domain.Label
	if err := r.db.WithContext(ctx).First(&l, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &l, nil
}

func (r *labelRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&domain.Label{}, id).Error
}

func (r *labelRepo) AttachToIssue(ctx context.Context, issueID, labelID uint) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&domain.IssueLabel{IssueID: issueID, LabelID: labelID}).Error
}

func (r *labelRepo) DetachFromIssue(ctx context.Context, issueID, labelID uint) error {
	return r.db.WithContext(ctx).
		Where("issue_id = ? AND label_id = ?", issueID, labelID).
		Delete(&domain.IssueLabel{}).Error
}

func (r *labelRepo) FindByIssueID(ctx context.Context, issueID uint) ([]*domain.Label, error) {
	var labels []*domain.Label
	if err := r.db.WithContext(ctx).
		Joins("JOIN issue_labels ON issue_labels.label_id = labels.id").
		Where("issue_labels.issue_id = ?", issueID).
		Find(&labels).Error; err != nil {
		return nil, err
	}
	return labels, nil
}
