package database

import (
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Code that creates tables with AutoMigrate itself (test helpers building an old
// schema by hand) has to size the indexed text columns first, or MySQL refuses
// them (Error 1170). PrepareModels does for such code what Migrate does first.
func TestPrepareModels_SizesIndexedStringsOnMySQL(t *testing.T) {
	db := mysqlWithoutServer(t)
	require.NoError(t, PrepareModels(db))

	stmt := &gorm.Statement{DB: db}
	require.NoError(t, stmt.Parse(&domain.User{}))
	assert.Equal(t, mysqlIndexedStringSize, stmt.Schema.FieldsByDBName["email"].Size)
}
