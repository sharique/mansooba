package service_test

// Role management on a project: changing roles, the last-admin rule, removing
// people, deleting an empty project, and the rescue of an abandoned one.

import (
	"context"
	"errors"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type rolesSvcEnv struct {
	svc      service.ProjectService
	members  *stubProjectMemberRepo
	projects *stubProjectRepo
	users    *stubUserRepo
	issues   *stubIssueRepoForProject
	notifier *recordingNotifier
	log      *recordingSystemLogService
}

// Users: 1 ada (creator, admin), 2 ben (member), 3 cy (viewer), 4 dee (outsider), 9 sys (system administrator).
func newRolesSvcEnv(t *testing.T) *rolesSvcEnv {
	t.Helper()
	ctx := context.Background()
	e := &rolesSvcEnv{
		members: newStubProjectMemberRepo(), users: newStubUserRepo(), issues: &stubIssueRepoForProject{},
		notifier: &recordingNotifier{}, log: &recordingSystemLogService{},
	}
	e.projects = linked(e.members)
	e.svc = service.NewProjectService(e.projects, e.members, e.users, e.issues, e.notifier).
		WithSystemLog(e.log)
	for _, u := range []domain.User{
		{ID: 1, Name: "Ada", Email: "ada@example.com", Password: "x", IsActive: true},
		{ID: 2, Name: "Ben", Email: "ben@example.com", Password: "x", IsActive: true},
		{ID: 3, Name: "Cy", Email: "cy@example.com", Password: "x", IsActive: true},
		{ID: 4, Name: "Dee", Email: "dee@example.com", Password: "x", IsActive: true},
		{ID: 9, Name: "Sys", Email: "sys@example.com", Password: "x", IsActive: true, IsAdmin: true},
	} {
		u := u
		require.NoError(t, e.users.Create(ctx, &u))
	}
	_, err := e.svc.Create(ctx, 1, dto.CreateProjectRequest{Name: "My Project"})
	require.NoError(t, err)
	require.NoError(t, e.svc.AddMember(ctx, "mypr", 1, dto.AddMemberRequest{Email: "ben@example.com", Role: "member"}))
	require.NoError(t, e.svc.AddMember(ctx, "mypr", 1, dto.AddMemberRequest{Email: "cy@example.com", Role: "viewer"}))
	e.notifier.memberships, e.log.entries = nil, nil // forget the setup
	return e
}

func (e *rolesSvcEnv) roleOf(userID uint) string {
	for _, m := range e.members.members {
		if m.UserID == userID {
			return m.Role
		}
	}
	return ""
}

func TestChangeMemberRole_AdminPromotesAndDemotes(t *testing.T) {
	e := newRolesSvcEnv(t)
	ctx := context.Background()
	got, err := e.svc.ChangeMemberRole(ctx, "mypr", 1, 2, "admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", got.Role)
	assert.Equal(t, "admin", e.roleOf(2))
	_, err = e.svc.ChangeMemberRole(ctx, "mypr", 1, 2, "viewer")
	require.NoError(t, err)
	assert.Equal(t, "viewer", e.roleOf(2))
}

func TestChangeMemberRole_OnlyAdminsMayChangeRoles(t *testing.T) {
	e := newRolesSvcEnv(t)
	ctx := context.Background()
	for _, caller := range []uint{2, 3} {
		_, err := e.svc.ChangeMemberRole(ctx, "mypr", caller, 3, "admin")
		assert.ErrorIs(t, err, domain.ErrForbidden, "caller %d", caller)
	}
	_, err := e.svc.ChangeMemberRole(ctx, "mypr", 4, 3, "admin")
	assert.ErrorIs(t, err, domain.ErrNotFound, "a stranger learns nothing")
	assert.Equal(t, "viewer", e.roleOf(3))
}

func TestChangeMemberRole_RefusesUnknownRolesAndNonMembers(t *testing.T) {
	e := newRolesSvcEnv(t)
	ctx := context.Background()
	_, err := e.svc.ChangeMemberRole(ctx, "mypr", 1, 2, "owner")
	assert.ErrorIs(t, err, domain.ErrInvalidInput)
	_, err = e.svc.ChangeMemberRole(ctx, "mypr", 1, 4, "member")
	assert.ErrorIs(t, err, domain.ErrNotFound, "dee is not a member")
}

// FR-004
func TestChangeMemberRole_NeverLeavesTheProjectWithoutAnAdmin(t *testing.T) {
	e := newRolesSvcEnv(t)
	ctx := context.Background()
	_, err := e.svc.ChangeMemberRole(ctx, "mypr", 1, 1, "member")
	assert.ErrorIs(t, err, domain.ErrLastProjectAdmin)
	assert.Equal(t, "admin", e.roleOf(1))

	// with a second admin the first may step down, by changing their own role
	_, err = e.svc.ChangeMemberRole(ctx, "mypr", 1, 2, "admin")
	require.NoError(t, err)
	_, err = e.svc.ChangeMemberRole(ctx, "mypr", 1, 1, "member")
	require.NoError(t, err)
	_, err = e.svc.ChangeMemberRole(ctx, "mypr", 2, 2, "viewer")
	assert.ErrorIs(t, err, domain.ErrLastProjectAdmin, "ben is now the last one")
}

// A disabled admin does not count as available.
func TestChangeMemberRole_DisabledAdminsDoNotCount(t *testing.T) {
	e := newRolesSvcEnv(t)
	ctx := context.Background()
	_, err := e.svc.ChangeMemberRole(ctx, "mypr", 1, 2, "admin")
	require.NoError(t, err)
	e.members.inactive = map[uint]bool{2: true}
	_, err = e.svc.ChangeMemberRole(ctx, "mypr", 1, 1, "member")
	assert.ErrorIs(t, err, domain.ErrLastProjectAdmin)
}

func TestChangeMemberRole_LogsAndNotifies(t *testing.T) {
	e := newRolesSvcEnv(t)
	ctx := context.Background()
	_, err := e.svc.ChangeMemberRole(ctx, "mypr", 1, 2, "admin")
	require.NoError(t, err)
	require.Len(t, e.log.entries, 1)
	entry := e.log.entries[0]
	assert.Equal(t, domain.SystemLogCategoryAdminAction, entry.Category)
	assert.Equal(t, "project_role_changed", entry.Action)
	assert.Equal(t, uint(1), entry.ActorID)
	assert.Equal(t, uint(2), entry.TargetID)
	assert.Contains(t, entry.Detail, "member -> admin")
	require.Len(t, e.notifier.roleChanges, 1)
	assert.Equal(t, uint(2), e.notifier.roleChanges[0].targetID)

	// changing your own role records the change but tells nobody; an unchanged role does neither
	e.notifier.roleChanges, e.log.entries = nil, nil
	_, err = e.svc.ChangeMemberRole(ctx, "mypr", 1, 2, "admin")
	require.NoError(t, err)
	assert.Empty(t, e.log.entries)
	assert.Empty(t, e.notifier.roleChanges)
}

func TestRemoveMember_RulesAndAudit(t *testing.T) {
	e := newRolesSvcEnv(t)
	ctx := context.Background()
	assert.ErrorIs(t, e.svc.RemoveMember(ctx, "mypr", 1, 1), domain.ErrSelfRemoval)
	assert.ErrorIs(t, e.svc.RemoveMember(ctx, "mypr", 2, 3), domain.ErrForbidden)

	require.NoError(t, e.svc.RemoveMember(ctx, "mypr", 1, 3))
	assert.Equal(t, "", e.roleOf(3))
	require.Len(t, e.log.entries, 1)
	assert.Equal(t, "project_member_removed", e.log.entries[0].Action)
	assert.Len(t, e.notifier.memberships, 1)

	// another admin may remove the first admin while they themselves stay
	_, err := e.svc.ChangeMemberRole(ctx, "mypr", 1, 2, "admin")
	require.NoError(t, err)
	require.NoError(t, e.svc.RemoveMember(ctx, "mypr", 2, 1))
	// ...and a sole admin can never be removed by anybody else: no one else is left to do it
	assert.ErrorIs(t, e.svc.RemoveMember(ctx, "mypr", 2, 2), domain.ErrSelfRemoval)
}

func TestAddMember_IsLogged(t *testing.T) {
	e := newRolesSvcEnv(t)
	require.NoError(t, e.svc.AddMember(context.Background(), "mypr", 1, dto.AddMemberRequest{Email: "dee@example.com", Role: "viewer"}))
	require.Len(t, e.log.entries, 1)
	assert.Equal(t, "project_member_added", e.log.entries[0].Action)
	assert.Contains(t, e.log.entries[0].Detail, "viewer")
}

// ── deleting a project (FR-005b) ─────────────────────────────────────────────

func TestDelete_RefusedWhileTasksRemain(t *testing.T) {
	e := newRolesSvcEnv(t)
	e.projects.tasks = 12
	err := e.svc.Delete(context.Background(), "mypr", 1, "My Project")
	var notEmpty *domain.ProjectNotEmptyError
	require.ErrorAs(t, err, &notEmpty)
	assert.Equal(t, 12, notEmpty.TaskCount)
	assert.NotEmpty(t, e.members.members, "nothing was removed")
}

func TestDelete_NeedsTheExactName(t *testing.T) {
	e := newRolesSvcEnv(t)
	for _, name := range []string{"", "my project", "My Proj"} {
		err := e.svc.Delete(context.Background(), "mypr", 1, name)
		assert.ErrorIs(t, err, domain.ErrInvalidInput, "%q", name)
	}
	_, err := e.svc.FindByKey(context.Background(), "mypr", 1)
	assert.NoError(t, err, "still there")
}

func TestDelete_RemovesTheEmptyProject(t *testing.T) {
	e := newRolesSvcEnv(t)
	ctx := context.Background()

	require.NoError(t, e.svc.Delete(ctx, "mypr", 1, "My Project"))
	_, err := e.projects.FindByKey(ctx, "mypr")
	assert.ErrorIs(t, err, domain.ErrNotFound)
	assert.Empty(t, e.members.members)
	require.Len(t, e.log.entries, 1)
	assert.Equal(t, "project_deleted", e.log.entries[0].Action)
}

func TestDelete_AnyAdminMayButNobodyElse(t *testing.T) {
	e := newRolesSvcEnv(t)
	ctx := context.Background()
	for _, caller := range []uint{2, 3} {
		assert.ErrorIs(t, e.svc.Delete(ctx, "mypr", caller, "My Project"), domain.ErrForbidden)
	}
	assert.ErrorIs(t, e.svc.Delete(ctx, "mypr", 4, "My Project"), domain.ErrNotFound)
	// a second admin who did not create the project may delete it: no owner right exists
	_, err := e.svc.ChangeMemberRole(ctx, "mypr", 1, 2, "admin")
	require.NoError(t, err)
	assert.NoError(t, e.svc.Delete(ctx, "mypr", 2, "My Project"))
}

// ── rescue of an abandoned project (FR-007d) ─────────────────────────────────

func TestAbandonedProjects_OnlySystemAdministratorsMayListThem(t *testing.T) {
	e := newRolesSvcEnv(t)
	ctx := context.Background()
	_, err := e.svc.ListWithoutAdmin(ctx, 1)
	assert.ErrorIs(t, err, domain.ErrForbidden, "a project admin is not a system administrator")

	p, _ := e.projects.FindByKey(ctx, "mypr")
	e.projects.abandoned = []*domain.Project{p}
	got, err := e.svc.ListWithoutAdmin(ctx, 9)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "mypr", got[0].Key)
	assert.Equal(t, "My Project", got[0].Name)
	assert.Equal(t, 3, got[0].MemberCount)
}

