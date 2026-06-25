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

	"golang.org/x/time/rate"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	_ "github.com/sharique/mansooba/docs"
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
	projectRepo := repository.NewProjectRepository(db)
	projectMemberRepo := repository.NewProjectMemberRepository(db)
	issueRepo := repository.NewIssueRepository(db)
	sprintRepo := repository.NewSprintRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	activityRepo := repository.NewActivityRepository(db)
	notifRepo := repository.NewNotificationRepository(db)
	settingRepo := repository.NewSettingRepository(db)
	issueRelationRepo := repository.NewIssueRelationRepository(db)

	// Services
	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	userSvc := service.NewUserService(userRepo)
	projectSvc := service.NewProjectService(projectRepo, projectMemberRepo, userRepo, issueRepo)
	activitySvc := service.NewActivityService(activityRepo, userRepo, issueRepo)
	issueSvc := service.NewIssueService(issueRepo, projectRepo, projectMemberRepo, activitySvc, userRepo)
	boardSvc := service.NewBoardService(issueRepo, projectRepo, projectMemberRepo)
	sprintSvc := service.NewSprintService(sprintRepo, issueRepo, projectRepo, projectMemberRepo)
	commentSvc := service.NewCommentService(commentRepo, issueRepo, projectMemberRepo, activitySvc, notifRepo, userRepo)
	labelSvc := service.NewLabelService(repository.NewLabelRepository(db), issueRepo, projectRepo, projectMemberRepo, activitySvc)
	settingSvc := service.NewSettingService(settingRepo)
	issueRelationSvc := service.NewIssueRelationService(issueRelationRepo, issueRepo)

	// Handlers
	healthHandler := handler.NewHealthHandler(sqlDB)
	authHandler := handler.NewAuthHandler(authSvc)
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

	// Echo
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Validator = &customValidator{validator.New()}
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
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)
	auth.POST("/refresh", authHandler.Refresh)

	// Protected routes
	api := e.Group("/api/v1", apimw.JWTAuth(cfg.JWTSecret))

	authMe := api.Group("/auth")
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

	// Block until SIGTERM or SIGINT.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
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
