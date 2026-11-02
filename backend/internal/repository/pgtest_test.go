package repository_test

import (
	"os"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// newPostgresTestDB opens the PostgreSQL named by POSTGRES_TEST_DSN and gives
// the test a clean schema. Without the variable the test is skipped, unless
// REQUIRE_POSTGRES=1 (CI sets it), when a missing database fails the build
// instead, the same convention as REQUIRE_LOCALSTACK.
//
// Locally:
//
//	docker run --rm -d -p 15432:5432 -e POSTGRES_USER=mansooba -e POSTGRES_PASSWORD=test \
//	  -e POSTGRES_DB=mansooba_test postgres:18-alpine
//	POSTGRES_TEST_DSN="host=localhost user=mansooba password=test dbname=mansooba_test port=15432 sslmode=disable" \
//	  go test ./internal/repository/ -run LastAdmin
func newPostgresTestDB(t *testing.T, models ...any) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("POSTGRES_TEST_DSN")
	if dsn == "" {
		if os.Getenv("REQUIRE_POSTGRES") == "1" {
			t.Fatal("REQUIRE_POSTGRES=1 but POSTGRES_TEST_DSN is not set")
		}
		t.Skip("POSTGRES_TEST_DSN not set; skipping the PostgreSQL concurrency test")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err, "open PostgreSQL")
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(20)
	t.Cleanup(func() { _ = sqlDB.Close() })

	// Start from nothing: concurrent tests must not see another test's rows.
	require.NoError(t, db.Exec(`DROP SCHEMA public CASCADE`).Error)
	require.NoError(t, db.Exec(`CREATE SCHEMA public`).Error)
	if len(models) == 0 {
		models = []any{&domain.User{}, &domain.Project{}, &domain.ProjectMember{}}
	}
	require.NoError(t, db.AutoMigrate(models...))
	return db
}
