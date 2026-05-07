package database

import (
	"fmt"

	"github.com/sharique/jira-go/pkg/config"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Open returns a *gorm.DB connection using the driver specified in cfg.
// Currently only "sqlite" is supported; "postgres" will be wired in a later task.
func Open(cfg *config.Config) (*gorm.DB, error) {
	switch cfg.DBDriver {
	case "sqlite":
		db, err := gorm.Open(sqlite.Open(cfg.DBDSN), &gorm.Config{})
		if err != nil {
			return nil, fmt.Errorf("database: failed to open sqlite at %q: %w", cfg.DBDSN, err)
		}
		return db, nil
	default:
		return nil, fmt.Errorf("database: unsupported driver %q", cfg.DBDriver)
	}
}
