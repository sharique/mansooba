package testdb_test

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sharique/mansooba/internal/testdb"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// fakeTB lets a test see what New does when no database is configured: Fatalf
// records the message and stops the goroutine, like the real thing.
type fakeTB struct {
	testing.TB
	msg string
}

type stopped struct{}

func (f *fakeTB) Helper()                        {}
func (f *fakeTB) Cleanup(func())                 {}
func (f *fakeTB) Logf(string, ...any)            {}
func (f *fakeTB) Fatalf(format string, a ...any) { f.msg = fmt.Sprintf(format, a...); panic(stopped{}) }
func (f *fakeTB) Fatal(a ...any)                 { f.msg = fmt.Sprint(a...); panic(stopped{}) }
func (f *fakeTB) Errorf(format string, a ...any) { f.msg = fmt.Sprintf(format, a...) }
func (f *fakeTB) Name() string                   { return "fake" }

func onPostgres() bool {
	d := strings.ToLower(os.Getenv("TEST_DB_DRIVER"))
	return d == "" || d == "postgres" || d == "postgresql"
}

func requirePostgres(t *testing.T) {
	t.Helper()
	if !onPostgres() {
		t.Skip("the template and stale-sweep mechanics are PostgreSQL-only")
	}
}

func currentDatabase(t *testing.T, db *gorm.DB) string {
	t.Helper()
	q := `SELECT current_database()`
	if !onPostgres() {
		q = `SELECT DATABASE()`
	}
	var name string
	require.NoError(t, db.Raw(q).Scan(&name).Error)
	return name
}

func databaseExists(t *testing.T, db *gorm.DB, name string) bool {
	t.Helper()
	var n int64
	q := `SELECT COUNT(*) FROM pg_database WHERE datname = ?`
	if !onPostgres() {
		q = `SELECT COUNT(*) FROM information_schema.schemata WHERE schema_name = ?`
	}
	require.NoError(t, db.Raw(q, name).Scan(&n).Error)
	return n > 0
}

func TestNew_GivesIsolatedMigratedDatabases(t *testing.T) {
	a := testdb.New(t)
	b := testdb.New(t)

	require.NoError(t, a.Exec(`INSERT INTO global_settings (setting_key, setting_value) VALUES ('only_in_a', 'x')`).Error)

	var inA, inB int64
	require.NoError(t, a.Raw(`SELECT COUNT(*) FROM global_settings WHERE setting_key = 'only_in_a'`).Scan(&inA).Error)
	require.NoError(t, b.Raw(`SELECT COUNT(*) FROM global_settings WHERE setting_key = 'only_in_a'`).Scan(&inB).Error)
	require.Equal(t, int64(1), inA)
	require.Equal(t, int64(0), inB, "a row written in one test database must not be visible in another")

	// The schema is the real one: users exists with the username column.
	require.True(t, a.Migrator().HasColumn("users", "username"))
}

func TestNew_DropsTheDatabaseWhenTheTestEnds(t *testing.T) {
	var name string
	t.Run("uses a database", func(t *testing.T) {
		db := testdb.New(t)
		name = currentDatabase(t, db)
		require.True(t, strings.HasPrefix(name, "test_"), name)
	})
	admin := testdb.New(t)
	require.False(t, databaseExists(t, admin, name), "the database must be dropped after its test")
}

func TestNew_ParallelTestsShareOneTemplate(t *testing.T) {
	requirePostgres(t)
	t.Run("group", func(t *testing.T) {
		for i := 0; i < 4; i++ {
			t.Run(fmt.Sprintf("parallel-%d", i), func(t *testing.T) {
				t.Parallel()
				db := testdb.New(t)
				require.NoError(t, db.Exec(`INSERT INTO global_settings (setting_key, setting_value) VALUES ('k', 'v')`).Error)
			})
		}
	})
	// The group (and so every parallel subtest) has finished: one template exists
	// for this schema, however many tests used it.
	admin := testdb.New(t)
	var n int64
	require.NoError(t, admin.Raw(`SELECT COUNT(*) FROM pg_database WHERE datname = ?`, testdb.TemplateName(t)).Scan(&n).Error)
	require.Equal(t, int64(1), n)
}

func TestNew_WithoutADatabaseFailsWithInstructions(t *testing.T) {
	t.Setenv("TEST_DB_DSN", "")
	fake := &fakeTB{TB: t}
	func() {
		defer func() { _ = recover() }()
		testdb.New(fake)
	}()
	require.Contains(t, fake.msg, "TEST_DB_DSN")
	require.Contains(t, fake.msg, "docker compose up -d postgres")
}

func TestNew_RemovesStaleTestDatabases(t *testing.T) {
	requirePostgres(t)
	admin := testdb.New(t)
	old := fmt.Sprintf("test_%d_stale", time.Now().Add(-3*time.Hour).Unix())
	fresh := fmt.Sprintf("test_%d_fresh", time.Now().Unix())
	require.NoError(t, admin.Exec(`CREATE DATABASE "`+old+`"`).Error)
	require.NoError(t, admin.Exec(`CREATE DATABASE "`+fresh+`"`).Error)
	t.Cleanup(func() {
		_ = admin.Exec(`DROP DATABASE IF EXISTS "` + fresh + `" WITH (FORCE)`).Error
		_ = admin.Exec(`DROP DATABASE IF EXISTS "` + old + `" WITH (FORCE)`).Error
	})

	testdb.RemoveStale(t) // the same sweep New runs once per process

	require.False(t, databaseExists(t, admin, old), "a database older than an hour must be removed")
	require.True(t, databaseExists(t, admin, fresh), "a recent database must be kept")
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
