package domain

import (
	"context"
	"time"
)

const (
	SettingKeyOrganizationName = "organization_name"
	SettingKeyDateFormat       = "date_format"
	SettingKeyTimeFormat       = "time_format"
	SettingKeyLocale           = "locale"
	SettingKeyWeekStartDay     = "week_start_day"
)

// GlobalSetting stores a single platform-wide configuration key-value pair.
type GlobalSetting struct {
	ID           uint      `gorm:"primaryKey"`
	SettingKey   string    `gorm:"uniqueIndex;not null"`
	SettingValue string    `gorm:"not null"`
	UpdatedByID  uint
	UpdatedAt    time.Time
}

// GlobalSettingRepository defines the persistence contract for GlobalSetting.
type GlobalSettingRepository interface {
	// FindAll returns all rows in the global_settings table.
	FindAll(ctx context.Context) ([]*GlobalSetting, error)
	// Upsert inserts or updates the row for the given key.
	Upsert(ctx context.Context, setting *GlobalSetting) error
}
