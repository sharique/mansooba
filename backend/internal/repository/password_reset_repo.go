package repository

import (
	"context"
	"errors"
	"time"

	"github.com/sharique/mansooba/internal/domain"
	"gorm.io/gorm"
)

type passwordResetRepo struct {
	db *gorm.DB
}

// NewPasswordResetRepository returns a PasswordResetRepository backed by db.
func NewPasswordResetRepository(db *gorm.DB) domain.PasswordResetRepository {
	return &passwordResetRepo{db: db}
}

func (r *passwordResetRepo) Upsert(ctx context.Context, token *domain.PasswordResetToken) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", token.UserID).Delete(&domain.PasswordResetToken{}).Error; err != nil {
			return err
		}
		return tx.Create(token).Error
	})
}

func (r *passwordResetRepo) FindByHash(ctx context.Context, hash string) (*domain.PasswordResetToken, error) {
	var t domain.PasswordResetToken
	err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	return &t, err
}

func (r *passwordResetRepo) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&domain.PasswordResetToken{}, id).Error
}

func (r *passwordResetRepo) PurgeExpired(ctx context.Context, cutoff time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("created_at < ?", cutoff).
		Delete(&domain.PasswordResetToken{})
	return result.RowsAffected, result.Error
}
