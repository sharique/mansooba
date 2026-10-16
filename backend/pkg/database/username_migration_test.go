package database_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/sharique/mansooba/internal/service"
	"github.com/sharique/mansooba/pkg/config"
	"github.com/sharique/mansooba/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// legacyUser is the users table as it was before usernames existed.
type legacyUser struct {
	ID                               uint   `gorm:"primaryKey"`
	Name                             string `gorm:"not null"`
	Email                            string `gorm:"uniqueIndex;not null"`
	Password                         string `gorm:"not null"`
	AvatarURL                        string
	Timezone                         string
	IsAdmin                          bool `gorm:"not null;default:false"`
	IsActive                         bool `gorm:"not null;default:true"`
	CreatedAt                        time.Time
	UpdatedAt                        time.Time
	TokenValidAfter                  *time.Time
	ConsecutiveFailedPasswordChanges int `gorm:"not null;default:0"`
}

func (legacyUser) TableName() string { return "users" }

func usernameColumnNullable(t *testing.T, db *gorm.DB) bool {
	t.Helper()
	cols, err := db.Migrator().ColumnTypes(&domain.User{})
	require.NoError(t, err)
	for _, c := range cols {
		if c.Name() == "username" {
			nullable, known := c.Nullable()
			require.True(t, known)
			return nullable
		}
	}
	require.Fail(t, "no username column")
	return false
}

func openSQLite(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.Open(&config.Config{DBDriver: "sqlite", DBDSN: ":memory:"})
	require.NoError(t, err)
	return db
}

// A fresh database gets the final schema straight away: NOT NULL and unique,
// enforced by the database itself.
func TestMigrate_FreshDatabaseHasANonNullUniqueUsername(t *testing.T) {
	db := openSQLite(t)
	require.NoError(t, database.Migrate(db))

	assert.False(t, usernameColumnNullable(t, db))
	assert.Error(t, db.Exec(`INSERT INTO users (name, email, password) VALUES ('x', 'x@x.com', 'h')`).Error, "no username is refused")
	require.NoError(t, db.Exec(`INSERT INTO users (name, email, username, password) VALUES ('a', 'a@x.com', 'sam', 'h')`).Error)
	assert.Error(t, db.Exec(`INSERT INTO users (name, email, username, password) VALUES ('b', 'b@x.com', 'sam', 'h')`).Error, "a duplicate is refused")
}

// An existing deployment: prepare (add the column, nullable), backfill, then
// Migrate, which makes the column NOT NULL. Awkward emails get distinct valid
// usernames and nothing else about the accounts changes.
func TestUpgrade_LegacyUsersTableEndsNonNull(t *testing.T) {
	upgradeLegacyUsers(t, openSQLite(t))
}

// The same upgrade on PostgreSQL, the production database. CI runs SQLite only,
// so this runs when USERNAME_MIGRATION_PG_DSN points at a throwaway PostgreSQL
// (for example a postgres:17 container on a spare port).
func TestUpgrade_LegacyUsersTableEndsNonNull_PostgreSQL(t *testing.T) {
	dsn := os.Getenv("USERNAME_MIGRATION_PG_DSN")
	if dsn == "" {
		t.Skip("set USERNAME_MIGRATION_PG_DSN to run against PostgreSQL")
	}
	db, err := database.Open(&config.Config{DBDriver: "postgres", DBDSN: dsn})
	require.NoError(t, err)
	for _, tbl := range []string{"notifications", "users"} {
		require.NoError(t, db.Exec(`DROP TABLE IF EXISTS `+tbl+` CASCADE`).Error)
	}
	upgradeLegacyUsers(t, db)
}

func upgradeLegacyUsers(t *testing.T, db *gorm.DB) {
	t.Helper()
	ctx := context.Background()
	require.NoError(t, db.AutoMigrate(&legacyUser{}))
	emails := []string{"sam-smith@x.com", "sam@a.com", "sam@b.com", "admin@x.com", "ab@x.com"}
	for _, e := range emails {
		require.NoError(t, db.Create(&legacyUser{Name: e, Email: e, Password: "h", IsActive: true}).Error)
	}

	// The server refuses to start (Migrate fails) until the backfill has run,
	// and says what to do.
	err := database.Migrate(db)
	var missing *database.UsernamesMissingError
	require.ErrorAs(t, err, &missing)
	assert.Equal(t, len(emails), missing.Count)
	assert.Contains(t, err.Error(), "backfill-usernames")

	// The one-off command's sequence: prepare, backfill, migrate.
	require.NoError(t, database.PrepareUsernames(db))
	repo := repository.NewUserRepository(db)
	svc := service.NewUsernameService(repo)
	n, err := svc.CountMissing(ctx)
	require.NoError(t, err)
	assert.Equal(t, len(emails), n)
	changed, err := svc.BackfillMissing(ctx)
	require.NoError(t, err)
	assert.Equal(t, len(emails), changed)
	require.NoError(t, database.Migrate(db))

	want := map[string]string{"sam-smith@x.com": "samsmith", "sam@a.com": "sam", "sam@b.com": "sam2", "admin@x.com": "admin2", "ab@x.com": "userab"}
	seen := map[string]bool{}
	for email, name := range want {
		u, err := repo.FindByEmail(ctx, email)
		require.NoError(t, err)
		assert.Equal(t, name, u.Username, email)
		assert.NoError(t, domain.ValidateUsername(u.Username))
		assert.False(t, seen[u.Username], "usernames are unique")
		seen[u.Username] = true
	}

	// The constraint is now enforced by the database.
	assert.False(t, usernameColumnNullable(t, db))
	assert.Error(t, db.Exec(`INSERT INTO users (name, email, password) VALUES ('x', 'x@x.com', 'h')`).Error)
	assert.Error(t, db.Exec(`INSERT INTO users (name, email, username, password) VALUES ('x', 'y@x.com', 'sam', 'h')`).Error)

	// Everything is idempotent: a second run changes nothing and the server's own migration passes.
	require.NoError(t, database.PrepareUsernames(db))
	changed, err = svc.BackfillMissing(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, changed)
	require.NoError(t, database.Migrate(db))
}
