// @title           jira-go API
// @version         1.0
// @description     Mini Jira clone REST API
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
package main

import (
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	_ "github.com/sharique/jira-go/docs"
	"github.com/sharique/jira-go/internal/handler"
	apimw "github.com/sharique/jira-go/internal/middleware"
	"github.com/sharique/jira-go/internal/repository"
	"github.com/sharique/jira-go/internal/service"
	"github.com/sharique/jira-go/pkg/apierror"
	"github.com/sharique/jira-go/pkg/config"
	"github.com/sharique/jira-go/pkg/database"
	"github.com/sharique/jira-go/pkg/logger"
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

	// Services
	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	projectSvc := service.NewProjectService(projectRepo, projectMemberRepo, userRepo, issueRepo)
	issueSvc := service.NewIssueService(issueRepo, projectRepo, projectMemberRepo)
	boardSvc := service.NewBoardService(issueRepo, projectRepo, projectMemberRepo)
	sprintSvc := service.NewSprintService(sprintRepo, issueRepo, projectRepo, projectMemberRepo)

	// Handlers
	authHandler := handler.NewAuthHandler(authSvc)
	projectHandler := handler.NewProjectHandler(projectSvc)
	issueHandler := handler.NewIssueHandler(issueSvc)
	boardHandler := handler.NewBoardHandler(boardSvc)
	sprintHandler := handler.NewSprintHandler(sprintSvc)

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

	// Public routes
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	auth := e.Group("/api/v1/auth")
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)
	auth.POST("/refresh", authHandler.Refresh)

	// Protected routes
	api := e.Group("/api/v1", apimw.JWTAuth(cfg.JWTSecret))
	projects := api.Group("/projects")
	projects.GET("", projectHandler.List)
	projects.POST("", projectHandler.Create)
	projects.GET("/:key", projectHandler.Get)
	projects.PUT("/:key", projectHandler.Update)
	projects.DELETE("/:key", projectHandler.Delete)
	projects.GET("/:key/members", projectHandler.ListMembers)
	projects.POST("/:key/members", projectHandler.AddMember)
	projects.DELETE("/:key/members/:userId", projectHandler.RemoveMember)

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

	e.GET("/swagger/*", echoSwagger.WrapHandler)

	addr := ":" + cfg.ServerPort
	log.Info("starting server", zap.String("address", addr))
	if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
		log.Fatal("server error", zap.Error(err))
	}
}
