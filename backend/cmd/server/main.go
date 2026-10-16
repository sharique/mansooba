// @title           jira-go API
// @version         1.0
// @description     Mini Jira clone REST API
// @BasePath        /api/v1
// @license.name    AGPL-3.0-only
// @license.url     https://www.gnu.org/licenses/agpl-3.0.html
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/sharique/mansooba/docs"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/httpserver"
	"github.com/sharique/mansooba/internal/pkg/attachmentstorage"
	"github.com/sharique/mansooba/internal/pkg/avatarstorage"
	"github.com/sharique/mansooba/internal/pkg/lokiclient"
	"github.com/sharique/mansooba/internal/pkg/rdsclient"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/sharique/mansooba/internal/service"
	"github.com/sharique/mansooba/pkg/config"
	"github.com/sharique/mansooba/pkg/database"
	"github.com/sharique/mansooba/pkg/logger"
	"go.uber.org/zap"
)

// version is baked in at image build time via -ldflags -X main.version=...
// "dev" for a local build.
var version = "dev"

func main() {
	cfg := config.Load()

	logger.Init(cfg.LogLevel)
	defer logger.Sync()
	log := logger.Logger

	sourceCodeURL, err := config.ValidateSourceCodeURL(cfg.SourceCodeURL)
	if err != nil {
		log.Fatal("invalid SOURCE_CODE_URL", zap.Error(err))
	}
	log.Info("instance", zap.String("version", version), zap.String("source_url", sourceCodeURL))

	// Create the application context early so long-running goroutines can stop
	// cleanly when the server receives SIGTERM/SIGINT.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(cfg)
	if err != nil {
		log.Fatal("failed to open database", zap.Error(err))
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("failed to get underlying sql.DB", zap.Error(err))
	}
	defer sqlDB.Close()

	log.Info("database connection established", zap.String("driver", cfg.DBDriver))

	if err := database.Migrate(db); err != nil {
		log.Fatal("failed to run migrations", zap.Error(err))
	}
	log.Info("migrations applied")

	// Repositories main itself still needs (background jobs and the System
	// Logs service); everything request-facing is wired by httpserver.NewHandlers.
	revokedTokenRepo := repository.NewRevokedTokenRepository(db)
	settingRepo := repository.NewSettingRepository(db)
	passwordResetRepo := repository.NewPasswordResetRepository(db)

	// Attachment storage (S3-compatible: real AWS S3 in prod, LocalStack in
	// local dev — same code path either way, only STORAGE_ENDPOINT differs).
	presignTTL, err := time.ParseDuration(cfg.StoragePresignTTL)
	if err != nil {
		log.Warn("invalid STORAGE_PRESIGN_TTL, defaulting to 1h", zap.String("value", cfg.StoragePresignTTL), zap.Error(err))
		presignTTL = time.Hour
	}
	attachmentStorage, err := attachmentstorage.New(attachmentstorage.Config{
		Endpoint:        cfg.StorageEndpoint,
		PresignEndpoint: cfg.StoragePresignEndpoint,
		Bucket:          cfg.StorageBucket,
		Region:          cfg.StorageRegion,
		AccessKeyID:     cfg.StorageAccessKeyID,
		SecretAccessKey: cfg.StorageSecretAccessKey,
		PresignTTL:      presignTTL,
		UsePathStyle:    cfg.StorageUsePathStyle,
	})
	if err != nil {
		log.Fatal("failed to initialize attachment storage", zap.Error(err))
	}

	// Avatars share the attachments bucket (and its S3 client) under the
	// avatars/ prefix — see ADR-033.
	avatarStorage := avatarstorage.New(attachmentStorage.Client(), cfg.StorageBucket)

	// System Logs backing store (011-system-logs, ADR-031) — Grafana Loki,
	// not the application database. Constructed unconditionally (unlike RDS
	// auto-stop, this feature has no per-deployment opt-out): if Loki is
	// unreachable in a given environment, Record calls simply fail
	// best-effort (FR-012) and are logged as warnings, with no special
	// nil-handling needed at any call site.
	lokiClient := lokiclient.New(cfg.LokiBaseURL)
	systemLogRepo := repository.NewLokiSystemLogRepository(lokiClient)
	systemLogSvc := service.NewSystemLogService(systemLogRepo, settingRepo, cfg.LokiRuntimeOverridesPath, log)

	// Keep Loki's runtime-overrides file in sync with the admin-configured
	// system_log_retention_days setting (FR-010, SC-005) — re-read every
	// tick rather than cached at startup, so a retention change takes effect
	// without a restart. Mirrors startRevokedTokenCleanup's ticker shape.
	logRetentionSyncInterval, err := time.ParseDuration(cfg.LokiRetentionSyncInterval)
	if err != nil {
		log.Warn("invalid LOKI_RETENTION_SYNC_INTERVAL, defaulting to 1m",
			zap.String("value", cfg.LokiRetentionSyncInterval), zap.Error(err))
		logRetentionSyncInterval = time.Minute
	}
	// Sync once at startup — a time.Ticker's first tick only fires after a
	// full interval, and the runtime-overrides file starts empty (T001), so
	// without this the configured retention wouldn't apply until the first tick.
	if err := systemLogSvc.SyncRetention(ctx); err != nil {
		log.Warn("initial system_log_retention_sync failed", zap.Error(err))
	}
	startSystemLogRetentionSync(ctx, systemLogSvc, logRetentionSyncInterval, log)

	// Start background goroutine to purge expired revocation records.
	cleanupInterval, err := time.ParseDuration(cfg.RevokedTokenCleanupInterval)
	if err != nil {
		log.Warn("invalid REVOKED_TOKEN_CLEANUP_INTERVAL, defaulting to 15m",
			zap.String("value", cfg.RevokedTokenCleanupInterval), zap.Error(err))
		cleanupInterval = 15 * time.Minute
	}
	startRevokedTokenCleanup(ctx, revokedTokenRepo, cleanupInterval, log)

	// Database idle auto-stop / wake-on-hit (spec 010, db-idle-autostop; see
	// docs/decisions/ADR-030 and docs/superpowers/specs/2026-07-20-rds-autostop-detection-design.md
	// in the docs repo). Config.RDSAutoStopApplies() confirms DB_DSN's hostname
	// is the *specific* AWS RDS instance named by RDS_INSTANCE_IDENTIFIER — not
	// just "some database using a driver AWS RDS also happens to support."
	// This matters because DB_DRIVER=postgres alone doesn't mean "this is the
	// AWS demo deployment": a contributor running local Postgres via
	// docker-compose (a supported dev path, docs/running-locally-using-docker.md)
	// is also DB_DRIVER=postgres, but has no real RDS instance to describe.
	// Local dev (SQLite or local Postgres/MySQL/MariaDB) never touches this
	// code path and needs no AWS credentials. dbLifecycleTracker and rdsClient
	// stay nil when disabled; startDBIdleCheck (US1) and the dbwake middleware
	// (US2) both check for nil before doing anything.
	var dbLifecycleTracker *service.DBLifecycleTracker
	var rdsClient *rdsclient.Client
	if cfg.RDSAutoStopApplies() {
		idleTimeout, err := time.ParseDuration(cfg.RDSIdleTimeout)
		if err != nil {
			log.Warn("invalid RDS_IDLE_TIMEOUT, defaulting to 10m", zap.String("value", cfg.RDSIdleTimeout), zap.Error(err))
			idleTimeout = 10 * time.Minute
		}
		idleCheckInterval, err := time.ParseDuration(cfg.RDSIdleCheckInterval)
		if err != nil {
			log.Warn("invalid RDS_IDLE_CHECK_INTERVAL, defaulting to 1m", zap.String("value", cfg.RDSIdleCheckInterval), zap.Error(err))
			idleCheckInterval = time.Minute
		}

		rdsClient, err = rdsclient.New(ctx, cfg.RDSInstanceIdentifier)
		if err != nil {
			log.Fatal("failed to initialize RDS client", zap.Error(err))
		}

		// Fail fast at startup if permissions/credentials are misconfigured
		// (FR-014/FR-015's fail-fast requirement, spec.md Edge Cases) — better
		// to fail loudly at boot than silently behave as if disabled. The
		// returned state also seeds dbLifecycleTracker below (SeedState) —
		// the backend can restart while the database is legitimately
		// stopped (that's the whole point of this feature), so a tracker
		// that defaulted to "running" regardless of reality would silently
		// never trigger a wake.
		initialState, err := rdsClient.DescribeState(ctx)
		if err != nil {
			log.Fatal("RDS auto-stop is enabled but the configured instance could not be described — check RDS_INSTANCE_IDENTIFIER and IAM permissions",
				zap.String("instance", cfg.RDSInstanceIdentifier), zap.Error(err))
		}

		dbLifecycleTracker = service.NewDBLifecycleTracker(idleTimeout, cfg.RDSStartFailureBound, time.Now)
		dbLifecycleTracker.SeedState(initialState)
		startDBIdleCheck(ctx, dbLifecycleTracker, rdsClient, idleCheckInterval, log, systemLogSvc)
		log.Info("db idle auto-stop enabled",
			zap.String("instance", cfg.RDSInstanceIdentifier),
			zap.Duration("idle_timeout", idleTimeout),
			zap.String("initial_state", initialState.String()))
	} else {
		log.Info("db idle auto-stop disabled",
			zap.String("driver", cfg.DBDriver),
			zap.Bool("flag_enabled", cfg.RDSAutoStopEnabled),
			zap.Bool("instance_configured", cfg.RDSInstanceIdentifier != ""),
			zap.String("dsn_host", cfg.RDSHostForDiagnostics()))
	}

	// Handlers: repositories, services and handlers are wired in httpserver.
	handlers := httpserver.NewHandlers(httpserver.Inputs{
		Cfg: cfg, Log: log, DB: db, SQLDB: sqlDB, Loki: lokiClient, SystemLogSvc: systemLogSvc,
		AttachmentStorage: attachmentStorage, AvatarStorage: avatarStorage,
		Version: version, SourceCodeURL: sourceCodeURL,
	})

	// Echo: validator, error handler, middleware and routes.
	e := httpserver.New(httpserver.Deps{
		Cfg: cfg, Log: log, H: handlers,
		Tracker: dbLifecycleTracker, RDS: rdsClient, SystemLogSvc: systemLogSvc,
	})

	shutdownDuration, err := time.ParseDuration(cfg.ShutdownTimeout)
	if err != nil {
		shutdownDuration = 30 * time.Second
	}

	// Start the server in a goroutine so main() can wait for OS signals.
	// Serve returns once the graceful shutdown (triggered by ctx) has finished.
	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		addr := ":" + cfg.ServerPort
		log.Info("starting server", zap.String("address", addr))
		err := httpserver.Serve(ctx, e, addr, shutdownDuration, log, nil)
		switch {
		case err == nil:
			log.Info("server shutdown complete")
		case errors.Is(err, httpserver.ErrShutdown):
			// already logged by Serve
		default:
			log.Fatal("server error", zap.Error(err))
		}
	}()

	// Purge unused password-reset tokens every 15 minutes.
	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				n, err := passwordResetRepo.PurgeExpired(ctx, time.Now().Add(-1*time.Hour))
				if err != nil {
					log.Warn("purge password reset tokens failed", zap.Error(err))
				} else if n > 0 {
					log.Info("purged expired password reset tokens", zap.Int64("count", n))
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Block until SIGTERM or SIGINT, then wait for the server to drain.
	<-ctx.Done()
	log.Info("shutdown signal received — draining requests")
	<-serverDone
}

