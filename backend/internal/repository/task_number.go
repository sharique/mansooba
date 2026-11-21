package repository

import (
	"fmt"

	"github.com/sharique/mansooba/internal/domain"
	"gorm.io/gorm"
)

// nextTaskKey issues the project's next task number inside tx and returns the
// task's key, "<project key>-<number>".
//
// The counter on the project row is increased first. That UPDATE takes the row's
// exclusive lock for the rest of the transaction, so two simultaneous creates in
// one project run one after the other and each reads its own number, and it
// conflicts with a project delete, which takes the same lock. The new value is
// read back with a SELECT in the same transaction: PostgreSQL could return it
// from the UPDATE, but MySQL and MariaDB cannot, and this form behaves the same
// on all three. If the caller's insert then fails, the transaction rolls back and
// so does the counter: a failed create leaves no gap, and a number that was
// never shown to anyone is not used up.
//
// It returns domain.ErrNotFound when the project no longer exists. The columns
// are named through the model so `key`, a reserved word in MySQL and MariaDB, is
// quoted for whichever database is in use.
func nextTaskKey(tx *gorm.DB, projectID uint) (string, error) {
	res := tx.Model(&domain.Project{}).
		Where("id = ?", projectID).
		UpdateColumn("task_counter", gorm.Expr("task_counter + 1"))
	if res.Error != nil {
		return "", res.Error
	}
	if res.RowsAffected == 0 {
		return "", domain.ErrNotFound
	}
	var p domain.Project
	if err := tx.Select("key", "task_counter").First(&p, projectID).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%d", p.Key, p.TaskCounter), nil
}
