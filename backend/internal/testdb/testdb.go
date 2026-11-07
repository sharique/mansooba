// Package testdb gives each test its own freshly migrated database on a real
// server, so tests run on the engines the product runs on (PostgreSQL, and
// MySQL/MariaDB on demand) instead of an in-memory stand-in.
//
// It is test support: import it from _test.go files and from code that builds
// a test server. Set TEST_DB_DSN (and TEST_DB_DRIVER for MySQL or MariaDB); see
// CONTRIBUTING.md for how to start a database.
//
// On PostgreSQL the first call in a run builds a template database with the
// real database.Migrate (about 650 ms); each test then gets a copy of it (about
// 30 ms). MySQL and MariaDB have no cheap copy, so each test migrates its own.
package testdb

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/sharique/mansooba/pkg/config"
	"github.com/sharique/mansooba/pkg/database"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	// staleAfter is how old a leftover test database must be before the sweep
	// removes it (a crashed run can leave one behind).
	staleAfter = time.Hour
	// advisoryLockKey serialises template creation across test binaries that
	// run in parallel.
	advisoryLockKey = 727274
)

var (
	adminMu   sync.Mutex
	adminDB   *gorm.DB
	adminKind string
	adminDSN  string

	templateOnce sync.Once
	templateErr  error
)

// New returns a database that only this test uses, already migrated with the
// real database.Migrate. It is dropped when the test ends.
func New(t testing.TB) *gorm.DB {
	t.Helper()
	return open(t, Config(t))
}

// NewEmpty is like New but the database has no tables, for tests that build an
// older schema by hand and then run the migration themselves.
func NewEmpty(t testing.TB) *gorm.DB {
	t.Helper()
	return open(t, ConfigEmpty(t))
}

// Config creates a fresh migrated database and returns the settings that reach
// it, for code that opens the database itself (the HTTP test server, commands).
// The database is dropped when the test ends.
func Config(t testing.TB) *config.Config {
	t.Helper()
	return create(t, true)
}

// ConfigEmpty is Config for a database with no tables.
func ConfigEmpty(t testing.TB) *config.Config {
	t.Helper()
	return create(t, false)
}

