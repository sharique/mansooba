package main

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"github.com/sharique/jira-go/internal/handler"
	apimw "github.com/sharique/jira-go/internal/middleware"
	"github.com/sharique/jira-go/internal/repository"
	"github.com/sharique/jira-go/internal/service"
	"github.com/sharique/jira-go/pkg/apierror"
	"github.com/sharique/jira-go/pkg/config"
	"github.com/sharique/jira-go/pkg/database"
	"github.com/sharique/jira-go/pkg/logger"
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

	// Services
	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)

	// Handlers
	authHandler := handler.NewAuthHandler(authSvc)

	// Echo
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Validator = &customValidator{validator.New()}
	e.HTTPErrorHandler = apierror.HTTPErrorHandler

	e.Use(echomw.Recover())
	e.Use(echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins: []string{cfg.CORSOrigins},
	}))

	// Public routes
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	auth := e.Group("/api/v1/auth")
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)
	auth.POST("/refresh", authHandler.Refresh)

	// Protected routes (future tasks add their handlers here)
	_ = e.Group("/api/v1", apimw.JWTAuth(cfg.JWTSecret))

	addr := ":" + cfg.ServerPort
	log.Info("starting server", zap.String("address", addr))
	if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
		log.Fatal("server error", zap.Error(err))
	}
}
