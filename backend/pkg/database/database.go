package database

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sharique/mansooba/pkg/config"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// gormOpen is gorm.Open, replaceable in tests.
var gormOpen = gorm.Open

// How long Open keeps trying a database that is not reachable yet, and how often.
// A database that is still starting (a fresh container, a restart) should not
// crash the server; one that stays unreachable should stop it with a clear message.
var (
	connectWait       = 30 * time.Second
	connectRetryEvery = time.Second
)

// Open returns a *gorm.DB for the database named by cfg.DBDriver and cfg.DBDSN.
// Supported drivers: postgres (postgresql), mysql, mariadb. Both settings are
// checked first (config.ValidateDatabase), so a missing or unsupported one, SQLite
// included, is refused before anything is touched. A database that is not
// reachable yet is retried for up to 30 seconds. The pool is configured via
// configurePool after the connection is established.
func Open(cfg *config.Config) (*gorm.DB, error) {
	if err := config.ValidateDatabase(cfg); err != nil {
		return nil, err
	}

	var dialector gorm.Dialector
	switch cfg.DatabaseKind() {
	case "postgres":
		dialector = postgres.Open(withConnectTimeout("postgres", cfg.DBDSN))
	case "mysql":
		dsn, err := mysqlDSNWithFoundRows(cfg.DBDSN)
		if err != nil {
			return nil, err
		}
		dialector = mysql.Open(dsn)
	}

	db, err := openWithWait(dialector, cfg)
	if err != nil {
		return nil, err
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

// openWithWait opens the database, retrying while it is not reachable yet.
func openWithWait(dialector gorm.Dialector, cfg *config.Config) (*gorm.DB, error) {
	kind := cfg.DatabaseKind()
	deadline := time.Now().Add(connectWait)
	for attempt := 1; ; attempt++ {
		// GORM logs every failed attempt and leaves its half-open pool behind.
		// Stay quiet while waiting, close what a failed attempt opened, and give
		// the database its normal logger once it is up.
		db, err := gormOpen(dialector, &gorm.Config{Logger: logger.Discard})
		if err == nil {
			db.Logger = logger.Default
			return db, nil
		}
		if db != nil {
			if sqlDB, e := db.DB(); e == nil {
				_ = sqlDB.Close()
			}
		}
		reason := redactSecrets(err.Error(), kind, cfg.DBDSN)
		if !retryable(err) {
			return nil, fmt.Errorf("database: could not connect to %s: %s", driverName(cfg), reason)
		}
		if time.Now().Add(connectRetryEvery).After(deadline) {
			return nil, fmt.Errorf("database: could not connect to %s at %s after %s: %s",
				driverName(cfg), dsnAddress(cfg), connectWait, reason)
		}
		if attempt == 1 {
			fmt.Fprintf(os.Stderr, "database: waiting for %s at %s (%s)\n", driverName(cfg), dsnAddress(cfg), reason)
		}
		time.Sleep(connectRetryEvery)
	}
}

// retryable reports whether err means "the database is not reachable (yet)", as
// opposed to a refusal that waiting will not fix, such as a wrong password or
// database name. It reads the error's type and code, not its wording.
func retryable(err error) bool {
	var netErr net.Error
	var pgErr *pgconn.PgError
	var myErr *mysqldriver.MySQLError
	switch {
	case errors.As(err, &netErr), errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF),
		errors.Is(err, driver.ErrBadConn), errors.Is(err, mysqldriver.ErrInvalidConn):
		return true
	case errors.As(err, &pgErr):
		// 57P03 the database system is starting up; 57P01 shutting down; 53300 too many connections.
		return pgErr.Code == "57P03" || pgErr.Code == "57P01" || pgErr.Code == "53300"
	case errors.As(err, &myErr):
		// 1040 too many connections; 1053 server shutdown in progress.
		return myErr.Number == 1040 || myErr.Number == 1053
	}
	return false
}

func driverName(cfg *config.Config) string { return strings.ToLower(strings.TrimSpace(cfg.DBDriver)) }

// connectTimeoutSeconds bounds one connection attempt, so a single attempt on an
// address that never answers cannot outlast the whole wait.
const connectTimeoutSeconds = 5

// withConnectTimeout adds connect_timeout to a PostgreSQL connection string that
// does not already set one. MySQL and MariaDB get their timeout in
// mysqlDSNWithFoundRows instead.
func withConnectTimeout(kind, dsn string) string {
	if kind != "postgres" {
		return dsn
	}
	if strings.Contains(dsn, "://") {
		u, err := url.Parse(dsn)
		if err != nil {
			return dsn
		}
		q := u.Query()
		if q.Get("connect_timeout") == "" {
			q.Set("connect_timeout", strconv.Itoa(connectTimeoutSeconds))
			u.RawQuery = q.Encode()
		}
		return u.String()
	}
	if _, set := parseKeyValueDSN(dsn)["connect_timeout"]; set {
		return dsn
	}
	return dsn + " connect_timeout=" + strconv.Itoa(connectTimeoutSeconds)
}

var rePassword = regexp.MustCompile(`(?i)password=\S+`)

// redactSecrets removes the password from text that may echo the connection
// string: the password's own value wherever it appears (it is read from the
// connection string, so quoted values, spaces and URL-style strings are covered),
// and anything that looks like password=... as a fallback.
func redactSecrets(text, kind, dsn string) string {
	if pw := dsnPassword(kind, dsn); len(pw) >= 3 {
		text = strings.ReplaceAll(text, pw, "***")
	}
	return rePassword.ReplaceAllString(text, "password=***")
}

// dsnPassword returns the password in a connection string, or "".
func dsnPassword(kind, dsn string) string {
	switch kind {
	case "postgres":
		if strings.Contains(dsn, "://") {
			if u, err := url.Parse(dsn); err == nil && u.User != nil {
				pw, _ := u.User.Password()
				return pw
			}
			return ""
		}
		return parseKeyValueDSN(dsn)["password"]
	case "mysql":
		if c, err := mysqldriver.ParseDSN(dsn); err == nil {
			return c.Passwd
		}
	}
	return ""
}

// parseKeyValueDSN reads a libpq key/value connection string: space-separated
// key=value pairs, a value optionally in single quotes with backslash escapes.
func parseKeyValueDSN(dsn string) map[string]string {
	out := map[string]string{}
	i := 0
	for i < len(dsn) {
		for i < len(dsn) && dsn[i] == ' ' {
			i++
		}
		start := i
		for i < len(dsn) && dsn[i] != '=' && dsn[i] != ' ' {
			i++
		}
		key := dsn[start:i]
		if i >= len(dsn) || dsn[i] != '=' {
			continue
		}
		i++ // '='
		var val strings.Builder
		if i < len(dsn) && dsn[i] == '\'' {
			i++
			for i < len(dsn) && dsn[i] != '\'' {
				if dsn[i] == '\\' && i+1 < len(dsn) {
					i++
				}
				val.WriteByte(dsn[i])
				i++
			}
			i++ // closing quote
		} else {
			for i < len(dsn) && dsn[i] != ' ' {
				val.WriteByte(dsn[i])
				i++
			}
		}
		if key != "" {
			out[key] = val.String()
		}
	}
	return out
}

// dsnAddress returns host:port from the connection string for messages, or a
// generic phrase when it cannot be read. It never returns the password.
func dsnAddress(cfg *config.Config) string {
	switch cfg.DatabaseKind() {
	case "postgres":
		host, port := "", "5432"
		if strings.Contains(cfg.DBDSN, "://") {
			if u, err := url.Parse(cfg.DBDSN); err == nil {
				host = u.Hostname()
				if p := u.Port(); p != "" {
					port = p
				}
			}
		} else {
			kv := parseKeyValueDSN(cfg.DBDSN)
			host = kv["host"]
			if p := kv["port"]; p != "" {
				port = p
			}
		}
		if host != "" {
			return net.JoinHostPort(host, port)
		}
	case "mysql":
		if c, err := mysqldriver.ParseDSN(cfg.DBDSN); err == nil && c.Addr != "" {
			return c.Addr
		}
	}
	return "the configured address"
}

// configurePool sets connection pool parameters on sqlDB from cfg: the maximum
// open and idle connections and the connection lifetime. Load supplies bounded
// defaults (25 open, 5 idle); DB_MAX_OPEN_CONNS=0 means unlimited.
func configurePool(sqlDB *sql.DB, cfg *config.Config) error {
	sqlDB.SetMaxOpenConns(cfg.DBMaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.DBMaxIdleConns)

	if cfg.DBConnMaxLifetime != "" && cfg.DBConnMaxLifetime != "0" {
		d, err := time.ParseDuration(cfg.DBConnMaxLifetime)
		if err != nil {
			return fmt.Errorf("database: invalid DB_CONN_MAX_LIFETIME %q: %w", cfg.DBConnMaxLifetime, err)
		}
		sqlDB.SetConnMaxLifetime(d)
	}

	return nil
}

// mysqlDSNWithFoundRows turns on clientFoundRows. Without it the MySQL driver
// reports the rows an UPDATE *changed*, so setting a column to the value it
// already has looks like "no row matched"; PostgreSQL reports rows *matched*.
// The repositories read RowsAffected to tell "nothing to change" from "no such
// row", so both databases must count the same way. The rest of the DSN is kept
// as the operator wrote it. The error never repeats the DSN, which holds the
// password.
func mysqlDSNWithFoundRows(dsn string) (string, error) {
	cfg, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		return "", fmt.Errorf("database: DB_DSN is not a valid MySQL/MariaDB connection string: %w", err)
	}
	cfg.ClientFoundRows = true
	if cfg.Timeout == 0 {
		cfg.Timeout = connectTimeoutSeconds * time.Second // one attempt must not outlast the whole wait
	}
	return cfg.FormatDSN(), nil
}
