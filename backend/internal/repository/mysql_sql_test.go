package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/sharique/mansooba/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// sqlRecorder is a GORM logger that keeps the SQL of every statement.
type sqlRecorder struct{ sqls *[]string }

func (r sqlRecorder) LogMode(logger.LogLevel) logger.Interface { return r }
func (r sqlRecorder) Info(context.Context, string, ...any)     {}
func (r sqlRecorder) Warn(context.Context, string, ...any)     {}
func (r sqlRecorder) Error(context.Context, string, ...any)    {}
func (r sqlRecorder) Trace(_ context.Context, _ time.Time, fc func() (string, int64), _ error) {
	sql, _ := fc()
	*r.sqls = append(*r.sqls, sql)
}

// mysqlDryRun is GORM with the MySQL dialect that never connects or runs anything: it only
// writes the SQL it would send, which the test reads back. MySQL treats some everyday
// words as reserved (`key`, for one), so SQL that is fine on SQLite and PostgreSQL can be a
// syntax error there.
func mysqlDryRun(t *testing.T) (*gorm.DB, *[]string) {
	t.Helper()
	var sqls []string
	db, err := gorm.Open(
		mysql.New(mysql.Config{DSN: "u:p@tcp(127.0.0.1:1)/none", SkipInitializeWithVersion: true}),
		&gorm.Config{DryRun: true, DisableAutomaticPing: true, Logger: sqlRecorder{sqls: &sqls}},
	)
	require.NoError(t, err)
	return db, &sqls
}

func TestProjectRepository_FindByKey_QuotesTheReservedWordKeyOnMySQL(t *testing.T) {
	db, sqls := mysqlDryRun(t)
	_, _ = repository.NewProjectRepository(db).FindByKey(context.Background(), "proj")

	require.NotEmpty(t, *sqls, "the lookup should have produced a statement")
	got := (*sqls)[len(*sqls)-1]
	assert.Contains(t, got, "LOWER(`key`)", "an unquoted key is a syntax error on MySQL")
	assert.NotContains(t, got, "LOWER(key)")
}
