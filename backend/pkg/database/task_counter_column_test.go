package database_test

import (
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/testdb"
	"github.com/sharique/mansooba/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Each project keeps a counter of the last task number it issued. It cannot be
// empty and starts at 0, so the first task of a new project is number 1.
func TestMigrate_ProjectsHaveATaskCounterThatStartsAtZero(t *testing.T) {
	db := testdb.NewEmpty(t)
	require.NoError(t, database.Migrate(db))

	cols, err := db.Migrator().ColumnTypes(&domain.Project{})
	require.NoError(t, err)
	var found bool
	for _, c := range cols {
		if c.Name() != "task_counter" {
			continue
		}
		found = true
		nullable, known := c.Nullable()
		assert.True(t, known)
		assert.False(t, nullable, "task_counter must not be nullable")
	}
	require.True(t, found, "projects.task_counter does not exist")

	// A project inserted without a value (the way an older version would) reads back 0:
	// the database default, not the application, supplies it.
	require.NoError(t, db.Table("projects").Create(map[string]any{"key": "CNT", "name": "Counter", "owner_id": 1}).Error)
	var counter uint
	require.NoError(t, db.Table("projects").Select("task_counter").Where("name = ?", "Counter").Scan(&counter).Error)
	assert.Equal(t, uint(0), counter)
}
