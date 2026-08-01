package service_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/sharique/mansooba/internal/service"
)

// mockSystemLogRepo is a test double for domain.SystemLogRepository, letting
// tests control Create's outcome and observe its calls without a real Loki.
type mockSystemLogRepo struct {
	mu        sync.Mutex
	createErr error
	created   []domain.SystemLogEntry
	createdCh chan struct{}
}

func newMockSystemLogRepo() *mockSystemLogRepo {
	return &mockSystemLogRepo{createdCh: make(chan struct{}, 10)}
}

func (m *mockSystemLogRepo) Create(_ context.Context, entry domain.SystemLogEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createErr == nil {
		m.created = append(m.created, entry)
	}
	m.createdCh <- struct{}{}
	return m.createErr
}

func (m *mockSystemLogRepo) FindPaginated(_ context.Context, _ domain.SystemLogListFilter) (domain.SystemLogListResult, error) {
	return domain.SystemLogListResult{}, nil
}

func (m *mockSystemLogRepo) waitForCreate(t *testing.T) {
	t.Helper()
	select {
	case <-m.createdCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Record's fire-and-forget goroutine to call Create")
	}
}

// newTestSettingStack returns a GlobalSettingRepository and a SettingService
// backed by the same in-memory database — the repository is what
// SystemLogService depends on directly (see systemlog_service.go's doc
// comment on the import-cycle this avoids); the service is only used by
// tests to conveniently set up settings data via Patch.
func newTestSettingStack(t *testing.T) (domain.GlobalSettingRepository, service.SettingService) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.GlobalSetting{}, &domain.User{}))
	repo := repository.NewSettingRepository(db)
	settingSvc := service.NewSettingService(repo, repository.NewUserRepository(db), stubSystemLogService{})
	return repo, settingSvc
}

func TestSystemLogService_Record_WritesEntryOnSuccess(t *testing.T) {
	repo := newMockSystemLogRepo()
	settingRepo, _ := newTestSettingStack(t)
	svc := service.NewSystemLogService(repo, settingRepo, "", zap.NewNop())

	svc.Record(context.Background(), domain.SystemLogEntry{
		Category: domain.SystemLogCategoryAuthentication,
		Action:   "login_failed",
		Outcome:  "failure",
		Actor:    "jane@example.com",
	})

	repo.waitForCreate(t)
	require.Len(t, repo.created, 1)
	assert.Equal(t, "login_failed", repo.created[0].Action)
	assert.False(t, repo.created[0].CreatedAt.IsZero(), "expected CreatedAt to be filled in when not set by the caller")
}

func TestSystemLogService_Record_NeverBlocksOrPanicsOnRepositoryFailure(t *testing.T) {
	repo := newMockSystemLogRepo()
	repo.createErr = errors.New("loki unreachable")
	core, logs := observer.New(zapcore.WarnLevel)
	settingRepo, _ := newTestSettingStack(t)
	svc := service.NewSystemLogService(repo, settingRepo, "", zap.New(core))

	start := time.Now()
	svc.Record(context.Background(), domain.SystemLogEntry{
		Category: domain.SystemLogCategoryAuthentication,
		Action:   "login_failed",
	})
	elapsed := time.Since(start)

	// Record itself must return near-instantly — the push (and its failure)
	// happen in the background goroutine, not synchronously (FR-012).
	assert.Less(t, elapsed, 50*time.Millisecond, "Record must not block on the underlying push")

	repo.waitForCreate(t)

	// FR-012: the failure itself must be independently observable via
	// process-level logging, even though the entry never reached Loki.
	require.Eventually(t, func() bool {
		return logs.FilterMessage("system_log_write_failed").Len() == 1
	}, time.Second, 10*time.Millisecond, "expected a system_log_write_failed warning to be logged")
}

func TestSystemLogService_List_DelegatesToRepository(t *testing.T) {
	repo := newMockSystemLogRepo()
	settingRepo, _ := newTestSettingStack(t)
	svc := service.NewSystemLogService(repo, settingRepo, "", zap.NewNop())

	_, err := svc.List(context.Background(), domain.SystemLogListFilter{Category: domain.SystemLogCategoryAuthentication})
	require.NoError(t, err)
}

func TestSystemLogService_SyncRetention_WritesRuntimeOverridesFile(t *testing.T) {
	repo := newMockSystemLogRepo()
	settingRepo, _ := newTestSettingStack(t)
	path := filepath.Join(t.TempDir(), "runtime-overrides.yaml")
	svc := service.NewSystemLogService(repo, settingRepo, path, zap.NewNop())

	require.NoError(t, svc.SyncRetention(context.Background()))

	body, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(body)
	assert.Contains(t, content, "retention_period: 90d", "expected the default 90-day retention to be written")
}

func TestSystemLogService_SyncRetention_ReflectsUpdatedSetting(t *testing.T) {
	repo := newMockSystemLogRepo()
	settingRepo, settingSvc := newTestSettingStack(t)
	retDays := "30"
	_, err := settingSvc.Patch(context.Background(), 1, dto.PatchSettingsRequest{SystemLogRetentionDays: &retDays})
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "runtime-overrides.yaml")
	svc := service.NewSystemLogService(repo, settingRepo, path, zap.NewNop())
	require.NoError(t, svc.SyncRetention(context.Background()))

	body, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(body), "retention_period: 30d")
}
