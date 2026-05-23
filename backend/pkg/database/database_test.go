package database

import (
	"database/sql"
	"testing"

	"github.com/sharique/mansooba/pkg/config"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newTestSQLDB opens an in-memory SQLite connection and returns the underlying *sql.DB.
func newTestSQLDB(t *testing.T) *sql.DB {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := gdb.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	return sqlDB
}

// TestOpen_UnsupportedDriver verifies that Open returns a descriptive error
// listing all supported driver names.
func TestOpen_UnsupportedDriver(t *testing.T) {
	cfg := &config.Config{DBDriver: "mongo", DBDSN: "mongodb://localhost"}
	_, err := Open(cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported driver")
	require.Contains(t, err.Error(), "mongo")
	require.Contains(t, err.Error(), "sqlite")
	require.Contains(t, err.Error(), "postgres")
	require.Contains(t, err.Error(), "mysql")
}

// TestOpen_SQLite_InMemory verifies that Open succeeds with an in-memory SQLite DSN.
func TestOpen_SQLite_InMemory(t *testing.T) {
	cfg := &config.Config{
		DBDriver:          "sqlite",
		DBDSN:             ":memory:",
		DBConnMaxLifetime: "0",
	}
	db, err := Open(cfg)
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Ping())
	_ = sqlDB.Close()
}

// TestConfigurePool_InvalidDuration verifies that an unparseable DB_CONN_MAX_LIFETIME
// returns an error that names the config key.
func TestConfigurePool_InvalidDuration(t *testing.T) {
	sqlDB := newTestSQLDB(t)
	cfg := &config.Config{DBDriver: "postgres", DBConnMaxLifetime: "not-a-duration"}
	err := configurePool(sqlDB, cfg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "DB_CONN_MAX_LIFETIME")
}

// TestConfigurePool_ValidDuration verifies that a well-formed duration string is accepted.
func TestConfigurePool_ValidDuration(t *testing.T) {
	sqlDB := newTestSQLDB(t)
	cfg := &config.Config{DBDriver: "postgres", DBConnMaxLifetime: "5m"}
	require.NoError(t, configurePool(sqlDB, cfg))
}

// TestConfigurePool_ZeroLifetime verifies that "0" (disabled expiry) is accepted.
func TestConfigurePool_ZeroLifetime(t *testing.T) {
	sqlDB := newTestSQLDB(t)
	cfg := &config.Config{DBDriver: "sqlite", DBConnMaxLifetime: "0"}
	require.NoError(t, configurePool(sqlDB, cfg))
}