func open(t testing.TB, cfg *config.Config) *gorm.DB {
	t.Helper()
	db, err := database.Open(cfg)
	if err != nil {
		t.Fatalf("testdb: open the test database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("testdb: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	// Tests that build an old schema by hand with AutoMigrate need the indexed text
	// columns sized, as the real migration does first (MySQL only; a no-op elsewhere).
	if err := database.PrepareModels(db); err != nil {
		t.Fatalf("testdb: %v", err)
	}
	return db
}

func create(t testing.TB, migrated bool) *config.Config {
	t.Helper()
	kind, dsn := settings(t)
	admin := connectAdmin(t, kind, dsn)

	name := fmt.Sprintf("test_%d_%s", time.Now().Unix(), randomSuffix())
	var perTestDSN string
	switch kind {
	case "postgres":
		if migrated {
			mustExec(t, admin, fmt.Sprintf(`CREATE DATABASE "%s" TEMPLATE "%s"`, name, ensureTemplate(t, admin, dsn)))
		} else {
			mustExec(t, admin, fmt.Sprintf(`CREATE DATABASE "%s"`, name))
		}
		perTestDSN = withDatabase(dsn, name)
	default:
		mustExec(t, admin, fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET utf8mb4", name))
		perTestDSN = mysqlDSNFor(t, dsn, name)
		if migrated {
			migrateOnce(t, kind, perTestDSN)
		}
	}
	t.Cleanup(func() {
		if err := dropDatabase(admin, kind, name); err != nil {
			t.Logf("testdb: could not drop %s (the next run's sweep removes it after an hour): %v", name, err)
		}
	})
	return &config.Config{
		DBDriver:          kind,
		DBDSN:             perTestDSN,
		DBMaxOpenConns:    20,
		DBMaxIdleConns:    5,
		DBConnMaxLifetime: "0",
	}
}

// TemplateName is the name of the PostgreSQL template database for the current
// schema. Exported for the helper's own tests.
func TemplateName(t testing.TB) string {
	t.Helper()
	return "mansooba_tmpl_" + schemaHash(t)
}

// RemoveStale drops test databases left behind by a crashed run (their name
// carries a creation time older than an hour). New runs it once per process.
func RemoveStale(t testing.TB) {
	t.Helper()
	kind, dsn := settings(t)
	admin := connectAdmin(t, kind, dsn)
	sweep(admin, kind)
}

// ── settings and connections ──────────────────────────────────────────────────

func settings(t testing.TB) (kind, dsn string) {
	t.Helper()
	dsn = strings.TrimSpace(os.Getenv("TEST_DB_DSN"))
	kind = strings.ToLower(strings.TrimSpace(os.Getenv("TEST_DB_DRIVER")))
	if kind == "" || kind == "postgresql" {
		kind = "postgres"
	}
	if kind == "mariadb" {
		kind = "mysql"
	}
	if dsn == "" {
		t.Fatalf("testdb: TEST_DB_DSN is not set. Tests run on a real database. Start PostgreSQL with " +
			"`docker compose up -d postgres` (from code/) and set TEST_DB_DSN, for example:\n" +
			"  export TEST_DB_DSN=\"host=localhost user=mansooba password=mansooba dbname=mansooba_test port=5432 sslmode=disable\"\n" +
			"For MariaDB or MySQL also set TEST_DB_DRIVER=mariadb (or mysql). See CONTRIBUTING.md.")
	}
	if kind != "postgres" && kind != "mysql" {
		t.Fatalf("testdb: unsupported TEST_DB_DRIVER %q (use postgres, mysql or mariadb)", kind)
	}
	return kind, dsn
}

// connectAdmin opens (once per process) the connection that creates and drops
// test databases, and runs the stale sweep the first time.
func connectAdmin(t testing.TB, kind, dsn string) *gorm.DB {
	t.Helper()
	adminMu.Lock()
	defer adminMu.Unlock()
	if adminDB != nil && adminKind == kind && adminDSN == dsn {
		return adminDB
	}
	var dialector gorm.Dialector
	if kind == "postgres" {
		dialector = postgres.Open(dsn)
	} else {
		dialector = mysql.Open(dsn)
	}
	db, err := gorm.Open(dialector, &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("testdb: cannot reach the test database (%s): %v\n"+
			"Start it with `docker compose up -d postgres` and check TEST_DB_DSN.", kind, err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("testdb: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("testdb: cannot reach the test database (%s): %v\n"+
			"Start it with `docker compose up -d postgres` and check TEST_DB_DSN.", kind, err)
	}
	sqlDB.SetMaxOpenConns(8)
	adminDB, adminKind, adminDSN = db, kind, dsn
	sweep(db, kind)
	return db
}

func mustExec(t testing.TB, db *gorm.DB, query string) {
	t.Helper()
	if err := db.Exec(query).Error; err != nil {
		t.Fatalf("testdb: %s: %v", query, err)
	}
}

func dropDatabase(admin *gorm.DB, kind, name string) error {
	if kind == "postgres" {
		return admin.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "%s" WITH (FORCE)`, name)).Error
	}
	return admin.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", name)).Error
}

func randomSuffix() string {
	return strconv.FormatUint(rand.Uint64N(1<<40), 36)
}

// ── stale databases ───────────────────────────────────────────────────────────

func sweep(admin *gorm.DB, kind string) {
	var names []string
	var err error
	if kind == "postgres" {
		err = admin.Raw(`SELECT datname FROM pg_database WHERE datname LIKE 'test\_%'`).Scan(&names).Error
	} else {
		err = admin.Raw("SHOW DATABASES LIKE 'test\\_%'").Scan(&names).Error
	}
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-staleAfter).Unix()
	for _, n := range names {
		if unix, ok := parseTestDatabaseName(n); ok && unix <= cutoff {
			_ = dropDatabase(admin, kind, n)
		}
	}
}

// testDatabaseName is exactly what Config names its databases:
// test_<unix seconds>_<base 36 suffix>. The sweep drops only names of this shape,
// so a database that merely starts with "test_" on a shared server is left alone.
var testDatabaseName = regexp.MustCompile(`^test_([0-9]{9,11})_[0-9a-z]+$`)

func parseTestDatabaseName(name string) (unix int64, ok bool) {
	m := testDatabaseName.FindStringSubmatch(name)
	if m == nil {
		return 0, false
	}
	unix, err := strconv.ParseInt(m[1], 10, 64)
	return unix, err == nil
}

// withDatabase returns the connection string with the database replaced by name.
// It accepts a key/value string (a later dbname wins) or a postgres:// URL.
func withDatabase(dsn, name string) string {
	if strings.Contains(dsn, "://") {
		if u, err := url.Parse(dsn); err == nil {
			u.Path = "/" + name
			return u.String()
		}
	}
	return dsn + " dbname=" + name
}

// ── PostgreSQL template ───────────────────────────────────────────────────────

// ensureTemplate makes sure the template database for the current schema
// exists and returns its name. A template is built under a temporary name and
// renamed when complete, so a run that is killed half way never leaves a
// half-migrated template behind.
func ensureTemplate(t testing.TB, admin *gorm.DB, dsn string) string {
	t.Helper()
	name := TemplateName(t)
	templateOnce.Do(func() { templateErr = buildTemplate(admin, dsn, name) })
	if templateErr != nil {
		t.Fatalf("testdb: build the template database %s: %v", name, templateErr)
	}
	return name
}

func buildTemplate(admin *gorm.DB, dsn, name string) error {
	sqlDB, err := admin.DB()
	if err != nil {
		return err
	}
	ctx := context.Background()
	conn, err := sqlDB.Conn(ctx) // one session holds the advisory lock
	if err != nil {
		return err
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, advisoryLockKey); err != nil {
		return err
	}
	defer conn.ExecContext(ctx, `SELECT pg_advisory_unlock($1)`, advisoryLockKey) //nolint:errcheck

	exists, err := databaseExists(ctx, conn, name)
	if err != nil || exists {
		return err
	}
	building := name + "_build"
	if _, err := conn.ExecContext(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS "%s" WITH (FORCE)`, building)); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, fmt.Sprintf(`CREATE DATABASE "%s"`, building)); err != nil {
		return err
	}
	if err := migrateDSN("postgres", withDatabase(dsn, building)); err != nil {
		_, _ = conn.ExecContext(ctx, fmt.Sprintf(`DROP DATABASE IF EXISTS "%s" WITH (FORCE)`, building))
		return err
	}
	_, err = conn.ExecContext(ctx, fmt.Sprintf(`ALTER DATABASE "%s" RENAME TO "%s"`, building, name))
	return err
}

func databaseExists(ctx context.Context, conn *sql.Conn, name string) (bool, error) {
	var n int
	err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM pg_database WHERE datname = $1`, name).Scan(&n)
	return n > 0, err
}

// schemaHash fingerprints the code that decides what a migrated database looks
// like (the models and the migration steps), so a change to either makes a new
// template instead of reusing a stale one.
func schemaHash(t testing.TB) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("testdb: cannot locate its own source")
	}
	backend := filepath.Join(filepath.Dir(file), "..", "..")
	var files []string
	for _, dir := range []string{"internal/domain", "pkg/database"} {
		matches, _ := filepath.Glob(filepath.Join(backend, dir, "*.go"))
		for _, m := range matches {
			if !strings.HasSuffix(m, "_test.go") {
				files = append(files, m)
			}
		}
	}
	sort.Strings(files)
	h := sha256.New()
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("testdb: %v", err)
		}
		h.Write([]byte(filepath.Base(f)))
		h.Write(b)
	}
	return hex.EncodeToString(h.Sum(nil))[:12]
}

// ── MySQL / MariaDB ───────────────────────────────────────────────────────────

func mysqlDSNFor(t testing.TB, dsn, dbName string) string {
	t.Helper()
	cfg, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("testdb: TEST_DB_DSN is not a valid MySQL/MariaDB DSN: %v", err)
	}
	cfg.DBName = dbName
	cfg.ParseTime = true
	return cfg.FormatDSN()
}

func migrateOnce(t testing.TB, kind, dsn string) {
	t.Helper()
	if err := migrateDSN(kind, dsn); err != nil {
		t.Fatalf("testdb: migrate the test database: %v", err)
	}
}

func migrateDSN(kind, dsn string) error {
	var dialector gorm.Dialector
	if kind == "postgres" {
		dialector = postgres.Open(dsn)
	} else {
		dialector = mysql.Open(dsn)
	}
	db, err := gorm.Open(dialector, &gorm.Config{Logger: logger.Discard})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	return database.Migrate(db)
}
