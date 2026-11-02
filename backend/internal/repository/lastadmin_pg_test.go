package repository_test

// Simultaneous requests on a real PostgreSQL. SQLite cannot show these races (its
// single connection already runs transactions one at a time), so the tests run
// only when POSTGRES_TEST_DSN is set; CI sets REQUIRE_POSTGRES=1 so they cannot
// be skipped there. See pgtest_test.go.

import (
	"context"
	"sync"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const raceRounds = 50

// runTogether starts every function at the same moment and returns their errors.
func runTogether(fns ...func() error) []error {
	var ready, done sync.WaitGroup
	start := make(chan struct{})
	errs := make([]error, len(fns))
	ready.Add(len(fns))
	done.Add(len(fns))
	for i, fn := range fns {
		go func() {
			defer done.Done()
			ready.Done()
			<-start
			errs[i] = fn()
		}()
	}
	ready.Wait()
	close(start)
	done.Wait()
	return errs
}

// ── project admins ───────────────────────────────────────────────────────────

func activeProjectAdmins(t *testing.T, db *gorm.DB, projectID uint) int64 {
	t.Helper()
	n, err := repository.NewProjectMemberRepository(db).CountActiveAdmins(context.Background(), projectID)
	require.NoError(t, err)
	return n
}

func TestLastAdmin_Project_TwoAdminsDemoteEachOtherAtOnce(t *testing.T) {
	db := newPostgresTestDB(t)
	f := newRolesRepoFixture(t, db)
	ctx := context.Background()
	require.NoError(t, f.members.ChangeRole(ctx, f.project.ID, f.ben.ID, "admin")) // ada and ben are admins

	refused := 0
	for round := 0; round < raceRounds; round++ {
		require.NoError(t, f.members.ChangeRole(ctx, f.project.ID, f.ada.ID, "admin"))
		require.NoError(t, f.members.ChangeRole(ctx, f.project.ID, f.ben.ID, "admin"))

		errs := runTogether(
			func() error { return f.members.ChangeRole(ctx, f.project.ID, f.ben.ID, "viewer") },
			func() error { return f.members.ChangeRole(ctx, f.project.ID, f.ada.ID, "viewer") },
		)
		require.GreaterOrEqual(t, activeProjectAdmins(t, db, f.project.ID), int64(1), "round %d left the project without an admin: %v", round, errs)
		for _, err := range errs {
			if err != nil {
				assert.ErrorIs(t, err, domain.ErrLastProjectAdmin)
				refused++
			}
		}
	}
	assert.Equal(t, raceRounds, refused, "exactly one of the two requests is refused every round")
}

func TestLastAdmin_Project_RemoveAndDemoteAtOnce(t *testing.T) {
	db := newPostgresTestDB(t)
	f := newRolesRepoFixture(t, db)
	ctx := context.Background()
	for round := 0; round < raceRounds; round++ {
		require.NoError(t, f.db.Exec(`DELETE FROM project_members WHERE project_id = ?`, f.project.ID).Error)
		for _, u := range []*domain.User{f.ada, f.ben} {
			require.NoError(t, f.members.Create(ctx, &domain.ProjectMember{ProjectID: f.project.ID, UserID: u.ID, Role: "admin"}))
		}
		errs := runTogether(
			func() error { return f.members.Remove(ctx, f.project.ID, f.ada.ID) },
			func() error { return f.members.ChangeRole(ctx, f.project.ID, f.ben.ID, "member") },
		)
		require.GreaterOrEqual(t, activeProjectAdmins(t, db, f.project.ID), int64(1), "round %d: %v", round, errs)
	}
}

// The control: the same change without the lock, forced to interleave. It must
// leave the project with no admin, which shows the tests above would notice.
func TestLastAdmin_Project_ControlWithoutTheLockLosesTheLastAdmin(t *testing.T) {
	db := newPostgresTestDB(t)
	f := newRolesRepoFixture(t, db)
	ctx := context.Background()
	require.NoError(t, f.members.ChangeRole(ctx, f.project.ID, f.ben.ID, "admin"))

	var updated, counted sync.WaitGroup
	updated.Add(2)
	counted.Add(2)
	naive := func(target uint) error {
		return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(`UPDATE project_members SET role = 'viewer' WHERE project_id = ? AND user_id = ?`, f.project.ID, target).Error; err != nil {
				return err
			}
			updated.Done()
			updated.Wait() // both have written, neither has committed
			var n int64
			tx.Raw(`SELECT COUNT(*) FROM project_members pm JOIN users u ON u.id = pm.user_id
				WHERE pm.project_id = ? AND pm.role = 'admin' AND u.is_active = true`, f.project.ID).Scan(&n)
			counted.Done()
			counted.Wait()
			if n == 0 {
				return domain.ErrLastProjectAdmin
			}
			return nil
		})
	}
	runTogether(func() error { return naive(f.ada.ID) }, func() error { return naive(f.ben.ID) })
	assert.Equal(t, int64(0), activeProjectAdmins(t, db, f.project.ID), "the unlocked version loses both admins; if this fails the race tests prove nothing")
}

