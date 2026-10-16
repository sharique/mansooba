package database

import (
	"fmt"

	"github.com/sharique/mansooba/internal/domain"
	"gorm.io/gorm"
)

// UsernamesMissingError is returned by Migrate when accounts exist without a
// username. Run the one-off backfill-usernames command (ADR-038), then start
// the server.
type UsernamesMissingError struct{ Count int }

func (e *UsernamesMissingError) Error() string {
	return fmt.Sprintf("%d accounts have no username yet: run ./backfill-usernames (or go run ./cmd/backfill-usernames) before starting the server", e.Count)
}

// PrepareUsernames gets a database that predates usernames ready to be
// backfilled: when the users table exists without a username column it adds the
// column, nullable (adding a NOT NULL column to a populated table fails on
// PostgreSQL and SQLite), together with its unique index (NULLs do not collide).
// It does nothing on a fresh database, where Migrate creates the final NOT NULL
// column directly, and is safe to run repeatedly.
func PrepareUsernames(db *gorm.DB) error {
	m := db.Migrator()
	if !m.HasTable(&domain.User{}) || m.HasColumn(&domain.User{}, "Username") {
		return nil
	}
	if err := db.Exec(`ALTER TABLE users ADD COLUMN username varchar(30)`).Error; err != nil {
		return fmt.Errorf("add users.username: %w", err)
	}
	if err := db.Exec(`CREATE UNIQUE INDEX idx_users_username ON users(username)`).Error; err != nil {
		return fmt.Errorf("index users.username: %w", err)
	}
	return nil
}

// countMissingUsernames counts accounts without a username; zero when the table
// or column does not exist yet.
func countMissingUsernames(db *gorm.DB) (int, error) {
	m := db.Migrator()
	if !m.HasTable(&domain.User{}) || !m.HasColumn(&domain.User{}, "Username") {
		return 0, nil
	}
	var n int64
	if err := db.Raw(`SELECT COUNT(*) FROM users WHERE username IS NULL OR username = ''`).Scan(&n).Error; err != nil {
		return 0, err
	}
	return int(n), nil
}

// enforceUsernameNotNull makes sure the column is NOT NULL once every account
// has a username. SQLite's AutoMigrate already rebuilds the table that way;
// PostgreSQL and MySQL leave an existing nullable column alone, so tighten it
// here. Idempotent, and a no-op on a fresh database.
func enforceUsernameNotNull(db *gorm.DB) error {
	cols, err := db.Migrator().ColumnTypes(&domain.User{})
	if err != nil {
		return err
	}
	for _, c := range cols {
		if c.Name() != "username" {
			continue
		}
		if nullable, known := c.Nullable(); known && nullable {
			return db.Migrator().AlterColumn(&domain.User{}, "Username")
		}
	}
	return nil
}
