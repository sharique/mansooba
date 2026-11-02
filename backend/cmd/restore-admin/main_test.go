package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/sharique/mansooba/pkg/config"
	"github.com/sharique/mansooba/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type recordedLog struct{ entries []domain.SystemLogEntry }

func (r *recordedLog) Record(_ context.Context, e domain.SystemLogEntry) {
	r.entries = append(r.entries, e)
}

func newDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.Open(&config.Config{DBDriver: "sqlite", DBDSN: ":memory:"})
	require.NoError(t, err)
	require.NoError(t, database.Migrate(db))
	return db
}

func addUser(t *testing.T, db *gorm.DB, email string, isAdmin, isActive bool) *domain.User {
	t.Helper()
	repo := repository.NewUserRepository(db)
	u := &domain.User{Name: email, Email: email, Username: email[:3] + "x", Password: "h", IsAdmin: isAdmin, IsActive: true}
	require.NoError(t, repo.Create(context.Background(), u))
	if !isActive {
		u.IsActive = false
		require.NoError(t, repo.UpdateAdminFields(context.Background(), u))
	}
	return u
}

func TestRestoreAdmin_EnablesAndPromotesTheAccount(t *testing.T) {
	db := newDB(t)
	u := addUser(t, db, "sys@example.com", true, false) // the only administrator, disabled
	var out, errOut bytes.Buffer
	log := &recordedLog{}

	code := restore(context.Background(), repository.NewUserRepository(db), log, "sys@example.com", &out, &errOut)

	assert.Equal(t, 0, code, errOut.String())
	got, err := repository.NewUserRepository(db).FindByID(context.Background(), u.ID)
	require.NoError(t, err)
	assert.True(t, got.IsAdmin && got.IsActive)
	require.Len(t, log.entries, 1)
	assert.Equal(t, "system_admin_restored", log.entries[0].Action)
	assert.Equal(t, domain.SystemLogCategoryAdminAction, log.entries[0].Category)
	assert.Contains(t, out.String(), "sys@example.com")
}

func TestRestoreAdmin_PromotesAnOrdinaryAccountToo(t *testing.T) {
	db := newDB(t)
	addUser(t, db, "old@example.com", true, false)
	u := addUser(t, db, "ben@example.com", false, true)
	var out, errOut bytes.Buffer
	assert.Equal(t, 0, restore(context.Background(), repository.NewUserRepository(db), &recordedLog{}, "ben@example.com", &out, &errOut), errOut.String())
	got, _ := repository.NewUserRepository(db).FindByID(context.Background(), u.ID)
	assert.True(t, got.IsAdmin && got.IsActive)
}

func TestRestoreAdmin_UnknownEmailChangesNothingAndFails(t *testing.T) {
	db := newDB(t)
	u := addUser(t, db, "sys@example.com", true, false)
	var out, errOut bytes.Buffer
	log := &recordedLog{}

	code := restore(context.Background(), repository.NewUserRepository(db), log, "ghost@example.com", &out, &errOut)

	assert.NotEqual(t, 0, code)
	assert.Contains(t, errOut.String(), "ghost@example.com")
	got, _ := repository.NewUserRepository(db).FindByID(context.Background(), u.ID)
	assert.False(t, got.IsActive, "nothing changed")
	assert.Empty(t, log.entries)
}

func TestRestoreAdmin_IsSafeToRepeat(t *testing.T) {
	db := newDB(t)
	addUser(t, db, "sys@example.com", true, false)
	var out, errOut bytes.Buffer
	repo := repository.NewUserRepository(db)
	assert.Equal(t, 0, restore(context.Background(), repo, &recordedLog{}, "sys@example.com", &out, &errOut))
	assert.Equal(t, 0, restore(context.Background(), repo, &recordedLog{}, "sys@example.com", &out, &errOut), errOut.String())
}

func TestRestoreAdmin_NeedsAnEmailArgument(t *testing.T) {
	var out, errOut bytes.Buffer
	assert.NotEqual(t, 0, run(nil, &out, &errOut))
	assert.Contains(t, errOut.String(), "--email")
}
