// @title           jira-go API
// @version         1.0
// @description     Mini Jira clone REST API
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"unicode"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	_ "github.com/sharique/mansooba/docs"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/email"
	"github.com/sharique/mansooba/internal/handler"
	apimw "github.com/sharique/mansooba/internal/middleware"
	"github.com/sharique/mansooba/internal/pkg/attachmentstorage"
	"github.com/sharique/mansooba/internal/pkg/rdsclient"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/sharique/mansooba/internal/service"
	"github.com/sharique/mansooba/pkg/apierror"
	"github.com/sharique/mansooba/pkg/config"
	"github.com/sharique/mansooba/pkg/database"
	"github.com/sharique/mansooba/pkg/logger"
	echoSwagger "github.com/swaggo/echo-swagger"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type customValidator struct{ v *validator.Validate }

func (cv *customValidator) Validate(i any) error { return cv.v.Struct(i) }

func main() {
	cfg := config.Load()

	logger.Init(cfg.LogLevel)
	defer logger.Sync()
	log := logger.Logger

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

	// Repositories
	userRepo := repository.NewUserRepository(db)
	revokedTokenRepo := repository.NewRevokedTokenRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	projectMemberRepo := repository.NewProjectMemberRepository(db)
	issueRepo := repository.NewIssueRepository(db)
	sprintRepo := repository.NewSprintRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	activityRepo := repository.NewActivityRepository(db)
	notifRepo := repository.NewNotificationRepository(db)
	settingRepo := repository.NewSettingRepository(db)
	issueRelationRepo := repository.NewIssueRelationRepository(db)
	passwordResetRepo := repository.NewPasswordResetRepository(db)
	attachmentRepo := repository.NewAttachmentRepository(db)

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
		// to fail loudly at boot than silently behave as if disabled.
		if _, err := rdsClient.DescribeState(ctx); err != nil {
			log.Fatal("RDS auto-stop is enabled but the configured instance could not be described — check RDS_INSTANCE_IDENTIFIER and IAM permissions",
				zap.String("instance", cfg.RDSInstanceIdentifier), zap.Error(err))
		}

		dbLifecycleTracker = service.NewDBLifecycleTracker(idleTimeout, cfg.RDSStartFailureBound, time.Now)
		startDBIdleCheck(ctx, dbLifecycleTracker, rdsClient, idleCheckInterval, log)
		log.Info("db idle auto-stop enabled", zap.String("instance", cfg.RDSInstanceIdentifier), zap.Duration("idle_timeout", idleTimeout))
	} else {
		log.Info("db idle auto-stop disabled",
			zap.String("driver", cfg.DBDriver),
			zap.Bool("flag_enabled", cfg.RDSAutoStopEnabled),
			zap.Bool("instance_configured", cfg.RDSInstanceIdentifier != ""))
	}

	// Services
	authSvc := service.NewAuthService(userRepo, revokedTokenRepo, log, cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	userSvc := service.NewUserService(userRepo)
	projectSvc := service.NewProjectService(projectRepo, projectMemberRepo, userRepo, issueRepo)
	activitySvc := service.NewActivityService(activityRepo, userRepo, issueRepo)
	issueSvc := service.NewIssueService(issueRepo, projectRepo, projectMemberRepo, activitySvc, userRepo, sprintRepo).
		WithAttachments(attachmentRepo, attachmentStorage)
	boardSvc := service.NewBoardService(issueRepo, projectRepo, projectMemberRepo)
	sprintSvc := service.NewSprintService(sprintRepo, issueRepo, projectRepo, projectMemberRepo)
	commentSvc := service.NewCommentService(commentRepo, issueRepo, projectMemberRepo, activitySvc, notifRepo, userRepo)
	attachmentSvc := service.NewAttachmentService(attachmentRepo, issueRepo, projectRepo, projectMemberRepo, activitySvc, userRepo, attachmentStorage)
	labelSvc := service.NewLabelService(repository.NewLabelRepository(db), issueRepo, projectRepo, projectMemberRepo, activitySvc)
	settingSvc := service.NewSettingService(settingRepo)
	issueRelationSvc := service.NewIssueRelationService(issueRelationRepo, issueRepo)
	var emailSender domain.EmailSender
	if cfg.SMTPHost != "" {
		emailSender = email.SMTPSender{Host: cfg.SMTPHost, Port: cfg.SMTPPort, From: cfg.SMTPFrom, AppBaseURL: cfg.AppBaseURL}
		log.Info("email delivery enabled", zap.String("smtp_host", cfg.SMTPHost), zap.String("smtp_port", cfg.SMTPPort))
	} else {
		emailSender = email.NoopSender{}
		log.Info("email delivery disabled — token returned in API response only")
	}
	passwordResetSvc := service.NewPasswordResetService(userRepo, passwordResetRepo, emailSender)

	adminUserSvc := service.NewAdminUserService(userRepo)

	// Setup service (wizard)
	accessTTL, _ := time.ParseDuration(cfg.JWTAccessTTL)
	setupSvc := service.NewSetupService(userRepo, projectSvc, cfg.JWTSecret, accessTTL, log, db)

	// Handlers
	healthHandler := handler.NewHealthHandler(sqlDB)
	authHandler := handler.NewAuthHandler(authSvc, userSvc)
	setupHandler := handler.NewSetupHandler(setupSvc)
	userHandler := handler.NewUserHandler(userSvc, activitySvc, issueSvc)
	projectHandler := handler.NewProjectHandler(projectSvc)
	issueHandler := handler.NewIssueHandler(issueSvc)
	boardHandler := handler.NewBoardHandler(boardSvc)
	sprintHandler := handler.NewSprintHandler(sprintSvc)
	commentHandler := handler.NewCommentHandler(commentSvc)
	attachmentHandler := handler.NewAttachmentHandler(attachmentSvc)
	activityHandler := handler.NewActivityHandler(activitySvc)
	labelHandler := handler.NewLabelHandler(labelSvc)
	notifHandler := handler.NewNotificationHandler(notifRepo)
	settingHandler := handler.NewSettingHandler(settingSvc, userSvc)
	issueRelationHandler := handler.NewIssueRelationHandler(issueRelationSvc)
	issueHandler = issueHandler.WithRelationRepo(issueRelationRepo)
	adminUserHandler := handler.NewAdminUserHandler(adminUserSvc, userSvc)
	passwordResetHandler := handler.NewPasswordResetHandler(passwordResetSvc)

	// Echo
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	v := validator.New()
	v.RegisterValidation("password_complexity", func(fl validator.FieldLevel) bool { //nolint:errcheck
		pw := fl.Field().String()
		var hasUpper, hasLower, hasDigit bool
		for _, r := range pw {
			switch {
			case unicode.IsUpper(r):
				hasUpper = true
			case unicode.IsLower(r):
				hasLower = true
			case unicode.IsDigit(r):
				hasDigit = true
			}
		}
		return len(pw) >= 8 && hasUpper && hasLower && hasDigit
	})
	e.Validator = &customValidator{v}
	e.HTTPErrorHandler = apierror.HTTPErrorHandler

	e.Use(echomw.Recover())
	e.Use(echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins: strings.Split(cfg.CORSOrigins, ","),
	}))

	// Body limit — reject payloads > BodySizeLimit with 413
	e.Use(echomw.BodyLimit(cfg.BodySizeLimit))

	// Request timeout — abort requests exceeding RequestTimeout with 503
	requestTimeout, err := time.ParseDuration(cfg.RequestTimeout)
	if err != nil {
		log.Warn("invalid REQUEST_TIMEOUT value, defaulting to 30s", zap.String("value", cfg.RequestTimeout), zap.Error(err))
		requestTimeout = 30 * time.Second
	}
	e.Use(echomw.TimeoutWithConfig(echomw.TimeoutConfig{
		Timeout: requestTimeout,
	}))

	// Database idle auto-stop / wake-on-hit (spec 010) — only registered when
	// the feature is actually enabled (T009 above), so both are a complete
	// no-op otherwise, not just an inert call. DBWake runs first so a request
	// hitting a stopped database gets the waking_up signal immediately,
	// without DBActivity or any handler attempting real (doomed) DB work.
	if dbLifecycleTracker != nil {
		e.Use(apimw.DBWake(dbLifecycleTracker, rdsClient, log))
		e.Use(apimw.DBActivity(dbLifecycleTracker))
	}

	// Public routes
	e.GET("/health", healthHandler.Check)

	auth := e.Group("/api/v1/auth")
	auth.Use(echomw.RateLimiterWithConfig(echomw.RateLimiterConfig{
		Skipper: echomw.DefaultSkipper,
		Store: echomw.NewRateLimiterMemoryStoreWithConfig(
			echomw.RateLimiterMemoryStoreConfig{
				Rate:      rate.Limit(cfg.AuthRateLimit),
				Burst:     cfg.AuthRateLimit,
				ExpiresIn: 3 * time.Minute,
			},
		),
		IdentifierExtractor: func(ctx echo.Context) (string, error) {
			return ctx.RealIP(), nil
		},
		ErrorHandler: func(context echo.Context, err error) error {
			return context.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
		},
		DenyHandler: func(context echo.Context, identifier string, err error) error {
			return context.JSON(http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
		},
	}))
	auth.POST("/login", authHandler.Login)
	auth.POST("/refresh", authHandler.Refresh)
	auth.POST("/logout", authHandler.Logout)
	auth.POST("/forgot-password", passwordResetHandler.ForgotPassword)
	auth.POST("/reset-password", passwordResetHandler.ResetPassword)

	// Setup wizard routes (public group — no JWT required)
	setup := e.Group("/api/v1/setup")
	setup.GET("/status", setupHandler.Status)

	// Account-creation steps are rate-limited (same config as auth)
	setupRateLimited := setup.Group("", echomw.RateLimiterWithConfig(echomw.RateLimiterConfig{
		Skipper: echomw.DefaultSkipper,
		Store: echomw.NewRateLimiterMemoryStoreWithConfig(
			echomw.RateLimiterMemoryStoreConfig{
				Rate:      rate.Limit(cfg.AuthRateLimit),
				Burst:     cfg.AuthRateLimit,
				ExpiresIn: 3 * time.Minute,
			},
		),
		IdentifierExtractor: func(ctx echo.Context) (string, error) {
			return ctx.RealIP(), nil
		},
		ErrorHandler: func(context echo.Context, err error) error {
			return context.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
		},
		DenyHandler: func(context echo.Context, identifier string, err error) error {
			return context.JSON(http.StatusTooManyRequests, map[string]string{"error": "Too many attempts. Please wait a minute and try again."})
		},
	}))
	setupRateLimited.POST("/admin", setupHandler.CreateAdmin)

	// Authenticated setup steps (require JWT from step 1)
	setupAuth := setup.Group("", apimw.JWTAuth(cfg.JWTSecret))
	setupAuth.Use(echomw.RateLimiterWithConfig(echomw.RateLimiterConfig{
		Skipper: echomw.DefaultSkipper,
		Store: echomw.NewRateLimiterMemoryStoreWithConfig(
			echomw.RateLimiterMemoryStoreConfig{
				Rate:      rate.Limit(cfg.AuthRateLimit),
				Burst:     cfg.AuthRateLimit,
				ExpiresIn: 3 * time.Minute,
			},
		),
		IdentifierExtractor: func(ctx echo.Context) (string, error) {
			return ctx.RealIP(), nil
		},
		ErrorHandler: func(context echo.Context, err error) error {
			return context.JSON(http.StatusForbidden, map[string]string{"error": "forbidden"})
		},
		DenyHandler: func(context echo.Context, identifier string, err error) error {
			return context.JSON(http.StatusTooManyRequests, map[string]string{"error": "Too many attempts. Please wait a minute and try again."})
		},
	}))
	setupAuth.POST("/user", setupHandler.CreateUser)
	setupAuth.POST("/project", setupHandler.CreateProject)
	setupAuth.POST("/seed", setupHandler.SeedData)

	// Protected routes
	api := e.Group("/api/v1", apimw.JWTAuth(cfg.JWTSecret))

	authMe := api.Group("/auth")
	authMe.POST("/register", authHandler.Register)
	authMe.GET("/me", userHandler.GetProfile)
	authMe.PUT("/me", userHandler.UpdateProfile)
	authMe.GET("/me/activity", userHandler.GetMyActivity)
	authMe.GET("/me/issues", userHandler.GetMyIssues)
	authMe.POST("/me/avatar", userHandler.UploadAvatar)
	authMe.DELETE("/me/avatar", userHandler.DeleteAvatar)

	// Public static file serving for uploaded avatars.
	// Intentionally unauthenticated — browser <img> tags fetch images as
	// unauthenticated sub-resources. Rationale documented in ADR-026.
	e.Static("/uploads", "uploads")

	projects := api.Group("/projects")
	projects.GET("", projectHandler.List)
	projects.POST("", projectHandler.Create)
	projects.GET("/:key", projectHandler.Get)
	projects.PUT("/:key", projectHandler.Update)
	projects.DELETE("/:key", projectHandler.Delete)
	projects.GET("/:key/members", projectHandler.ListMembers)
	projects.POST("/:key/members", projectHandler.AddMember)
	projects.DELETE("/:key/members/:userId", projectHandler.RemoveMember)
	projects.GET("/:key/labels", labelHandler.List)
	projects.POST("/:key/labels", labelHandler.Create)
	projects.DELETE("/:key/labels/:lid", labelHandler.Delete)

	issues := api.Group("/projects/:key/issues")
	issues.GET("", issueHandler.List)
	issues.POST("", issueHandler.Create)
	issues.GET("/:id", issueHandler.Get)
	issues.PUT("/:id", issueHandler.Update)
	issues.DELETE("/:id", issueHandler.Delete)

	api.GET("/projects/:key/board", boardHandler.GetBoard)

	sprints := api.Group("/projects/:key/sprints")
	sprints.GET("", sprintHandler.List)
	sprints.POST("", sprintHandler.Create)
	sprints.GET("/:id", sprintHandler.Get)
	sprints.PUT("/:id", sprintHandler.Update)
	sprints.DELETE("/:id", sprintHandler.Delete)
	sprints.POST("/:id/start", sprintHandler.Start)
	sprints.POST("/:id/complete", sprintHandler.Complete)
	sprints.GET("/:id/burndown", sprintHandler.Burndown)
	sprints.GET("/:id/issues", sprintHandler.GetIssues)

	api.GET("/projects/:key/backlog", sprintHandler.Backlog)
	api.GET("/projects/:key/velocity", sprintHandler.Velocity)

	issueItems := api.Group("/issues/:id")
	issueItems.GET("/comments", commentHandler.List)
	issueItems.POST("/comments", commentHandler.Create)
	issueItems.PUT("/comments/:cid", commentHandler.Update)
	issueItems.DELETE("/comments/:cid", commentHandler.Delete)
	issueItems.GET("/activity", activityHandler.ListByIssue)
	issueItems.GET("/labels", labelHandler.ListByIssue)
	issueItems.POST("/labels/:lid", labelHandler.Attach)
	issueItems.DELETE("/labels/:lid", labelHandler.Detach)
	issueItems.GET("/relations", issueRelationHandler.List)
	issueItems.POST("/relations", issueRelationHandler.Create)
	issueItems.DELETE("/relations/:rid", issueRelationHandler.Delete)

	// Attachment upload gets a larger body-size limit than the global default
	// (research.md Decision 3) — batch uploads of multiple ~10MB files would
	// otherwise be rejected by the global 4M BodyLimit applied to every route.
	issueItems.POST("/attachments", attachmentHandler.Upload, echomw.BodyLimit("25M"))
	issueItems.GET("/attachments", attachmentHandler.List)
	issueItems.GET("/attachments/:aid/download", attachmentHandler.Download)
	issueItems.DELETE("/attachments/:aid", attachmentHandler.Delete)

	admin := api.Group("/admin")
	admin.GET("/users", adminUserHandler.ListUsers)
	admin.PATCH("/users/:id", adminUserHandler.PatchUser)

	settings := api.Group("/settings")
	settings.GET("", settingHandler.GetAll)
	settings.PATCH("", settingHandler.Patch)

	notifs := api.Group("/notifications")
	notifs.GET("", notifHandler.List)
	notifs.PUT("/:id/read", notifHandler.MarkRead)

	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// Start the server in a goroutine so main() can wait for OS signals.
	go func() {
		addr := ":" + cfg.ServerPort
		log.Info("starting server", zap.String("address", addr))
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
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

	// Block until SIGTERM or SIGINT.
	<-ctx.Done()

	log.Info("shutdown signal received — draining requests")

	shutdownDuration, err := time.ParseDuration(cfg.ShutdownTimeout)
	if err != nil {
		shutdownDuration = 30 * time.Second
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownDuration)
	defer cancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", zap.Error(err))
	} else {
		log.Info("server shutdown complete")
	}
}

// startDBIdleCheck launches a background goroutine that periodically checks
// whether the database has been idle long enough to stop, and — while a
// start is pending — polls for it having become available again (spec 010,
// db-idle-autostop). Mirrors startRevokedTokenCleanup's ticker/select/
// ctx.Done() shape. The goroutine stops when ctx is cancelled.
func startDBIdleCheck(ctx context.Context, tracker *service.DBLifecycleTracker, client *rdsclient.Client, interval time.Duration, log *zap.Logger) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				attempted, stopErr := tracker.CheckAndStop(ctx, client)
				if attempted {
					if stopErr != nil {
						service.LogDBLifecycleEvent(log, "db_auto_stop", "idle_timeout", "failed", stopErr)
					} else {
						service.LogDBLifecycleEvent(log, "db_auto_stop", "idle_timeout", "succeeded", nil)
					}
				}

				justStarted, pollErr := tracker.CheckStartProgress(ctx, client)
				if pollErr != nil {
					log.Warn("db_auto_start poll failed", zap.Error(pollErr))
				} else if justStarted {
					service.LogDBLifecycleEvent(log, "db_auto_start", "incoming_request", "succeeded", nil)
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
