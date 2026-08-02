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
	require.NoError(t, db.AutoMigrate(&domain.GlobalSetting{}, &domain.User{}))
	return service.NewSettingService(repository.NewSettingRepository(db), repository.NewUserRepository(db), stubSystemLogService{})
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

func TestSettingService_GetAll_DefaultsSystemLogRetentionTo90Days(t *testing.T) {
	svc := newSettingService(t)
	resp, err := svc.GetAll(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "90", resp.SystemLogRetentionDays)
}

func TestSettingService_SystemLogRetentionDays_ValidValues(t *testing.T) {
	svc := newSettingService(t)
	ctx := context.Background()
	for _, v := range []string{"1", "30", "90", "3650"} {
		resp, err := svc.Patch(ctx, 1, dto.PatchSettingsRequest{SystemLogRetentionDays: strPtr(v)})
		require.NoError(t, err, "expected %q to be accepted", v)
		assert.Equal(t, v, resp.SystemLogRetentionDays)
	}
}

func TestSettingService_SystemLogRetentionDays_InvalidValues(t *testing.T) {
	svc := newSettingService(t)
	ctx := context.Background()
	for _, v := range []string{"0", "-1", "not-a-number", "1.5", ""} {
		_, err := svc.Patch(ctx, 1, dto.PatchSettingsRequest{SystemLogRetentionDays: strPtr(v)})
		assert.ErrorIs(t, err, service.ErrInvalidSettingValue, "expected %q to be rejected", v)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// 011-system-logs (US1): Record calls from Patch (T025)
// ──────────────────────────────────────────────────────────────────────────────

func TestSettingService_Patch_RecordsOneEntryPerChangedKey(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.GlobalSetting{}, &domain.User{}))
	userRepo := repository.NewUserRepository(db)
	require.NoError(t, userRepo.Create(context.Background(), &domain.User{Name: "Admin", Email: "admin@example.com", IsActive: true}))
	admin, err := userRepo.FindByEmail(context.Background(), "admin@example.com")
	require.NoError(t, err)

	systemLogSvc := &recordingSystemLogService{}
	svc := service.NewSettingService(repository.NewSettingRepository(db), userRepo, systemLogSvc)

	_, err = svc.Patch(context.Background(), admin.ID, dto.PatchSettingsRequest{
		OrganizationName: strPtr("Acme Corp"),
		TimeFormat:       strPtr("12h"),
	})
	require.NoError(t, err)

	entries := systemLogSvc.all()
	require.Len(t, entries, 2, "expected one entry per changed key")
	for _, e := range entries {
		assert.Equal(t, domain.SystemLogCategorySettingsChange, e.Category)
		assert.Equal(t, "admin@example.com", e.Actor)
		assert.Equal(t, admin.ID, e.ActorID)
	}
}

func TestSettingService_Patch_RecordsChangeToRetentionSettingItself(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.GlobalSetting{}, &domain.User{}))
	userRepo := repository.NewUserRepository(db)

	systemLogSvc := &recordingSystemLogService{}
	svc := service.NewSettingService(repository.NewSettingRepository(db), userRepo, systemLogSvc)

	_, err = svc.Patch(context.Background(), 1, dto.PatchSettingsRequest{SystemLogRetentionDays: strPtr("30")})
	require.NoError(t, err)

	entries := systemLogSvc.all()
	require.Len(t, entries, 1)
	assert.Contains(t, entries[0].Detail, "system_log_retention_days")
	assert.Contains(t, entries[0].Detail, "30")
}

func TestSettingService_Patch_NoOpRequest_RecordsNothing(t *testing.T) {
	systemLogSvc := &recordingSystemLogService{}
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.GlobalSetting{}, &domain.User{}))
	svc := service.NewSettingService(repository.NewSettingRepository(db), repository.NewUserRepository(db), systemLogSvc)

	_, err = svc.Patch(context.Background(), 1, dto.PatchSettingsRequest{})
	require.NoError(t, err)
	assert.Empty(t, systemLogSvc.all(), "a Patch with no fields set should record nothing")
}
