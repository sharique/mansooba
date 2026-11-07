package config

import (
	"errors"
	"fmt"
	"strings"
)

// sqliteGuideURL is where an operator who still has SQLite configured finds out
// what to do. It is a link into this repository, which an operator running only
// the published image can still open.
const sqliteGuideURL = "https://github.com/sharique/mansooba/blob/main/docs/deployments/moving-off-sqlite.md"

// DatabaseKind returns the canonical database family for DB_DRIVER: "postgres"
// for postgres or postgresql, "mysql" for mysql or mariadb, and "" for anything
// else (including an unset value). Matching ignores case and surrounding spaces.
func (c *Config) DatabaseKind() string {
	return normalizeDriver(strings.ToLower(strings.TrimSpace(c.DBDriver)))
}

// ValidateDatabase checks the two database settings before anything connects.
// Both are required and neither has a default, so a missing one stops the server
// with a message that names it. SQLite is refused by name. The driver is checked
// before the connection string, so a SQLite setup whose connection string is a
// file path still gets the SQLite message.
func ValidateDatabase(cfg *Config) error {
	driver := strings.ToLower(strings.TrimSpace(cfg.DBDriver))
	switch driver {
	case "":
		return errors.New("database: DB_DRIVER is not set. Set it to one of: postgres, mysql, mariadb.")
	case "sqlite", "sqlite3":
		return fmt.Errorf("database: SQLite is no longer supported. Set DB_DRIVER to one of: postgres, mysql, mariadb. See %s", sqliteGuideURL)
	}
	if normalizeDriver(driver) == "" {
		return fmt.Errorf("database: unsupported DB_DRIVER %q. Supported: postgres, mysql, mariadb.", cfg.DBDriver)
	}
	if strings.TrimSpace(cfg.DBDSN) == "" {
		return fmt.Errorf("database: DB_DSN is not set. Give the connection string for %s; see backend/.env.example.", driver)
	}
	return nil
}
