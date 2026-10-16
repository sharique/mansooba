// Backfill-usernames gives every account that has no username a generated one.
// It is the one-off admin process for the username column (constitution
// principle VI.12, ADR-038): it shares the application's packages and
// configuration, prepares and migrates the schema (like cmd/seed), is safe to
// run repeatedly, and is run by the deploy before the server starts.
// Usage: go run ./cmd/backfill-usernames (from the backend/ directory)
package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/sharique/mansooba/internal/service"
	"github.com/sharique/mansooba/pkg/config"
	"github.com/sharique/mansooba/pkg/database"
)

func main() {
	os.Exit(run(os.Stdout, os.Stderr))
}

// run does the work and returns the process exit code.
func run(out, errOut io.Writer) (code int) {
	// config.Load panics on missing required settings; report it like any
	// other failure.
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(errOut, "backfill-usernames: configuration error: %v\n", r)
			code = 1
		}
	}()
	cfg := config.Load()

	db, err := database.Open(cfg)
	if err != nil {
		fmt.Fprintf(errOut, "backfill-usernames: open database: %v\n", err)
		return 1
	}
	if sqlDB, err := db.DB(); err == nil {
		defer sqlDB.Close()
	}
	// Order matters on a database that predates usernames: add the column
	// (nullable), give every account a username, and only then migrate, which
	// makes the column NOT NULL (ADR-038). On a fresh database there is nothing
	// to backfill and Migrate creates the final schema.
	if err := database.PrepareUsernames(db); err != nil {
		fmt.Fprintf(errOut, "backfill-usernames: prepare: %v\n", err)
		return 1
	}
	svc := service.NewUsernameService(repository.NewUserRepository(db))
	changed := 0
	if db.Migrator().HasTable(&domain.User{}) {
		changed, err = svc.BackfillMissing(context.Background())
		if err != nil {
			fmt.Fprintf(errOut, "backfill-usernames: %v (%d accounts changed before the failure)\n", err, changed)
			return 1
		}
	}
	if err := database.Migrate(db); err != nil {
		fmt.Fprintf(errOut, "backfill-usernames: migrate: %v\n", err)
		return 1
	}
	fmt.Fprintf(out, "backfill-usernames: %d accounts changed\n", changed)
	return 0
}
