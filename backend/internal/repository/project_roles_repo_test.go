package repository_test

// Last-admin guards and the project queries behind role management, on real
// SQLite. The simultaneous-request cases are in project_lastadmin_pg_test.go
// (PostgreSQL).

import (
	"context"
	"errors"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type rolesRepoFixture struct {
	db      *gorm.DB
	members domain.ProjectMemberRepository
	project *domain.Project
	ada     *domain.User // admin
	ben     *domain.User // member
	cy      *domain.User // admin, second
}

func newRolesRepoFixture(t *testing.T, db *gorm.DB) *rolesRepoFixture {
	t.Helper()
	ctx := context.Background()
	users := repository.NewUserRepository(db)
	f := &rolesRepoFixture{db: db, members: repository.NewProjectMemberRepository(db)}
	f.ada = createUser(t, users, "ada@example.com", false, true)
	f.ben = createUser(t, users, "ben@example.com", false, true)
	f.cy = createUser(t, users, "cy@example.com", false, true)
	f.project = &domain.Project{Key: "demo", Name: "Demo", OwnerID: f.ada.ID}
	require.NoError(t, repository.NewProjectRepository(db).Create(ctx, f.project))
	for u, role := range map[*domain.User]string{f.ada: "admin", f.ben: "member"} {
		require.NoError(t, f.members.Create(ctx, &domain.ProjectMember{ProjectID: f.project.ID, UserID: u.ID, Role: role}))
	}
	return f
}

func roleOf(t *testing.T, f *rolesRepoFixture, u *domain.User) string {
	t.Helper()
	m, err := f.members.FindByProjectAndUser(context.Background(), f.project.ID, u.ID)
	require.NoError(t, err)
	return m.Role
}

func TestMemberRepo_ChangeRole_Works(t *testing.T) {
	f := newRolesRepoFixture(t, newTestDB(t))
	ctx := context.Background()
	require.NoError(t, f.members.ChangeRole(ctx, f.project.ID, f.ben.ID, "admin"))
	assert.Equal(t, "admin", roleOf(t, f, f.ben))
	require.NoError(t, f.members.ChangeRole(ctx, f.project.ID, f.ben.ID, "viewer"))
	assert.Equal(t, "viewer", roleOf(t, f, f.ben))
	assert.ErrorIs(t, f.members.ChangeRole(ctx, f.project.ID, f.cy.ID, "member"), domain.ErrNotFound, "cy is not a member")
}

func TestMemberRepo_ChangeRole_RefusesToDemoteTheOnlyActiveAdmin(t *testing.T) {
	f := newRolesRepoFixture(t, newTestDB(t))
	ctx := context.Background()
	assert.ErrorIs(t, f.members.ChangeRole(ctx, f.project.ID, f.ada.ID, "member"), domain.ErrLastProjectAdmin)
	assert.Equal(t, "admin", roleOf(t, f, f.ada), "nothing changed")

	// with a second admin the first may step down
	require.NoError(t, f.members.ChangeRole(ctx, f.project.ID, f.ben.ID, "admin"))
	require.NoError(t, f.members.ChangeRole(ctx, f.project.ID, f.ada.ID, "viewer"))
	assert.Equal(t, "viewer", roleOf(t, f, f.ada))
	// ...and now ben is the last one
	assert.ErrorIs(t, f.members.ChangeRole(ctx, f.project.ID, f.ben.ID, "member"), domain.ErrLastProjectAdmin)
}

// A disabled admin does not count: with one active and one disabled admin the
// active one is the last.
func TestMemberRepo_DisabledAdminsDoNotCount(t *testing.T) {
	f := newRolesRepoFixture(t, newTestDB(t))
	ctx := context.Background()
	require.NoError(t, f.members.ChangeRole(ctx, f.project.ID, f.ben.ID, "admin"))
	require.NoError(t, f.db.Model(&domain.User{}).Where("id = ?", f.ben.ID).Update("is_active", false).Error)

	n, err := f.members.CountActiveAdmins(ctx, f.project.ID)
	require.NoError(t, err)
	assert.EqualValues(t, 1, n)
	assert.ErrorIs(t, f.members.ChangeRole(ctx, f.project.ID, f.ada.ID, "member"), domain.ErrLastProjectAdmin)
	assert.ErrorIs(t, f.members.Remove(ctx, f.project.ID, f.ada.ID), domain.ErrLastProjectAdmin)
}

func TestMemberRepo_Remove(t *testing.T) {
	f := newRolesRepoFixture(t, newTestDB(t))
	ctx := context.Background()
	require.NoError(t, f.members.Remove(ctx, f.project.ID, f.ben.ID))
	_, err := f.members.FindByProjectAndUser(ctx, f.project.ID, f.ben.ID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.ErrorIs(t, f.members.Remove(ctx, f.project.ID, f.ben.ID), domain.ErrNotFound)
	assert.ErrorIs(t, f.members.Remove(ctx, f.project.ID, f.ada.ID), domain.ErrLastProjectAdmin)
}

// FR-002 / one membership per person per project.
func TestMemberRepo_SecondMembershipIsAConflict(t *testing.T) {
	f := newRolesRepoFixture(t, newTestDB(t))
	err := f.members.Create(context.Background(), &domain.ProjectMember{ProjectID: f.project.ID, UserID: f.ben.ID, Role: "viewer"})
	assert.True(t, errors.Is(err, domain.ErrConflict), "got %v", err)
}

func TestMemberRepo_AddAdminIfNone(t *testing.T) {
	f := newRolesRepoFixture(t, newTestDB(t))
	ctx := context.Background()
	rescuer := createUser(t, repository.NewUserRepository(f.db), "sys@example.com", true, true)

	// a project that still has an active admin is left alone
	_, err := f.members.AddAdminIfNone(ctx, f.project.ID, rescuer.ID)
	assert.ErrorIs(t, err, domain.ErrConflict)
	_, err = f.members.FindByProjectAndUser(ctx, f.project.ID, rescuer.ID)
	assert.ErrorIs(t, err, domain.ErrNotFound)

	// once its only admin is disabled it can be rescued
	require.NoError(t, f.db.Model(&domain.User{}).Where("id = ?", f.ada.ID).Update("is_active", false).Error)
	m, err := f.members.AddAdminIfNone(ctx, f.project.ID, rescuer.ID)
	require.NoError(t, err)
	assert.Equal(t, "admin", m.Role)
	// a second rescue now finds an active admin
	other := createUser(t, repository.NewUserRepository(f.db), "sys2@example.com", true, true)
	_, err = f.members.AddAdminIfNone(ctx, f.project.ID, other.ID)
	assert.ErrorIs(t, err, domain.ErrConflict)

	_, err = f.members.AddAdminIfNone(ctx, 99999, rescuer.ID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

// An existing lower membership is raised, not duplicated.
func TestMemberRepo_AddAdminIfNone_RaisesAnExistingMembership(t *testing.T) {
	f := newRolesRepoFixture(t, newTestDB(t))
	ctx := context.Background()
	require.NoError(t, f.db.Model(&domain.User{}).Where("id = ?", f.ada.ID).Update("is_active", false).Error)
	m, err := f.members.AddAdminIfNone(ctx, f.project.ID, f.ben.ID)
	require.NoError(t, err)
	assert.Equal(t, "admin", m.Role)
	all, err := f.members.FindByProjectID(ctx, f.project.ID)
	require.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestProjectRepo_CreateWithAdmin(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	ada := createUser(t, repository.NewUserRepository(db), "ada@example.com", false, true)
	projects := repository.NewProjectRepository(db)
	members := repository.NewProjectMemberRepository(db)

	p := &domain.Project{Key: "new", Name: "New", OwnerID: ada.ID}
	require.NoError(t, projects.CreateWithAdmin(ctx, p, ada.ID))
	require.NotZero(t, p.ID)
	m, err := members.FindByProjectAndUser(ctx, p.ID, ada.ID)
	require.NoError(t, err)
	assert.Equal(t, "admin", m.Role)

	// the second insert fails (duplicate key): the first project is untouched and no stray rows remain
	dup := &domain.Project{Key: "new", Name: "Dup", OwnerID: ada.ID}
	assert.Error(t, projects.CreateWithAdmin(ctx, dup, ada.ID))
	all, err := members.FindByUserID(ctx, ada.ID)
	require.NoError(t, err)
	assert.Len(t, all, 1)
}

func TestProjectRepo_FindWithoutActiveAdmin(t *testing.T) {
	f := newRolesRepoFixture(t, newTestDB(t))
	ctx := context.Background()
	projects := repository.NewProjectRepository(f.db)

	got, err := projects.FindWithoutActiveAdmin(ctx)
	require.NoError(t, err)
	assert.Empty(t, got, "demo has an active admin")

	require.NoError(t, f.db.Model(&domain.User{}).Where("id = ?", f.ada.ID).Update("is_active", false).Error)
	got, err = projects.FindWithoutActiveAdmin(ctx)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "demo", got[0].Key)

	// a project with no members at all is abandoned too
	empty := &domain.Project{Key: "empty", Name: "Empty", OwnerID: 99999}
	require.NoError(t, projects.Create(ctx, empty))
	got, err = projects.FindWithoutActiveAdmin(ctx)
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestProjectRepo_DeleteIfEmpty(t *testing.T) {
	db := newTestDB(t)
	require.NoError(t, db.AutoMigrate(&domain.Label{}, &domain.IssueLabel{}))
	f := newRolesRepoFixture(t, db)
	ctx := context.Background()
	projects := repository.NewProjectRepository(db)
	sprints := repository.NewSprintRepository(db)
	labels := repository.NewLabelRepository(db)
	issues := repository.NewIssueRepository(db)

	// a second project whose rows must survive
	other := &domain.Project{Key: "other", Name: "Other", OwnerID: f.ada.ID}
	require.NoError(t, projects.CreateWithAdmin(ctx, other, f.ada.ID))
	for _, pid := range []uint{f.project.ID, other.ID} {
		require.NoError(t, sprints.Create(ctx, &domain.Sprint{ProjectID: pid, Name: "S", Status: domain.SprintStatusPlanning}))
		l := &domain.Label{ProjectID: pid, Name: "bug", Color: "#e11d48"}
		require.NoError(t, labels.Create(ctx, l))
		require.NoError(t, labels.AttachToIssue(ctx, 1000+pid, l.ID))
	}

	// with a task in the project nothing is deleted
	task := &domain.Issue{ProjectID: f.project.ID, Key: "demo-1", Title: "t", Type: domain.IssueTypeTask, Status: domain.IssueStatusTodo, Priority: domain.IssuePriorityLow, ReporterID: f.ada.ID}
	require.NoError(t, issues.Create(ctx, task))
	err := projects.DeleteIfEmpty(ctx, f.project.ID)
	var notEmpty *domain.ProjectNotEmptyError
	require.ErrorAs(t, err, &notEmpty)
	assert.Equal(t, 1, notEmpty.TaskCount)
	_, err = projects.FindByID(ctx, f.project.ID)
	assert.NoError(t, err, "the project is untouched")
	members, _ := f.members.FindByProjectID(ctx, f.project.ID)
	assert.Len(t, members, 2)

	// once empty, everything that belongs to it goes, and nothing else
	require.NoError(t, issues.Delete(ctx, task.ID))
	require.NoError(t, projects.DeleteIfEmpty(ctx, f.project.ID))
	_, err = projects.FindByID(ctx, f.project.ID)
	assert.ErrorIs(t, err, domain.ErrNotFound)
	members, _ = f.members.FindByProjectID(ctx, f.project.ID)
	assert.Empty(t, members)
	s1, _ := sprints.FindByProject(ctx, f.project.ID)
	s2, _ := sprints.FindByProject(ctx, other.ID)
	assert.Empty(t, s1)
	assert.Len(t, s2, 1)
	l1, _ := labels.FindByProjectID(ctx, f.project.ID)
	l2, _ := labels.FindByProjectID(ctx, other.ID)
	assert.Empty(t, l1)
	assert.Len(t, l2, 1)
	gone, _ := labels.FindByIssueID(ctx, 1000+f.project.ID)
	assert.Empty(t, gone, "issue links go with the label")
	kept, _ := labels.FindByIssueID(ctx, 1000+other.ID)
	assert.Len(t, kept, 1)

	assert.ErrorIs(t, projects.DeleteIfEmpty(ctx, 99999), domain.ErrNotFound)
}