// startDBIdleCheck launches a background goroutine that periodically checks
// whether the database has been idle long enough to stop, and — while a
// start is pending — polls for it having become available again (spec 010,
// db-idle-autostop). Mirrors startRevokedTokenCleanup's ticker/select/
// ctx.Done() shape. The goroutine stops when ctx is cancelled.
func startDBIdleCheck(ctx context.Context, tracker *service.DBLifecycleTracker, client *rdsclient.Client, interval time.Duration, log *zap.Logger, systemLogSvc service.SystemLogService) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				attempted, stopErr := tracker.CheckAndStop(ctx, client)
				if attempted {
					if stopErr != nil {
						service.LogDBLifecycleEvent(ctx, log, systemLogSvc, "db_auto_stop", "idle_timeout", "failed", stopErr)
					} else {
						service.LogDBLifecycleEvent(ctx, log, systemLogSvc, "db_auto_stop", "idle_timeout", "succeeded", nil)
					}
				}

				justStarted, pollErr := tracker.CheckStartProgress(ctx, client)
				if pollErr != nil {
					log.Warn("db_auto_start poll failed", zap.Error(pollErr))
				} else if justStarted {
					service.LogDBLifecycleEvent(ctx, log, systemLogSvc, "db_auto_start", "incoming_request", "succeeded", nil)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

// startSystemLogRetentionSync launches a background goroutine that
// periodically reconciles Loki's runtime-overrides file with the current
// system_log_retention_days setting (011-system-logs, FR-010, SC-005).
// Mirrors startRevokedTokenCleanup's ticker/select/ctx.Done() shape. The
// goroutine stops when ctx is cancelled.
func startSystemLogRetentionSync(ctx context.Context, svc service.SystemLogService, interval time.Duration, log *zap.Logger) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := svc.SyncRetention(ctx); err != nil {
					log.Warn("system_log_retention_sync failed", zap.Error(err))
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

// startRevokedTokenCleanup launches a background goroutine that periodically
// deletes expired rows from the revoked_tokens table.
// The goroutine stops when ctx is cancelled (i.e., on SIGTERM/SIGINT).
func startRevokedTokenCleanup(ctx context.Context, repo domain.RevokedTokenRepository, interval time.Duration, log *zap.Logger) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				n, err := repo.DeleteExpired(ctx)
				if err != nil {
					log.Warn("revoked_token_cleanup failed", zap.Error(err))
				} else {
					log.Info("revoked_token_cleanup", zap.Int64("deleted", n))
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
