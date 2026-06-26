// Seed populates a wizard-complete Mansooba database with demo data.
// Usage: go run ./cmd/seed (from the backend/ directory)
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/sharique/mansooba/internal/repository"
	"github.com/sharique/mansooba/internal/seed"
	"github.com/sharique/mansooba/pkg/config"
	"github.com/sharique/mansooba/pkg/database"
	"github.com/sharique/mansooba/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	logger.Init(cfg.LogLevel)
	log := logger.Logger

	db, err := database.Open(cfg)
	if err != nil {
		log.Fatal("failed to open database", zap.Error(err))
	}
	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	if err := database.Migrate(db); err != nil {
		log.Fatal("failed to run migrations", zap.Error(err))
	}

	ctx := context.Background()
	userRepo := repository.NewUserRepository(db)

	admin, err := userRepo.FindFirstAdmin(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: run the setup wizard first (no admin user found)")
		os.Exit(1)
	}

	result, err := seed.Seed(ctx, db, admin.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: seed failed: %v\n", err)
		os.Exit(1)
	}

	if result.Skipped {
		fmt.Println("Seed data already present — skipping.")
		return
	}

	fmt.Println("Seed data created:")
	fmt.Printf("  Project:  %s [%s]\n", result.ProjectName, result.ProjectKey)
	fmt.Println("  Sprint:   Sprint 1 (active)")
	fmt.Printf("  Issues: %d  Labels: %d  Comments: %d\n", result.IssuesCreated, result.LabelsCreated, result.CommentsCreated)
}
