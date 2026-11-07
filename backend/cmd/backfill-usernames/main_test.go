package main

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/sharique/mansooba/internal/repository"
	"github.com/sharique/mansooba/internal/testdb"
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

// legacyDatabase creates an empty database holding only the old users table
// (no username column) with the given accounts, and returns its settings.
func legacyDatabase(t *testing.T, emails ...string) *config.Config {
	t.Helper()
	cfg := testdb.ConfigEmpty(t)
	db, err := database.Open(cfg)
	require.NoError(t, err)
	require.NoError(t, database.SizeIndexedStrings(db, &legacyUser{})) // MySQL cannot index unsized text
	require.NoError(t, db.AutoMigrate(&legacyUser{}))
	for _, e := range emails {
		require.NoError(t, db.Create(&legacyUser{Name: e, Email: e, Password: "h", IsActive: true}).Error)
	}
	closeDB(t, db)
	return cfg
}

// useDatabase points the command at cfg the way an operator would: through the
// environment.
func useDatabase(t *testing.T, cfg *config.Config) {
	t.Helper()
	t.Setenv("DB_DRIVER", cfg.DBDriver)
	t.Setenv("DB_DSN", cfg.DBDSN)
	t.Setenv("JWT_SECRET", "test-secret")
}

func closeDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
}

func TestRun_UpgradesALegacyDatabaseAndIsIdempotent(t *testing.T) {
	cfg := legacyDatabase(t, "sam@a.com", "sam@b.com")
	useDatabase(t, cfg)

	var out, errOut bytes.Buffer
	assert.Equal(t, 0, run(&out, &errOut), errOut.String())
	assert.Contains(t, out.String(), "2 accounts changed")

	// The database is now fully migrated: every account has a username and the
	// server's own migration passes.
	db, err := database.Open(cfg)
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
	useDatabase(t, testdb.ConfigEmpty(t))
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
