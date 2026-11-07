package database

import (
	"errors"
	"net"
	"syscall"
	"testing"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/sharique/mansooba/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ── a failed attempt must not leak its pool or fill the log ──────────────────

func TestOpen_ClosesThePoolOfAFailedAttemptAndRestoresTheLogger(t *testing.T) {
	fastWait(t, 5*time.Second, 5*time.Millisecond)
	real := gormOpen
	var firstAttempt *gorm.DB
	var attempts int
	gormOpen = func(d gorm.Dialector, c ...gorm.Option) (*gorm.DB, error) {
		attempts++
		db, err := real(d, &gorm.Config{DisableAutomaticPing: true})
		require.NoError(t, err)
		if attempts == 1 {
			firstAttempt = db
			// gorm returns the half-open database together with the ping error
			return db, &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}
		}
		return db, nil
	}
	t.Cleanup(func() { gormOpen = real })

	db, err := Open(&config.Config{DBDriver: "postgres", DBDSN: "host=127.0.0.1 port=1 user=u dbname=d"})
	require.NoError(t, err)
	require.Equal(t, 2, attempts)

	first, err := firstAttempt.DB()
	require.NoError(t, err)
	assert.ErrorContains(t, first.Ping(), "closed", "the pool of the failed attempt must be closed")

	assert.False(t, db.Logger == logger.Discard, "connection errors are silenced while waiting, but the database logs normally afterwards")
}

// ── passwords never reach a message ──────────────────────────────────────────

func TestRedactSecrets(t *testing.T) {
	tests := []struct {
		name, kind, dsn, text string
	}{
		{"plain key/value", "postgres", "host=h user=u password=hunter2 dbname=d", "failed: password=hunter2 rejected"},
		{"quoted with spaces", "postgres", "host=h user=u password='two words here' dbname=d", "failed: two words here rejected"},
		{"escaped quote", "postgres", `host=h password='it\'s' dbname=d`, "failed: it's rejected"},
		{"URL form", "postgres", "postgres://app:s3cret@db.internal:5432/x", "dial postgres://app:s3cret@db.internal:5432/x failed"},
		{"mysql", "mysql", "app:pa55word@tcp(db:3306)/x", "access denied using pa55word"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := redactSecrets(tt.text, tt.kind, tt.dsn)
			for _, secret := range []string{"hunter2", "two words here", "it's", "s3cret", "pa55word"} {
				assert.NotContains(t, out, secret)
			}
			assert.Contains(t, out, "***")
		})
	}
	assert.Equal(t, "nothing to hide", redactSecrets("nothing to hide", "postgres", "host=h dbname=d"))
}

// ── what is worth waiting for ────────────────────────────────────────────────

func TestRetryable_MySQLServerStartUpAndCapacityErrors(t *testing.T) {
	assert.True(t, retryable(&mysqldriver.MySQLError{Number: 1040}), "too many connections")
	assert.True(t, retryable(&mysqldriver.MySQLError{Number: 1053}), "server shutdown in progress")
	assert.False(t, retryable(&mysqldriver.MySQLError{Number: 1045}), "access denied")
	assert.False(t, retryable(&mysqldriver.MySQLError{Number: 1049}), "unknown database")
	assert.False(t, retryable(errors.New("other")))
}

// One attempt must not outlast the whole wait, so a connection string without a
// timeout gets a short one.
func TestWithConnectTimeout(t *testing.T) {
	assert.Equal(t, "host=h dbname=d connect_timeout=5", withConnectTimeout("postgres", "host=h dbname=d"))
	assert.Equal(t, "host=h connect_timeout=30", withConnectTimeout("postgres", "host=h connect_timeout=30"), "an explicit timeout is kept")
	assert.Contains(t, withConnectTimeout("postgres", "postgres://u:p@h/d?sslmode=disable"), "connect_timeout=5")
	assert.Contains(t, withConnectTimeout("postgres", "postgres://u:p@h/d?connect_timeout=9"), "connect_timeout=9")
	assert.Equal(t, "anything", withConnectTimeout("mysql", "anything"), "MySQL gets its timeout from the driver config instead")
}

func TestMySQLDSN_GetsADialTimeoutUnlessOneIsGiven(t *testing.T) {
	out, err := mysqlDSNWithFoundRows("u:p@tcp(h:3306)/d")
	require.NoError(t, err)
	assert.Contains(t, out, "timeout=5s")
	out, err = mysqlDSNWithFoundRows("u:p@tcp(h:3306)/d?timeout=20s")
	require.NoError(t, err)
	assert.Contains(t, out, "timeout=20s")
}

// ── the address shown when giving up ─────────────────────────────────────────

func TestDSNAddress(t *testing.T) {
	pg := func(dsn string) string { return dsnAddress(&config.Config{DBDriver: "postgres", DBDSN: dsn}) }
	assert.Equal(t, "db.internal:5432", pg("host=db.internal user=u"))
	assert.Equal(t, "db.internal:6432", pg("host=db.internal port=6432"))
	assert.Equal(t, "db.internal:5433", pg("host='db.internal' port='5433' user=u"))
	assert.Equal(t, "db.internal:6432", pg("postgres://app:s3cret@db.internal:6432/x"))
	assert.Equal(t, "db.internal:5432", pg("postgres://app:s3cret@db.internal/x"))
	assert.Equal(t, "the configured address", pg("dbname=d"))

	my := func(dsn string) string { return dsnAddress(&config.Config{DBDriver: "mariadb", DBDSN: dsn}) }
	assert.Equal(t, "db:3307", my("u:p@tcp(db:3307)/d"))
}

var _ = postgres.Open // keep the driver import used by tests that build lazy databases
