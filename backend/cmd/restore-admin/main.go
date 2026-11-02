// Restore-admin gives an installation back a system administrator. Run it on the
// server when the administrator accounts exist but none of them can sign in (all
// disabled, or the flag was removed some other way). It enables the account with
// the given email address and makes it a system administrator, and writes the
// action to the system log.
//
// It is a one-off admin process in the sense of constitution principle VI.12: it
// shares the application's packages and configuration and is run beside the
// server. The setup wizard is deliberately not reopened for this case, because
// that would let a stranger claim an installation whose administrators are only
// disabled.
//
// Usage:  ./restore-admin --email you@example.com
//
//	go run ./cmd/restore-admin --email you@example.com   (from backend/)
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/pkg/lokiclient"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/sharique/mansooba/internal/service"
	"github.com/sharique/mansooba/pkg/config"
	"github.com/sharique/mansooba/pkg/database"
	"github.com/sharique/mansooba/pkg/logger"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// systemLog is the part of the system log service this command needs.
type systemLog interface {
	Record(ctx context.Context, entry domain.SystemLogEntry)
}

func run(args []string, out, errOut io.Writer) (code int) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(errOut, "restore-admin: configuration error: %v\n", r)
			code = 1
		}
	}()
	fs := flag.NewFlagSet("restore-admin", flag.ContinueOnError)
	fs.SetOutput(errOut)
	email := fs.String("email", "", "email address of the account to enable and make a system administrator")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *email == "" {
		fmt.Fprintln(errOut, "restore-admin: --email is required")
		return 2
	}

	cfg := config.Load()
	logger.Init(cfg.LogLevel)
	db, err := database.Open(cfg)
	if err != nil {
		fmt.Fprintf(errOut, "restore-admin: open database: %v\n", err)
		return 1
	}
	if sqlDB, err := db.DB(); err == nil {
		defer sqlDB.Close()
	}

	settingRepo := repository.NewSettingRepository(db)
	sysLog := service.NewSystemLogService(
		repository.NewLokiSystemLogRepository(lokiclient.New(cfg.LokiBaseURL)),
		settingRepo, cfg.LokiRuntimeOverridesPath, logger.Logger)
	return restore(context.Background(), repository.NewUserRepository(db), sysLog, *email, out, errOut)
}

// restore enables the account and makes it a system administrator. It returns
// the process exit code.
func restore(ctx context.Context, users domain.UserRepository, log systemLog, email string, out, errOut io.Writer) int {
	u, err := users.FindByEmail(ctx, email)
	if err != nil {
		fmt.Fprintf(errOut, "restore-admin: no account with the email address %s\n", email)
		return 1
	}
	yes := true
	if err := users.UpdateAdminFieldsGuarded(ctx, u.ID, domain.AdminChange{IsAdmin: &yes, IsActive: &yes}); err != nil {
		fmt.Fprintf(errOut, "restore-admin: %v\n", err)
		return 1
	}
	log.Record(ctx, domain.SystemLogEntry{
		Category: domain.SystemLogCategoryAdminAction,
		Action:   "system_admin_restored",
		Outcome:  "success",
		Actor:    "restore-admin (command line)",
		Target:   u.Email,
		TargetID: u.ID,
		Detail:   "account enabled and made a system administrator",
	})
	fmt.Fprintf(out, "restore-admin: %s is now an active system administrator\n", u.Email)
	return 0
}