// ── system administrators ────────────────────────────────────────────────────

func activeSystemAdmins(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	n, err := repository.NewUserRepository(db).CountActiveAdmins(context.Background())
	require.NoError(t, err)
	return n
}

func TestLastAdmin_System_TwoAdministratorsDemoteEachOtherAtOnce(t *testing.T) {
	db := newPostgresTestDB(t)
	ctx := context.Background()
	users := repository.NewUserRepository(db)
	a := createUser(t, users, "a@example.com", true, true)
	b := createUser(t, users, "b@example.com", true, true)

	refused := 0
	for round := 0; round < raceRounds; round++ {
		a.IsAdmin, b.IsAdmin = true, true
		require.NoError(t, users.UpdateAdminFields(ctx, a))
		require.NoError(t, users.UpdateAdminFields(ctx, b))

		demoteA, demoteB := *a, *b
		demoteA.IsAdmin, demoteB.IsAdmin = false, false
		errs := runTogether(
			func() error { return guarded(users, ctx, &demoteA) },
			func() error { return guarded(users, ctx, &demoteB) },
		)
		require.GreaterOrEqual(t, activeSystemAdmins(t, db), int64(1), "round %d: %v", round, errs)
		for _, err := range errs {
			if err != nil {
				assert.ErrorIs(t, err, domain.ErrLastAdmin)
				refused++
			}
		}
	}
	assert.Equal(t, raceRounds, refused)
}

func TestLastAdmin_System_DisableAndDemoteAtOnce(t *testing.T) {
	db := newPostgresTestDB(t)
	ctx := context.Background()
	users := repository.NewUserRepository(db)
	a := createUser(t, users, "a@example.com", true, true)
	b := createUser(t, users, "b@example.com", true, true)
	for round := 0; round < raceRounds; round++ {
		a.IsAdmin, a.IsActive, b.IsAdmin, b.IsActive = true, true, true, true
		require.NoError(t, users.UpdateAdminFields(ctx, a))
		require.NoError(t, users.UpdateAdminFields(ctx, b))
		disableA, demoteB := *a, *b
		disableA.IsActive = false
		demoteB.IsAdmin = false
		runTogether(
			func() error { return guarded(users, ctx, &disableA) },
			func() error { return guarded(users, ctx, &demoteB) },
		)
		require.GreaterOrEqual(t, activeSystemAdmins(t, db), int64(1), "round %d", round)
	}
}

func TestLastAdmin_System_ControlWithoutTheLockLosesTheLastAdministrator(t *testing.T) {
	db := newPostgresTestDB(t)
	ctx := context.Background()
	users := repository.NewUserRepository(db)
	a := createUser(t, users, "a@example.com", true, true)
	b := createUser(t, users, "b@example.com", true, true)

	var updated sync.WaitGroup
	updated.Add(2)
	naive := func(id uint) error {
		return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(`UPDATE users SET is_admin = false WHERE id = ?`, id).Error; err != nil {
				return err
			}
			updated.Done()
			updated.Wait()
			var n int64
			tx.Raw(`SELECT COUNT(*) FROM users WHERE is_admin = true AND is_active = true`).Scan(&n)
			if n == 0 {
				return domain.ErrLastAdmin
			}
			return nil
		})
	}
	runTogether(func() error { return naive(a.ID) }, func() error { return naive(b.ID) })
	assert.Equal(t, int64(0), activeSystemAdmins(t, db), "the unlocked version loses both administrators; if this fails the race tests prove nothing")
}
