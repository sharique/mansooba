package repository

import (
	"context"

	"github.com/sharique/mansooba/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type settingRepo struct{ db *gorm.DB }

// NewSettingRepository returns a GlobalSettingRepository backed by GORM.
func NewSettingRepository(db *gorm.DB) domain.GlobalSettingRepository {
	return &settingRepo{db: db}
}

func (r *settingRepo) FindAll(ctx context.Context) ([]*domain.GlobalSetting, error) {
	var settings []*domain.GlobalSetting
	if err := r.db.WithContext(ctx).Find(&settings).Error; err != nil {
		return nil, err
	}
	return settings, nil
}

func (r *settingRepo) Upsert(ctx context.Context, s *domain.GlobalSetting) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "setting_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"setting_value", "updated_by_id", "updated_at"}),
		}).
		Create(s).Error
}
