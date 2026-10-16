package database

import (
	"github.com/sharique/mansooba/internal/domain"
	"gorm.io/gorm"
)

// Migrate runs GORM AutoMigrate for all domain models.
// It creates or updates tables to match the current struct definitions.
// Called once at server startup before the HTTP listener begins.
func Migrate(db *gorm.DB) error {
	// Existing databases predate usernames: add the column first (nullable, so the
	// statement works on a populated table) and refuse to go on while accounts
	// still lack one; cmd/backfill-usernames fills them in (ADR-038).
	if err := PrepareUsernames(db); err != nil {
		return err
	}
	if n, err := countMissingUsernames(db); err != nil {
		return err
	} else if n > 0 {
		return &UsernamesMissingError{Count: n}
	}
	if err := autoMigrateAll(db); err != nil {
		return err
	}
	return enforceUsernameNotNull(db)
}

func autoMigrateAll(db *gorm.DB) error {
	return db.AutoMigrate(
		&domain.User{},
		&domain.RevokedToken{},
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
		&domain.Attachment{},
	)
}
