package service_test

import (
	"context"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/sharique/mansooba/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newSettingService(t *testing.T) service.SettingService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.GlobalSetting{}))
	return service.NewSettingService(repository.NewSettingRepository(db))
}

func strPtr(s string) *string { return &s }

func TestSettingService_DateFormats_ValidValues(t *testing.T) {
	svc := newSettingService(t)
	ctx := context.Background()
	for _, f := range []string{"YYYY-MM-DD", "DD/MM/YYYY", "MM/DD/YYYY", "D-MMM-YYYY"} {
		_, err := svc.Patch(ctx, 1, dto.PatchSettingsRequest{DateFormat: strPtr(f)})
		assert.NoError(t, err, "expected %q to be accepted", f)
	}
}

func TestSettingService_DateFormats_InvalidValues(t *testing.T) {
	svc := newSettingService(t)
	ctx := context.Background()
	for _, f := range []string{"not-valid", "dd/mm/yyyy", "D/MMM/YYYY", "D MMM YYYY", ""} {
		_, err := svc.Patch(ctx, 1, dto.PatchSettingsRequest{DateFormat: strPtr(f)})
		assert.ErrorIs(t, err, service.ErrInvalidSettingValue, "expected %q to be rejected", f)
	}
}

func TestSettingService_Patch_AllFields_WithNewFormat(t *testing.T) {
	svc := newSettingService(t)
	ctx := context.Background()
	resp, err := svc.Patch(ctx, 1, dto.PatchSettingsRequest{
		OrganizationName: strPtr("Mansooba"),
		DateFormat:       strPtr("D-MMM-YYYY"),
		TimeFormat:       strPtr("24h"),
		Locale:           strPtr("en-US"),
		WeekStartDay:     strPtr("monday"),
	})
	require.NoError(t, err)
	assert.Equal(t, "D-MMM-YYYY", resp.DateFormat)
}
