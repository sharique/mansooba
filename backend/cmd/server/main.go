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

	"golang.org/x/time/rate"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	_ "github.com/sharique/mansooba/docs"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/email"
	"github.com/sharique/mansooba/internal/handler"
	apimw "github.com/sharique/mansooba/internal/middleware"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/sharique/mansooba/internal/service"
	"github.com/sharique/mansooba/pkg/apierror"
	"github.com/sharique/mansooba/pkg/config"
	"github.com/sharique/mansooba/pkg/database"
	"github.com/sharique/mansooba/pkg/logger"
	echoSwagger "github.com/swaggo/echo-swagger"
	"go.uber.org/zap"
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

	// Start background goroutine to purge expired revocation records.
	cleanupInterval, err := time.ParseDuration(cfg.RevokedTokenCleanupInterval)
	if err != nil {
		log.Warn("invalid REVOKED_TOKEN_CLEANUP_INTERVAL, defaulting to 15m",
			zap.String("value", cfg.RevokedTokenCleanupInterval), zap.Error(err))
		cleanupInterval = 15 * time.Minute
	}
	startRevokedTokenCleanup(ctx, revokedTokenRepo, cleanupInterval, log)

	// Services
	authSvc := service.NewAuthService(userRepo, revokedTokenRepo, log, cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	userSvc := service.NewUserService(userRepo)
	projectSvc := service.NewProjectService(projectRepo, projectMemberRepo, userRepo, issueRepo)
	activitySvc := service.NewActivityService(activityRepo, userRepo, issueRepo)
	issueSvc := service.NewIssueService(issueRepo, projectRepo, projectMemberRepo, activitySvc, userRepo, sprintRepo)
	boardSvc := service.NewBoardService(issueRepo, projectRepo, projectMemberRepo)
	sprintSvc := service.NewSprintService(sprintRepo, issueRepo, projectRepo, projectMemberRepo)
	commentSvc := service.NewCommentService(commentRepo, issueRepo, projectMemberRepo, activitySvc, notifRepo, userRepo)
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
