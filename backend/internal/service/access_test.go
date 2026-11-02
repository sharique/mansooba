package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/sharique/mansooba/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Real SQLite, real repositories: access decisions are a security path
// (Constitution III), so no stubs here.
type accessFixture struct {
	db      *gorm.DB
	access  service.Access
	project *domain.Project
	admin   *domain.User
	member  *domain.User
	viewer  *domain.User
	outside *domain.User
	sysAdm  *domain.User
}

func newAccessFixture(t *testing.T) *accessFixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&domain.User{}, &domain.Project{}, &domain.ProjectMember{}))

	mk := func(name string, isAdmin bool) *domain.User {
		u := &domain.User{Name: name, Email: name + "@example.com", Username: name, Password: "x", IsAdmin: isAdmin, IsActive: true}
		require.NoError(t, db.Create(u).Error)
		return u
	}
	f := &accessFixture{db: db}
	f.admin, f.member, f.viewer, f.outside = mk("ada", false), mk("ben", false), mk("cy", false), mk("dee", false)
	f.sysAdm = mk("sys", true)
	f.project = &domain.Project{Key: "demo", Name: "Demo", OwnerID: f.admin.ID}
	require.NoError(t, db.Create(f.project).Error)
	for u, role := range map[*domain.User]string{f.admin: "admin", f.member: "member", f.viewer: "viewer"} {
		require.NoError(t, db.Create(&domain.ProjectMember{ProjectID: f.project.ID, UserID: u.ID, Role: role}).Error)
	}
	f.access = service.NewAccess(
		repository.NewProjectRepository(db),
		repository.NewProjectMemberRepository(db),
		repository.NewUserRepository(db),
	)
	return f
}

func TestAccess_RequireProject_AllowsAndDeniesByRole(t *testing.T) {
	f := newAccessFixture(t)
	ctx := context.Background()

	m, err := f.access.RequireProject(ctx, f.project.ID, f.member.ID, domain.CapTaskWrite)
	require.NoError(t, err)
	assert.Equal(t, "member", m.Role)

	_, err = f.access.RequireProject(ctx, f.project.ID, f.viewer.ID, domain.CapProjectView)
	assert.NoError(t, err)

	_, err = f.access.RequireProject(ctx, f.project.ID, f.viewer.ID, domain.CapTaskWrite)
	require.Error(t, err)
	assert.True(t, errors.Is(err, domain.ErrForbidden))
	var denied *domain.AccessDeniedError
	require.True(t, errors.As(err, &denied))
	assert.Equal(t, "Viewers can't edit tasks", denied.Reason())

	_, err = f.access.RequireProject(ctx, f.project.ID, f.member.ID, domain.CapSprintManage)
	assert.True(t, errors.Is(err, domain.ErrForbidden))

	_, err = f.access.RequireProject(ctx, f.project.ID, f.admin.ID, domain.CapProjectDelete)
	assert.NoError(t, err)
}

// FR-006: a project the caller cannot see and a project that does not exist
// look exactly alike.
func TestAccess_NonMemberAndMissingProjectGiveTheSameAnswer(t *testing.T) {
	f := newAccessFixture(t)
	ctx := context.Background()

	_, errHidden := f.access.RequireProject(ctx, f.project.ID, f.outside.ID, domain.CapProjectView)
	_, errMissing := f.access.RequireProject(ctx, 99999, f.outside.ID, domain.CapProjectView)
	assert.True(t, errors.Is(errHidden, domain.ErrNotFound))
	assert.Equal(t, errMissing, errHidden)

	_, _, errHiddenKey := f.access.RequireProjectByKey(ctx, "demo", f.outside.ID, domain.CapProjectView)
	_, _, errMissingKey := f.access.RequireProjectByKey(ctx, "nope", f.outside.ID, domain.CapProjectView)
	assert.True(t, errors.Is(errHiddenKey, domain.ErrNotFound))
	assert.Equal(t, errMissingKey, errHiddenKey)
}

func TestAccess_SystemAdministratorIsNotAProjectMember(t *testing.T) {
	f := newAccessFixture(t)
	_, err := f.access.RequireProject(context.Background(), f.project.ID, f.sysAdm.ID, domain.CapProjectView)
	assert.True(t, errors.Is(err, domain.ErrNotFound), "the system administrator flag must grant no project access")
}

func TestAccess_RequireProjectByKey_LowerCasesTheKey(t *testing.T) {
	f := newAccessFixture(t)
	p, m, err := f.access.RequireProjectByKey(context.Background(), "DEMO", f.member.ID, domain.CapProjectView)
	require.NoError(t, err)
	assert.Equal(t, f.project.ID, p.ID)
	assert.Equal(t, "member", m.Role)
}

func TestAccess_RoleChangeTakesEffectOnTheNextCall(t *testing.T) {
	f := newAccessFixture(t)
	ctx := context.Background()
	_, err := f.access.RequireProject(ctx, f.project.ID, f.member.ID, domain.CapTaskWrite)
	require.NoError(t, err)

	require.NoError(t, f.db.Model(&domain.ProjectMember{}).
		Where("project_id = ? AND user_id = ?", f.project.ID, f.member.ID).Update("role", "viewer").Error)

	_, err = f.access.RequireProject(ctx, f.project.ID, f.member.ID, domain.CapTaskWrite)
	assert.True(t, errors.Is(err, domain.ErrForbidden), "no caching of roles")
}

func TestAccess_UnknownRoleInTheDatabaseHasNoAccess(t *testing.T) {
	f := newAccessFixture(t)
	require.NoError(t, f.db.Model(&domain.ProjectMember{}).
		Where("project_id = ? AND user_id = ?", f.project.ID, f.member.ID).Update("role", "owner").Error)
	_, err := f.access.RequireProject(context.Background(), f.project.ID, f.member.ID, domain.CapProjectView)
	assert.True(t, errors.Is(err, domain.ErrForbidden))
}

func TestAccess_RequireSystem(t *testing.T) {
	f := newAccessFixture(t)
	ctx := context.Background()
	assert.NoError(t, f.access.RequireSystem(ctx, f.sysAdm.ID, domain.SysUsers))
	assert.True(t, errors.Is(f.access.RequireSystem(ctx, f.admin.ID, domain.SysUsers), domain.ErrForbidden),
		"a project admin is not a system administrator")

	require.NoError(t, f.db.Model(&domain.User{}).Where("id = ?", f.sysAdm.ID).Update("is_active", false).Error)
	assert.True(t, errors.Is(f.access.RequireSystem(ctx, f.sysAdm.ID, domain.SysUsers), domain.ErrForbidden),
		"a disabled system administrator has no rights")

	assert.True(t, errors.Is(f.access.RequireSystem(ctx, 99999, domain.SysUsers), domain.ErrForbidden))
}
