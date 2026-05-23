package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/sharique/mansooba/pkg/config"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Open returns a *gorm.DB using the driver named in cfg.DBDriver.
// Supported values: sqlite, postgres, postgresql, mysql, mariadb.
// Connection pool is configured via configurePool after the connection is established.
func Open(cfg *config.Config) (*gorm.DB, error) {
	gormCfg := &gorm.Config{}
	var (
		db  *gorm.DB
		err error
	)

	switch cfg.DBDriver {
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(cfg.DBDSN), gormCfg)
	case "postgres", "postgresql":
		db, err = gorm.Open(postgres.Open(cfg.DBDSN), gormCfg)
	case "mysql", "mariadb":
		db, err = gorm.Open(mysql.Open(cfg.DBDSN), gormCfg)
	default:
		return nil, fmt.Errorf("database: unsupported driver %q (supported: sqlite, postgres, mysql, mariadb)", cfg.DBDriver)
	}

	if err != nil {
		return nil, fmt.Errorf("database: failed to open %q: %w", cfg.DBDriver, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("database: failed to get sql.DB: %w", err)
	}

	if err := configurePool(sqlDB, cfg); err != nil {
		return nil, err
	}

	return db, nil
}

// configurePool sets connection pool parameters on sqlDB.
//
// SQLite is always capped at MaxOpenConns=1 to prevent "database is locked" errors.
// SQLite uses file-level write locks; a second concurrent writer gets SQLITE_BUSY immediately.
//
// For postgres/mysql the pool is set from cfg, allowing production tuning via env vars.
// DB_MAX_OPEN_CONNS=0 means unlimited (Go default); DB_MAX_IDLE_CONNS=2 by default.
func configurePool(sqlDB *sql.DB, cfg *config.Config) error {
	maxOpen := cfg.DBMaxOpenConns
	maxIdle := cfg.DBMaxIdleConns

	if cfg.DBDriver == "sqlite" {
		// Enforce single connection for SQLite regardless of config.
		maxOpen = 1
		if maxIdle == 0 {
			maxIdle = 1
		}
	}

	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)

	if cfg.DBConnMaxLifetime != "" && cfg.DBConnMaxLifetime != "0" {
		d, err := time.ParseDuration(cfg.DBConnMaxLifetime)
		if err != nil {
			return fmt.Errorf("database: invalid DB_CONN_MAX_LIFETIME %q: %w", cfg.DBConnMaxLifetime, err)
		}
		sqlDB.SetConnMaxLifetime(d)
	}

	return nil
}
