package database

import (
	"os"
	"strings"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// mysqlWithoutServer opens GORM with the MySQL dialect but never connects, so the column
// types GORM would choose can be inspected without a server.
func mysqlWithoutServer(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(
		mysql.New(mysql.Config{DSN: "u:p@tcp(127.0.0.1:1)/none", SkipInitializeWithVersion: true}),
		&gorm.Config{DisableAutomaticPing: true},
	)
	require.NoError(t, err)
	return db
}

// indexedStrings lists every string column that takes part in an index, with the
// index it belongs to, for the given models.
func indexedStrings(t *testing.T, db *gorm.DB, models []any) []indexedString {
	t.Helper()
	var out []indexedString
	for _, m := range models {
		stmt := &gorm.Statement{DB: db}
		require.NoError(t, stmt.Parse(m))
		for _, idx := range stmt.Schema.ParseIndexes() {
			for _, opt := range idx.Fields {
				if opt.Field != nil && opt.Field.DataType == schema.String {
					out = append(out, indexedString{table: stmt.Schema.Table, index: idx.Name, field: opt.Field, prefix: opt.Length})
				}
			}
		}
	}
	return out
}

type indexedString struct {
	table, index string
	field        *schema.Field
	prefix       int
}

func (s indexedString) sqlType(db *gorm.DB) string { return db.Dialector.DataTypeOf(s.field) }

func isTextOrBlob(sqlType string) bool {
	return strings.Contains(sqlType, "text") || strings.Contains(sqlType, "blob")
}

// The problem this file guards against: on MySQL, GORM gives a string column a varchar
// type only when its tag says `index` or `unique`. A column tagged `uniqueIndex` gets
// longtext, and MySQL refuses to index it (Error 1170), so a fresh MySQL database could
// never be migrated.
func TestMySQL_UnsizedUniqueIndexStringsAreTextWithoutTheFix(t *testing.T) {
	db := mysqlWithoutServer(t)
	var text []string
	for _, s := range indexedStrings(t, db, migrationModels()) {
		if s.prefix == 0 && isTextOrBlob(s.sqlType(db)) {
			text = append(text, s.table+"."+s.field.DBName)
		}
	}
	assert.Contains(t, text, "users.email", "this is the column MySQL rejected; if it is no longer text, the fix may be unnecessary")
	assert.Contains(t, text, "issues.key")
}

func TestSizeIndexedStrings_MakesEveryIndexedColumnIndexableOnMySQL(t *testing.T) {
	db := mysqlWithoutServer(t)
	require.NoError(t, sizeIndexedStrings(db, migrationModels()...))

	fields := indexedStrings(t, db, migrationModels())
	require.NotEmpty(t, fields)
	for _, s := range fields {
		if s.prefix > 0 {
			continue // an index with an explicit prefix length can use a text column
		}
		typ := s.sqlType(db)
		assert.Falsef(t, isTextOrBlob(typ), "%s.%s (index %s) is %q, which MySQL cannot index", s.table, s.field.DBName, s.index, typ)
	}
}

func TestSizeIndexedStrings_KeepsAnExplicitSizeAndLeavesOtherColumnsAlone(t *testing.T) {
	db := mysqlWithoutServer(t)
	require.NoError(t, sizeIndexedStrings(db, migrationModels()...))

	byName := func(model any, col string) *schema.Field {
		stmt := &gorm.Statement{DB: db}
		require.NoError(t, stmt.Parse(model))
		return stmt.Schema.FieldsByDBName[col]
	}
	assert.Equal(t, 30, byName(&domain.User{}, "username").Size, "an explicit size:30 must stay")
	assert.Equal(t, 64, byName(&domain.PasswordResetToken{}, "token_hash").Size, "an explicit size:64 must stay")
	assert.Equal(t, mysqlIndexedStringSize, byName(&domain.User{}, "email").Size)
	// Text that is never indexed must stay long text, or descriptions would be cut short.
	assert.Equal(t, 0, byName(&domain.Issue{}, "description").Size)
	assert.Contains(t, db.Dialector.DataTypeOf(byName(&domain.Issue{}, "description")), "text")
}

func TestSizeIndexedStrings_DoesNothingOnOtherDatabases(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, sizeIndexedStrings(db, migrationModels()...))

	stmt := &gorm.Statement{DB: db}
	require.NoError(t, stmt.Parse(&domain.User{}))
	assert.Equal(t, 0, stmt.Schema.FieldsByDBName["email"].Size, "PostgreSQL and SQLite columns must not change")
}

// Against a real server. Set MYSQL_TEST_DSN to an empty scratch database, for example
//
//	MYSQL_TEST_DSN='root:secret@tcp(127.0.0.1:3306)/mansooba?charset=utf8mb4&parseTime=True&loc=Local'
//
// The test creates every table, then runs the migration a second time.
func TestMigrate_MySQL_FreshDatabase(t *testing.T) {
	dsn := os.Getenv("MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("MYSQL_TEST_DSN not set")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, Migrate(db), "first migration on an empty database")
	require.NoError(t, Migrate(db), "migrating again must be a no-op")
}
