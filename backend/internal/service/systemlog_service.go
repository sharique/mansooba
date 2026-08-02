package service

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/sharique/mansooba/internal/domain"
)

// lokiDefaultTenant is the tenant ID Loki uses internally when
// auth_enabled is false (this project's local-config.yaml default,
// single-tenant mode) and no X-Scope-OrgID header is sent — which
// lokiclient never sends. Runtime overrides must target this tenant to
// actually apply.
const lokiDefaultTenant = "fake"

// systemLogRetentionFallbackDays is used only if the retention setting is
// absent or unparsable — defensive; validateSettingValue in
// setting_service.go should already prevent this from occurring.
const systemLogRetentionFallbackDays = 90

// SystemLogService records and queries System Log entries (011-system-logs)
// and keeps Loki's retention configuration in sync with the admin-configured
// setting.
type SystemLogService interface {
	// Record durably logs entry, best-effort (FR-012): it never returns an
	// error to the caller, and the underlying push happens in a
	// fire-and-forget goroutine so it never blocks or delays the caller. A
	// push failure is caught internally and logged via zap.Warn — the
	// application's existing process-level logging — so it's diagnosable by
	// an operator even though it never reaches System Logs itself.
	Record(ctx context.Context, entry domain.SystemLogEntry)
	// List returns a page of entries matching filter (FR-006, FR-007).
	List(ctx context.Context, filter domain.SystemLogListFilter) (domain.SystemLogListResult, error)
	// SyncRetention reconciles Loki's runtime-overrides file with the
	// current system_log_retention_days setting (FR-010, SC-005). It only
	// writes *configuration* — Loki's own compactor performs the actual
	// expiry (research.md Decision 3).
	SyncRetention(ctx context.Context) error
}

type systemLogServiceImpl struct {
	repo                 domain.SystemLogRepository
	settingRepo          domain.GlobalSettingRepository
	runtimeOverridesPath string
	log                  *zap.Logger
}

// NewSystemLogService returns a SystemLogService. runtimeOverridesPath MUST
// be the exact path Loki was started with via -runtime-config.file (T001) —
// otherwise SyncRetention's writes are never picked up.
//
// Depends on domain.GlobalSettingRepository directly, not SettingService —
// SettingService.Patch itself calls SystemLogService.Record (FR-003, every
// settings change is auditable), so depending on the service instead of the
// repository here would create an import cycle: SettingService ->
// SystemLogService -> SettingService.
func NewSystemLogService(repo domain.SystemLogRepository, settingRepo domain.GlobalSettingRepository, runtimeOverridesPath string, log *zap.Logger) SystemLogService {
	return &systemLogServiceImpl{
		repo:                 repo,
		settingRepo:          settingRepo,
		runtimeOverridesPath: runtimeOverridesPath,
		log:                  log,
	}
}

func (s *systemLogServiceImpl) Record(ctx context.Context, entry domain.SystemLogEntry) {
	go func() {
		// A goroutine outliving the caller's request must not inherit a
		// context that's about to be cancelled when that request completes —
		// context.Background() is deliberate here, not an oversight.
		bgCtx := context.Background()
		if entry.CreatedAt.IsZero() {
			entry.CreatedAt = time.Now()
		}
		if err := s.repo.Create(bgCtx, entry); err != nil {
			s.log.Warn("system_log_write_failed",
				zap.String("category", entry.Category),
				zap.String("action", entry.Action),
				zap.Error(err))
		}
	}()
}

// List resolves an unbounded query (no From) to the current retention
// window before delegating to the repository, rather than letting the
// repository's own generous safety-cap default (defaultLookback in
// loki_systemlog_repository.go) apply. Live testing against a real Loki
// instance showed that widening the query range directly widens latency
// (roughly linear with days queried, since the filesystem-backed TSDB
// index is sharded per day) — an unfiltered query defaulting to a range
// far past the actual retention window scans for entries that provably
// can't exist, for nothing. Bounding it to retention keeps the common
// "no filter" case fast while still honoring FR-006 ("no filter = all
// retained entries").
func (s *systemLogServiceImpl) List(ctx context.Context, filter domain.SystemLogListFilter) (domain.SystemLogListResult, error) {
	if filter.From == nil {
		days, err := s.retentionDays(ctx)
		if err != nil {
			return domain.SystemLogListResult{}, fmt.Errorf("systemlog_service: resolve retention for default range: %w", err)
		}
		from := time.Now().AddDate(0, 0, -days)
		filter.From = &from
	}
	return s.repo.FindPaginated(ctx, filter)
}

// retentionDays reads system_log_retention_days, defaulting to
// systemLogRetentionFallbackDays when absent or unparsable.
func (s *systemLogServiceImpl) retentionDays(ctx context.Context) (int, error) {
	rows, err := s.settingRepo.FindAll(ctx)
	if err != nil {
		return 0, fmt.Errorf("get settings: %w", err)
	}

	days := systemLogRetentionFallbackDays
	for _, row := range rows {
		if row.SettingKey != domain.SettingKeySystemLogRetentionDays {
			continue
		}
		if n, err := strconv.Atoi(row.SettingValue); err == nil && n >= 1 {
			days = n
		}
		break
	}
	return days, nil
}

func (s *systemLogServiceImpl) SyncRetention(ctx context.Context) error {
	days, err := s.retentionDays(ctx)
	if err != nil {
		return fmt.Errorf("systemlog_service: %w", err)
	}

	overrides := lokiRuntimeOverrides{
		Overrides: map[string]lokiTenantOverride{
			lokiDefaultTenant: {RetentionPeriod: fmt.Sprintf("%dd", days)},
		},
	}
	buf, err := yaml.Marshal(overrides)
	if err != nil {
		return fmt.Errorf("systemlog_service: marshal runtime overrides: %w", err)
	}

	if err := os.WriteFile(s.runtimeOverridesPath, buf, 0o644); err != nil {
		return fmt.Errorf("systemlog_service: write runtime overrides file: %w", err)
	}
	return nil
}

// lokiRuntimeOverrides mirrors the shape Loki's -runtime-config.file expects
// for per-tenant overrides (Loki docs: "Runtime Configuration File").
type lokiRuntimeOverrides struct {
	Overrides map[string]lokiTenantOverride `yaml:"overrides"`
}

type lokiTenantOverride struct {
	RetentionPeriod string `yaml:"retention_period"`
}
