package database_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A large installation (100 projects of 100 tasks) must not wait on the one-time
// step: it has to finish well inside the start-up budget.
func TestTaskCounterStep_TenThousandTasksTakeUnderFiveSeconds(t *testing.T) {
	db := oldTaskDatabase(t)
	var tasks []domain.Issue
	for p := 1; p <= 100; p++ {
		key := fmt.Sprintf("P%d", p)
		id := addOldProject(t, db, key)
		for n := 1; n <= 100; n++ {
			tasks = append(tasks, domain.Issue{
				ProjectID: id, Key: fmt.Sprintf("%s-%d", key, n), Title: "t", Type: domain.IssueTypeTask,
				Status: domain.IssueStatusTodo, Priority: domain.IssuePriorityLow, ReporterID: 1,
			})
		}
	}
	require.NoError(t, db.CreateInBatches(tasks, 500).Error)

	// Migrate also upgrades the schema, so time the step on its own: add the column
	// first, then run Migrate again with the marker absent.
	require.NoError(t, database.Migrate(db))
	require.NoError(t, db.Table("projects").Where("1 = 1").UpdateColumn("task_counter", 0).Error)
	require.NoError(t, db.Where("setting_key = ?", taskCountersMarker).Delete(&domain.GlobalSetting{}).Error)

	start := time.Now()
	require.NoError(t, database.Migrate(db))
	elapsed := time.Since(start)
	t.Logf("task counter step over 10,000 tasks: %s", elapsed)

	assert.Less(t, elapsed, 5*time.Second)
	assert.Equal(t, uint(100), counter(t, db, 1))
}