func TestClaimAdmin_OnlyWhenNoActiveAdminAndOnlyBySystemAdministrators(t *testing.T) {
	e := newRolesSvcEnv(t)
	ctx := context.Background()

	_, err := e.svc.ClaimAdmin(ctx, "mypr", 1)
	assert.ErrorIs(t, err, domain.ErrForbidden)
	_, err = e.svc.ClaimAdmin(ctx, "mypr", 9)
	assert.ErrorIs(t, err, domain.ErrConflict, "the project still has an active admin")
	_, err = e.svc.FindByKey(ctx, "mypr", 9)
	assert.ErrorIs(t, err, domain.ErrNotFound, "no access without the explicit step")

	e.members.inactive = map[uint]bool{1: true}
	m, err := e.svc.ClaimAdmin(ctx, "MYPR", 9)
	require.NoError(t, err)
	assert.Equal(t, "admin", m.Role)
	assert.Equal(t, "admin", e.roleOf(9))
	_, err = e.svc.FindByKey(ctx, "mypr", 9)
	assert.NoError(t, err, "now a member")

	require.Len(t, e.log.entries, 1)
	assert.Equal(t, "project_admin_claimed", e.log.entries[0].Action)
	require.Len(t, e.notifier.claims, 1)
	assert.Equal(t, uint(9), e.notifier.claims[0].actorID)

	_, err = e.svc.ClaimAdmin(ctx, "nosuch", 9)
	assert.True(t, errors.Is(err, domain.ErrNotFound))
}

func TestAddMember_RefusesADisabledAccountAndADuplicate(t *testing.T) {
	e := newRolesSvcEnv(t)
	ctx := context.Background()
	u, err := e.users.FindByID(ctx, 4)
	require.NoError(t, err)
	u.IsActive = false
	assert.ErrorIs(t, e.svc.AddMember(ctx, "mypr", 1, dto.AddMemberRequest{Email: "dee@example.com", Role: "member"}), domain.ErrInvalidInput)

	err = e.svc.AddMember(ctx, "mypr", 1, dto.AddMemberRequest{Email: "ben@example.com", Role: "member"})
	assert.ErrorIs(t, err, domain.ErrAlreadyMember)
	assert.ErrorIs(t, err, domain.ErrConflict, "still a 409")
}
