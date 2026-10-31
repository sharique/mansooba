package database

import (
	"github.com/sharique/mansooba/internal/domain"
	"gorm.io/gorm"
)

// Migrate runs GORM AutoMigrate for all domain models.
// It creates or updates tables to match the current struct definitions.
// Called once at server startup before the HTTP listener begins.
func Migrate(db *gorm.DB) error {
	// MySQL cannot index the longtext columns GORM picks for `uniqueIndex` strings;
	// size them before anything is migrated (see sizeIndexedStrings).
	if err := sizeIndexedStrings(db, migrationModels()...); err != nil {
		return err
	}
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

// migrationModels lists every model the app stores, in the order they are migrated.
func migrationModels() []any {
	return []any{
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
	}
}

func autoMigrateAll(db *gorm.DB) error {
	return db.AutoMigrate(migrationModels()...)
}
