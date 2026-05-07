package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sharique/jira-go/pkg/config"
	"github.com/sharique/jira-go/pkg/database"
	"github.com/sharique/jira-go/pkg/logger"
	"go.uber.org/zap"
)

// main is the application entry point.
// It loads config, initialises the logger and database, runs migrations,
// registers routes, and starts the Echo HTTP server.
func main() {
	// Load configuration
	cfg := config.Load()

	// Initialise logger
	logger.Init(cfg.LogLevel)
	defer logger.Sync()

	log := logger.Logger

	// Open database
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

	// Create Echo instance
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Middleware
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{cfg.CORSOrigins},
	}))

	// Routes
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Start server
	addr := ":" + cfg.ServerPort
	log.Info("starting server", zap.String("address", addr))
	if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
		log.Fatal("server error", zap.Error(err))
	}
}
