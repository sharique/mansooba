package database_test

// The one-time upgrade that gives every existing project its task counter. The
// database starts out as an older version left it: a projects table without the
// counter, tasks with whatever numbers they have, deletion records in the
// activity history. Migrate must set each counter to a safe value, rename nothing,
// and be safe to repeat.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/sharique/mansooba/internal/testdb"
	"github.com/sharique/mansooba/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const taskCountersMarker = "task_counters_step_done"

// legacyProject is the projects table before task counters existed.
type legacyProject struct {
	ID          uint   `gorm:"primaryKey"`
	Key         string `gorm:"uniqueIndex;not null"`
	Name        string `gorm:"not null"`
	Description string
	OwnerID     uint `gorm:"not null;index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (legacyProject) TableName() string { return "projects" }

// oldTaskDatabase is an empty database with the pre-counter tables; nothing has
// been migrated yet.
func oldTaskDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	db := testdb.NewEmpty(t)
	require.NoError(t, database.SizeIndexedStrings(db, &legacyProject{})) // MySQL cannot index unsized text
	require.NoError(t, db.AutoMigrate(&legacyProject{}, &domain.Sprint{}, &domain.Issue{}, &domain.ActivityEvent{}, &domain.GlobalSetting{}))
	return db
}

func addOldProject(t *testing.T, db *gorm.DB, key string) uint {
	t.Helper()
	p := &legacyProject{Key: key, Name: key, OwnerID: 1}
	require.NoError(t, db.Create(p).Error)
	return p.ID
}

func addOldTask(t *testing.T, db *gorm.DB, projectID uint, key string) {
	t.Helper()
	require.NoError(t, db.Create(&domain.Issue{
		ProjectID: projectID, Key: key, Title: key, Type: domain.IssueTypeTask, Status: domain.IssueStatusTodo,
		Priority: domain.IssuePriorityLow, ReporterID: 1,
	}).Error)
}

// addDeletion records the deletion of a task the way v0.4.1 and later do.
func addDeletion(t *testing.T, db *gorm.DB, projectID uint, key string) {
	t.Helper()
	require.NoError(t, db.Create(&domain.ActivityEvent{
		IssueID: 1000, ActorID: 1, Kind: domain.ActivityIssueDeleted, ProjectID: projectID, IssueKey: key, IssueTitle: "gone",
	}).Error)
}

func counter(t *testing.T, db *gorm.DB, projectID uint) uint {
	t.Helper()
	var c uint
	require.NoError(t, db.Table("projects").Select("task_counter").Where("id = ?", projectID).Scan(&c).Error)
	return c
}

func allTaskKeys(t *testing.T, db *gorm.DB) map[uint]string {
	t.Helper()
	var issues []domain.Issue
	require.NoError(t, db.Order("id").Find(&issues).Error)
	out := map[uint]string{}
	for _, i := range issues {
		out[i.ID] = i.Key
	}
	return out
}

func markerCount(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&domain.GlobalSetting{}).Where("setting_key = ?", taskCountersMarker).Count(&n).Error)
	return n
}

func nextKey(t *testing.T, db *gorm.DB, projectID uint) string {
	t.Helper()
	issue := &domain.Issue{ProjectID: projectID, Title: "next", Type: domain.IssueTypeTask, Status: domain.IssueStatusTodo, Priority: domain.IssuePriorityLow, ReporterID: 1}
	require.NoError(t, repository.NewIssueRepository(db).Create(context.Background(), issue))
	return issue.Key
}

func TestTaskCounterStep_GapsStartAfterTheHighestNumber(t *testing.T) {
	db := oldTaskDatabase(t)
	p := addOldProject(t, db, "GAP")
	for _, n := range []int{1, 2, 3, 5} { // 4 was deleted long ago
		addOldTask(t, db, p, fmt.Sprintf("GAP-%d", n))
	}
	require.NoError(t, database.Migrate(db))
	assert.Equal(t, uint(5), counter(t, db, p))
	assert.Equal(t, "GAP-6", nextKey(t, db, p))
}

func TestTaskCounterStep_HighestNumberAboveTheTaskCount(t *testing.T) {
	db := oldTaskDatabase(t)
	p := addOldProject(t, db, "HIGH")
	for _, n := range []int{1, 2, 9} {
		addOldTask(t, db, p, fmt.Sprintf("HIGH-%d", n))
	}
	require.NoError(t, database.Migrate(db))
	assert.Equal(t, uint(9), counter(t, db, p), "3 tasks, but the highest number is 9: the next create used to fail")
	assert.Equal(t, "HIGH-10", nextKey(t, db, p))
}

func TestTaskCounterStep_ADeletionRecordHigherThanEveryLiveTask(t *testing.T) {
	db := oldTaskDatabase(t)
	p := addOldProject(t, db, "REC")
	for _, n := range []int{1, 2, 3, 4, 5} {
		addOldTask(t, db, p, fmt.Sprintf("REC-%d", n))
	}
	addDeletion(t, db, p, "REC-7")
	other := addOldProject(t, db, "OTH")
	addDeletion(t, db, other, "OTH-40") // another project's record must not leak into this one
	require.NoError(t, database.Migrate(db))

	assert.Equal(t, uint(7), counter(t, db, p), "a number named in the history is never issued again")
	assert.Equal(t, uint(40), counter(t, db, other))
	assert.Equal(t, "REC-8", nextKey(t, db, p))
}

func TestTaskCounterStep_AProjectWithNoTasksStartsAtOne(t *testing.T) {
	db := oldTaskDatabase(t)
	p := addOldProject(t, db, "EMPTY")
	require.NoError(t, database.Migrate(db))
	assert.Equal(t, uint(0), counter(t, db, p))
	assert.Equal(t, "EMPTY-1", nextKey(t, db, p))
}

func TestTaskCounterStep_TheTaskCountAloneIsAFloor(t *testing.T) {
	db := oldTaskDatabase(t)
	p := addOldProject(t, db, "ODD")
	// keys that do not end in a number are ignored for the highest number, but the tasks still count
	addOldTask(t, db, p, "ODD-abc")
	addOldTask(t, db, p, "something")
	addOldTask(t, db, p, "ODD-2")
	require.NoError(t, database.Migrate(db))
	assert.Equal(t, uint(3), counter(t, db, p), "3 tasks (count 3) beats the highest parsable number (2)")
}

func TestTaskCounterStep_AProjectKeyWithADashIsReadFromTheLastDash(t *testing.T) {
	db := oldTaskDatabase(t)
	p := addOldProject(t, db, "MY-PROJ")
	addOldTask(t, db, p, "MY-PROJ-4")
	require.NoError(t, database.Migrate(db))
	assert.Equal(t, uint(4), counter(t, db, p))
}

func TestTaskCounterStep_ChangesNoKeyAndRecordsItsMarker(t *testing.T) {
	db := oldTaskDatabase(t)
	a, b := addOldProject(t, db, "AAA"), addOldProject(t, db, "BBB")
	for _, k := range []string{"AAA-1", "AAA-3"} {
		addOldTask(t, db, a, k)
	}
	addOldTask(t, db, b, "BBB-2")
	before := allTaskKeys(t, db)

	require.NoError(t, database.Migrate(db))

	assert.Equal(t, before, allTaskKeys(t, db), "the upgrade renames nothing")
	assert.Equal(t, int64(1), markerCount(t, db))
	assert.Equal(t, uint(3), counter(t, db, a))
	assert.Equal(t, uint(2), counter(t, db, b), "projects are independent")
}

func TestTaskCounterStep_RunningMigrateAgainChangesNothing(t *testing.T) {
	db := oldTaskDatabase(t)
	p := addOldProject(t, db, "AGAIN")
	addOldTask(t, db, p, "AGAIN-6")
	require.NoError(t, database.Migrate(db))
	require.Equal(t, uint(6), counter(t, db, p))

	// the project issues more numbers; a later start must not recompute or lower anything
	require.NoError(t, db.Table("projects").Where("id = ?", p).UpdateColumn("task_counter", 50).Error)
	require.NoError(t, database.Migrate(db))
	require.NoError(t, database.Migrate(db))
	assert.Equal(t, uint(50), counter(t, db, p))
	assert.Equal(t, int64(1), markerCount(t, db), "the marker is written once")
}

// A run that was interrupted before it recorded its marker is simply run again,
// and it never lowers a counter that is already higher.
func TestTaskCounterStep_AnInterruptedRunCanBeRepeatedAndNeverLowersACounter(t *testing.T) {
	db := oldTaskDatabase(t)
	first, second := addOldProject(t, db, "FIRST"), addOldProject(t, db, "SECOND")
	addOldTask(t, db, first, "FIRST-4")
	addOldTask(t, db, second, "SECOND-8")
	require.NoError(t, database.Migrate(db))

	// as if the first run set only some counters and stopped before the marker
	require.NoError(t, db.Table("projects").Where("id = ?", first).UpdateColumn("task_counter", 99).Error)
	require.NoError(t, db.Table("projects").Where("id = ?", second).UpdateColumn("task_counter", 0).Error)
	require.NoError(t, db.Where("setting_key = ?", taskCountersMarker).Delete(&domain.GlobalSetting{}).Error)

	require.NoError(t, database.Migrate(db))
	assert.Equal(t, uint(99), counter(t, db, first), "a higher counter is never lowered")
	assert.Equal(t, uint(8), counter(t, db, second), "the unfinished project is completed")
	assert.Equal(t, int64(1), markerCount(t, db))
}

func TestTaskCounterStep_OnAFreshDatabaseOnlyRecordsTheMarker(t *testing.T) {
	db := testdb.NewEmpty(t)
	require.NoError(t, database.Migrate(db))
	assert.Equal(t, int64(1), markerCount(t, db))
	var projects int64
	require.NoError(t, db.Model(&domain.Project{}).Count(&projects).Error)
	assert.Zero(t, projects)
}

// The counter is part of the project's row, so a project copied to another
// database (a restore, a move) carries it and carries on from it.
func TestTaskCounter_TravelsWithTheProjectsRow(t *testing.T) {
	db := testdb.New(t)
	require.NoError(t, db.Create(&domain.Project{Key: "MOVED", Name: "Moved", OwnerID: 1, TaskCounter: 7}).Error)
	var p domain.Project
	require.NoError(t, db.Where(&domain.Project{Key: "MOVED"}).First(&p).Error)
	assert.Equal(t, "MOVED-8", nextKey(t, db, p.ID))
}

func TestTaskCounter_KeepsCountingPastThe32BitLimit(t *testing.T) {
	db := testdb.New(t)
	require.NoError(t, db.Create(&domain.Project{Key: "BIG", Name: "Big", OwnerID: 1, TaskCounter: 2147483647}).Error)
	var p domain.Project
	require.NoError(t, db.Where(&domain.Project{Key: "BIG"}).First(&p).Error)
	assert.Equal(t, "BIG-2147483648", nextKey(t, db, p.ID))
}

// Two servers starting at the same moment on the first upgrade both find the
// marker missing and both run the step. The one that finishes second sees the
// marker already written and must treat the step as done, not fail to start.
func TestTaskCounterStep_TwoStartsAtOnceBothSucceed(t *testing.T) {
	for round := 0; round < 5; round++ {
		db := oldTaskDatabase(t)
		p := addOldProject(t, db, "TWIN")
		for n := 1; n <= 300; n++ { // enough work that the starts overlap
			addOldTask(t, db, p, fmt.Sprintf("TWIN-%d", n))
		}
		require.NoError(t, database.Migrate(db)) // schema first
		require.NoError(t, db.Table("projects").Where("id = ?", p).UpdateColumn("task_counter", 0).Error)
		require.NoError(t, db.Where("setting_key = ?", taskCountersMarker).Delete(&domain.GlobalSetting{}).Error)

		const starts = 8
		errs := make(chan error, starts)
		start := make(chan struct{})
		for i := 0; i < starts; i++ {
			go func() {
				<-start
				_, _, err := database.RunTaskCountersOnce(db)
				errs <- err
			}()
		}
		close(start)
		for i := 0; i < starts; i++ {
			require.NoError(t, <-errs, "round %d", round)
		}
		assert.Equal(t, uint(300), counter(t, db, p))
		assert.Equal(t, int64(1), markerCount(t, db))
	}
}
