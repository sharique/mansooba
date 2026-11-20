package database

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// taskCountersMarker records that the one-time task counter step has run.
const taskCountersMarker = "task_counters_step_done"

// runTaskCountersOnce gives every existing project a task counter the first
// time only. The counters and the marker are written in one transaction, so a
// run that stops half way leaves nothing behind and is simply run again. A fresh
// database has no projects, so only the marker is recorded. It returns how many
// counters it raised.
func runTaskCountersOnce(db *gorm.DB) (raised int, highest uint, err error) {
	var done int64
	if err := db.Model(&domain.GlobalSetting{}).Where("setting_key = ?", taskCountersMarker).Count(&done).Error; err != nil {
		return 0, 0, fmt.Errorf("read upgrade marker: %w", err)
	}
	if done > 0 {
		return 0, 0, nil
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		var e error
		raised, highest, e = setTaskCounters(tx)
		if e != nil {
			return e
		}
		if e := tx.Create(&domain.GlobalSetting{SettingKey: taskCountersMarker, SettingValue: "1"}).Error; e != nil {
			return fmt.Errorf("record upgrade marker: %w", e)
		}
		return nil
	})
	if err != nil {
		// Another server starting at the same moment may have finished the step
		// first: its marker is there now, and its counters are what we would have
		// set (a counter only ever goes up), so this start has nothing left to do.
		if db.Model(&domain.GlobalSetting{}).Where("setting_key = ?", taskCountersMarker).Count(&done).Error == nil && done > 0 {
			return 0, 0, nil
		}
		return 0, 0, err
	}
	return raised, highest, nil
}

// setTaskCounters raises each project's counter to the largest of: the number of
// its tasks, the highest number in its tasks' keys, and the highest number in the
// keys recorded when its tasks were deleted. Nothing is renamed and a counter is
// never lowered. The number is read in Go, from the text after the key's last
// dash, because turning text into a number differs between the databases.
func setTaskCounters(tx *gorm.DB) (raised int, highest uint, err error) {
	type row struct {
		ProjectID uint
		Key       string
	}
	best := map[uint]uint{}
	note := func(projectID uint, n uint) {
		if n > best[projectID] {
			best[projectID] = n
		}
	}

	var counts []struct {
		ProjectID uint
		N         int64
	}
	if err := tx.Model(&domain.Issue{}).Select("project_id, count(*) AS n").Group("project_id").Scan(&counts).Error; err != nil {
		return 0, 0, fmt.Errorf("count tasks: %w", err)
	}
	for _, c := range counts {
		note(c.ProjectID, uint(c.N))
	}

	var keys []row
	if err := tx.Model(&domain.Issue{}).Select("project_id, " + tx.Statement.Quote("key") + " AS " + tx.Statement.Quote("key")).Scan(&keys).Error; err != nil {
		return 0, 0, fmt.Errorf("read task keys: %w", err)
	}
	for _, k := range keys {
		note(k.ProjectID, numberAfterLastDash(k.Key))
	}

	var deleted []struct {
		ProjectID uint
		IssueKey  string
	}
	if err := tx.Model(&domain.ActivityEvent{}).Where("kind = ?", domain.ActivityIssueDeleted).Select("project_id, issue_key").Scan(&deleted).Error; err != nil {
		return 0, 0, fmt.Errorf("read deletion history: %w", err)
	}
	for _, d := range deleted {
		note(d.ProjectID, numberAfterLastDash(d.IssueKey))
	}

	for projectID, n := range best {
		res := tx.Model(&domain.Project{}).Where("id = ? AND task_counter < ?", projectID, n).UpdateColumn("task_counter", n)
		if res.Error != nil {
			return raised, highest, fmt.Errorf("set task counter: %w", res.Error)
		}
		raised += int(res.RowsAffected)
		if res.RowsAffected > 0 && n > highest {
			highest = n
		}
	}
	return raised, highest, nil
}

// numberAfterLastDash reads the number from a key such as "PROJ-12" or
// "MY-PROJ-4". Text that does not end in a number gives 0.
func numberAfterLastDash(key string) uint {
	i := strings.LastIndex(key, "-")
	if i < 0 {
		return 0
	}
	n, err := strconv.ParseUint(key[i+1:], 10, 64)
	if err != nil {
		return 0
	}
	return uint(n)
}

func logTaskCountersStep(raised int, highest uint) {
	if logger.Logger == nil || raised == 0 {
		return
	}
	logger.Logger.Info("task counters set", zap.Int("projects", raised), zap.Uint("highest_counter", highest))
}
