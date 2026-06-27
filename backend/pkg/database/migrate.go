package database

import (
	"github.com/sharique/mansooba/internal/domain"
	"gorm.io/gorm"
)

// Migrate runs GORM AutoMigrate for all domain models.
// It creates or updates tables to match the current struct definitions.
// Called once at server startup before the HTTP listener begins.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&domain.User{},
		&domain.Project{},
		&domain.ProjectMember{},
		&domain.Sprint{},
		&domain.Label{},
		&domain.Issue{},
		&domain.Comment{},
		&domain.ActivityEvent{},
		&domain.IssueLabel{},
		&domain.Notification{},
		&domain.GlobalSetting{},
		&domain.IssueRelation{},
		&domain.PasswordResetToken{},
	)
}
