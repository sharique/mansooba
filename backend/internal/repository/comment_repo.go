package repository

import (
	"context"
	"errors"

	"github.com/sharique/mansooba/internal/domain"
	"gorm.io/gorm"
)

type commentRepo struct{ db *gorm.DB }

func NewCommentRepository(db *gorm.DB) domain.CommentRepository {
	return &commentRepo{db: db}
}

func (r *commentRepo) Create(ctx context.Context, c *domain.Comment) error {
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *commentRepo) FindByIssueID(ctx context.Context, issueID uint) ([]*domain.Comment, error) {
	var comments []*domain.Comment
	if err := r.db.WithContext(ctx).
		Where("issue_id = ?", issueID).
		Order("created_at ASC").
		Find(&comments).Error; err != nil {
		return nil, err
	}
	return comments, nil
}

func (r *commentRepo) FindByID(ctx context.Context, id uint) (*domain.Comment, error) {
	var c domain.Comment
	if err := r.db.WithContext(ctx).First(&c, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *commentRepo) Update(ctx context.Context, c *domain.Comment) error {
	return r.db.WithContext(ctx).Save(c).Error
}

func (r *commentRepo) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&domain.Comment{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}
