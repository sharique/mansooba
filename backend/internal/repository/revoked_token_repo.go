package repository

import (
	"context"
	"time"

	"github.com/sharique/mansooba/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type gormRevokedTokenRepository struct{ db *gorm.DB }

// NewRevokedTokenRepository returns a GORM-backed RevokedTokenRepository.
func NewRevokedTokenRepository(db *gorm.DB) domain.RevokedTokenRepository {
	return &gormRevokedTokenRepository{db: db}
}

// Create inserts a revocation record. Duplicate JTI is silently ignored.
func (r *gormRevokedTokenRepository) Create(ctx context.Context, token *domain.RevokedToken) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(token).Error
}

// Exists returns true if the given JTI is in the revocation table.
func (r *gormRevokedTokenRepository) Exists(ctx context.Context, jti string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.RevokedToken{}).
		Where("jti = ?", jti).Count(&count).Error
	return count > 0, err
}

// DeleteExpired removes all records whose ExpiresAt is before now.
func (r *gormRevokedTokenRepository) DeleteExpired(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&domain.RevokedToken{})
	return result.RowsAffected, result.Error
}
