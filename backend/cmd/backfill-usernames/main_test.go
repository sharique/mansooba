package main

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/sharique/mansooba/internal/repository"
	"github.com/sharique/mansooba/pkg/config"
	"github.com/sharique/mansooba/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// legacyUser is the users table before usernames existed.
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

func legacyDatabase(t *testing.T, emails ...string) (dsn string) {
	t.Helper()
	dsn = filepath.Join(t.TempDir(), "backfill.db")
	db, err := database.Open(&config.Config{DBDriver: "sqlite", DBDSN: dsn})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&legacyUser{}))
	for _, e := range emails {
		require.NoError(t, db.Create(&legacyUser{Name: e, Email: e, Password: "h", IsActive: true}).Error)
	}
	closeDB(t, db)
	return dsn
}

func closeDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
}

func TestRun_UpgradesALegacyDatabaseAndIsIdempotent(t *testing.T) {
	dsn := legacyDatabase(t, "sam@a.com", "sam@b.com")
	t.Setenv("DB_DRIVER", "sqlite")
	t.Setenv("DB_DSN", dsn)
	t.Setenv("JWT_SECRET", "test-secret")

	var out, errOut bytes.Buffer
	assert.Equal(t, 0, run(&out, &errOut), errOut.String())
	assert.Contains(t, out.String(), "2 accounts changed")

	// The database is now fully migrated: every account has a username and the
	// server's own migration passes.
	db, err := database.Open(&config.Config{DBDriver: "sqlite", DBDSN: dsn})
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	users, err := repository.NewUserRepository(db).FindByEmail(context.Background(), "sam@b.com")
	require.NoError(t, err)
	assert.Equal(t, "sam2", users.Username)
	closeDB(t, db)

	out.Reset()
	assert.Equal(t, 0, run(&out, &errOut), errOut.String())
	assert.Contains(t, out.String(), "0 accounts changed")
}

func TestRun_OnAFreshDatabaseJustMigrates(t *testing.T) {
	t.Setenv("DB_DRIVER", "sqlite")
	t.Setenv("DB_DSN", filepath.Join(t.TempDir(), "fresh.db"))
	t.Setenv("JWT_SECRET", "test-secret")
	var out, errOut bytes.Buffer
	assert.Equal(t, 0, run(&out, &errOut), errOut.String())
	assert.Contains(t, out.String(), "0 accounts changed")
}

func TestRun_FailsWhenTheDatabaseCannotBeOpened(t *testing.T) {
	t.Setenv("DB_DRIVER", "nonsense")
	t.Setenv("DB_DSN", "x")
	t.Setenv("JWT_SECRET", "test-secret")
	var out, errOut bytes.Buffer
	assert.NotEqual(t, 0, run(&out, &errOut))
	assert.NotEmpty(t, errOut.String())
}
