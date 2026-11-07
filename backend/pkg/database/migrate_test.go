package database_test

import (
	"testing"

	"github.com/sharique/mansooba/internal/testdb"
	"github.com/sharique/mansooba/pkg/database"
	"github.com/stretchr/testify/require"
)

// The migration builds every table on an empty database and is a no-op the
// second time. Run on MySQL or MariaDB (TEST_DB_DRIVER) this is also the check
// that a fresh database of those engines can be migrated at all.
func TestMigrate_FreshDatabaseThenAgain(t *testing.T) {
	db := testdb.NewEmpty(t)
	require.NoError(t, database.Migrate(db), "first migration on an empty database")
	require.NoError(t, database.Migrate(db), "migrating again must be a no-op")
}
