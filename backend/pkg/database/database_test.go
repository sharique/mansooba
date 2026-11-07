package database

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sharique/mansooba/pkg/config"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newTestSQLDB returns a *sql.DB that never connects (database/sql opens lazily),
// which is all the pool-setting tests need.
func newTestSQLDB(t *testing.T) *sql.DB {
	t.Helper()
	sqlDB, err := sql.Open("pgx", "host=127.0.0.1 port=1 user=none dbname=none")
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	return sqlDB
}

// Open applies the same settings checks as the server's start-up, so every
// command that opens the database refuses a bad setting the same way.
func TestOpen_RefusesBadSettingsBeforeConnecting(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Config
		want string
	}{
		{"unsupported driver", config.Config{DBDriver: "mongo", DBDSN: "mongodb://localhost"}, `unsupported DB_DRIVER "mongo"`},
		{"sqlite", config.Config{DBDriver: "sqlite", DBDSN: ":memory:"}, "SQLite is no longer supported"},
		{"no driver", config.Config{DBDSN: "host=x"}, "DB_DRIVER is not set"},
		{"no connection string", config.Config{DBDriver: "postgres"}, "DB_DSN is not set"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Open(&tt.cfg)
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.want)
		})
	}
}

// A refused SQLite setting must not create the file it names.
func TestOpen_SQLiteRefusalLeavesNoFileBehind(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dev.db")
	_, err := Open(&config.Config{DBDriver: "sqlite", DBDSN: path})
	require.Error(t, err)
	_, statErr := os.Stat(path)
	require.True(t, os.IsNotExist(statErr), "the SQLite file must not be created")
}

// The pool follows the settings exactly; nothing forces a single connection.
func TestConfigurePool_AppliesTheSettings(t *testing.T) {
	sqlDB := newTestSQLDB(t)
	require.NoError(t, configurePool(sqlDB, &config.Config{DBDriver: "postgres", DBMaxOpenConns: 7, DBMaxIdleConns: 3}))
	require.Equal(t, 7, sqlDB.Stats().MaxOpenConnections)

	require.NoError(t, configurePool(sqlDB, &config.Config{DBDriver: "mysql", DBMaxOpenConns: 0, DBMaxIdleConns: 2}))
	require.Equal(t, 0, sqlDB.Stats().MaxOpenConnections, "0 means unlimited")
}

// ── waiting for a database that is still starting ─────────────────────────────

// failTimes makes gormOpen fail that many times, then open without pinging (so no
// server is needed for the attempt that gets through).
func failTimes(t *testing.T, n int) *int {
	t.Helper()
	calls := 0
	real := gormOpen
	gormOpen = func(d gorm.Dialector, c ...gorm.Option) (*gorm.DB, error) {
		calls++
		if calls <= n {
			return nil, &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}
		}
		return real(d, &gorm.Config{DisableAutomaticPing: true})
	}
	t.Cleanup(func() { gormOpen = real })
	return &calls
}

func fastWait(t *testing.T, total, every time.Duration) {
	t.Helper()
	w, e := connectWait, connectRetryEvery
	connectWait, connectRetryEvery = total, every
	t.Cleanup(func() { connectWait, connectRetryEvery = w, e })
}

func TestOpen_RetriesUntilTheDatabaseIsUp(t *testing.T) {
	fastWait(t, 5*time.Second, 10*time.Millisecond)
	calls := failTimes(t, 3)
	db, err := Open(&config.Config{DBDriver: "postgres", DBDSN: "host=127.0.0.1 port=1 user=u dbname=d", DBConnMaxLifetime: "0"})
	require.NoError(t, err)
	require.NotNil(t, db)
	require.Equal(t, 4, *calls, "three refusals, then the attempt that got through")
}

func TestOpen_GivesUpAfterTheBoundAndSaysWhere(t *testing.T) {
	fastWait(t, 200*time.Millisecond, 20*time.Millisecond)
	failTimes(t, 1<<30)
	_, err := Open(&config.Config{DBDriver: "postgres", DBDSN: "host=db.internal port=5432 user=u password=hunter2 dbname=d"})
	require.Error(t, err)
	msg := err.Error()
	require.Contains(t, msg, "could not connect to postgres at db.internal:5432 after 200ms")
	require.NotContains(t, msg, "hunter2", "the password must never be printed")
}

func TestOpen_DoesNotRetryAWrongSetting(t *testing.T) {
	fastWait(t, 5*time.Second, 10*time.Millisecond)
	calls := failTimes(t, 1<<30)
	start := time.Now()
	_, err := Open(&config.Config{DBDriver: "sqlite", DBDSN: "x"})
	require.Error(t, err)
	require.Equal(t, 0, *calls)
	require.Less(t, time.Since(start), time.Second)
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
	cfg := &config.Config{DBDriver: "postgres", DBConnMaxLifetime: "0"}
	require.NoError(t, configurePool(sqlDB, cfg))
}

// Only "the database is not reachable yet" is worth waiting for. A wrong
// password or database name will not fix itself, so it fails at once.
func TestOpen_DoesNotWaitForARefusedLogin(t *testing.T) {
	fastWait(t, 5*time.Second, 10*time.Millisecond)
	calls := 0
	real := gormOpen
	gormOpen = func(gorm.Dialector, ...gorm.Option) (*gorm.DB, error) {
		calls++
		return nil, &pgconn.PgError{Code: "28P01", Message: "password authentication failed for user \"u\""}
	}
	t.Cleanup(func() { gormOpen = real })

	start := time.Now()
	_, err := Open(&config.Config{DBDriver: "postgres", DBDSN: "host=127.0.0.1 port=5432 user=u password=wrong dbname=d"})
	require.Error(t, err)
	require.Equal(t, 1, calls)
	require.Less(t, time.Since(start), time.Second)
	require.Contains(t, err.Error(), "password authentication failed")
	require.NotContains(t, err.Error(), "wrong", "the password itself must not be printed")
}

func TestRetryable(t *testing.T) {
	refused := &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}
	require.True(t, retryable(refused))
	require.True(t, retryable(fmt.Errorf("failed to connect: %w", refused)))
	require.True(t, retryable(io.ErrUnexpectedEOF))
	require.True(t, retryable(&pgconn.PgError{Code: "57P03"}), "the database system is starting up")
	require.False(t, retryable(&pgconn.PgError{Code: "28P01"}), "bad password")
	require.False(t, retryable(&pgconn.PgError{Code: "3D000"}), "no such database")
	require.False(t, retryable(&mysqldriver.MySQLError{Number: 1045}), "access denied")
	require.False(t, retryable(errors.New("something else")))
}
