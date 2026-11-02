package domain_test

import (
	"errors"
	"sort"
	"testing"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The expected table, written out independently of the implementation. It is
// the table in specs/021-role-access-boundaries/data-model.md.
var expectedProject = map[domain.Capability]struct{ viewer, member, admin bool }{
	domain.CapProjectView:         {true, true, true},
	domain.CapTaskWrite:           {false, true, true},
	domain.CapTaskDeleteAny:       {false, false, true},
	domain.CapCommentWrite:        {false, true, true},
	domain.CapCommentDeleteAny:    {false, false, true},
	domain.CapLabelWrite:          {false, true, true},
	domain.CapLabelDelete:         {false, false, true},
	domain.CapAttachmentWrite:     {false, true, true},
	domain.CapAttachmentDeleteAny: {false, false, true},
	domain.CapSprintManage:        {false, false, true},
	domain.CapProjectEdit:         {false, false, true},
	domain.CapMemberManage:        {false, false, true},
	domain.CapProjectDelete:       {false, false, true},
}

func TestRoleCan_MatchesTheDocumentedTable(t *testing.T) {
	require.Len(t, expectedProject, 13)
	for c, want := range expectedProject {
		assert.Equal(t, want.viewer, domain.RoleCan(domain.RoleViewer, c), "viewer %s", c)
		assert.Equal(t, want.member, domain.RoleCan(domain.RoleMember, c), "member %s", c)
		assert.Equal(t, want.admin, domain.RoleCan(domain.RoleAdmin, c), "admin %s", c)
	}
}

func TestRoleCan_UnknownRoleHasNothing(t *testing.T) {
	for _, role := range []string{"", "owner", "Admin", "root", "superuser"} {
		for c := range expectedProject {
			assert.False(t, domain.RoleCan(role, c), "role %q capability %s", role, c)
		}
	}
}

func TestRoleCan_UnknownCapabilityIsDenied(t *testing.T) {
	assert.False(t, domain.RoleCan(domain.RoleAdmin, domain.Capability("nope")))
}

func TestIsValidRole(t *testing.T) {
	for _, r := range []string{"admin", "member", "viewer"} {
		assert.True(t, domain.IsValidRole(r), r)
	}
	for _, r := range []string{"", "owner", "ADMIN", "guest"} {
		assert.False(t, domain.IsValidRole(r), r)
	}
	assert.Equal(t, []string{"admin", "member", "viewer"}, domain.Roles())
}

func TestCapabilitiesOf_IsSortedAndMatchesRoleCan(t *testing.T) {
	for _, role := range domain.Roles() {
		caps := domain.CapabilitiesOf(role)
		names := make([]string, len(caps))
		for i, c := range caps {
			names[i] = string(c)
			assert.True(t, domain.RoleCan(role, c), "%s lists %s but RoleCan denies it", role, c)
		}
		assert.True(t, sort.StringsAreSorted(names), "%s capabilities are not sorted: %v", role, names)
		n := 0
		for c := range expectedProject {
			if domain.RoleCan(role, c) {
				n++
			}
		}
		assert.Len(t, caps, n, role)
	}
	assert.Empty(t, domain.CapabilitiesOf("owner"))
}

func TestSystemAdminCan(t *testing.T) {
	for _, c := range []domain.SystemCapability{
		domain.SysUsers, domain.SysSettings, domain.SysLogs, domain.SysRescueProject,
	} {
		assert.True(t, domain.SystemAdminCan(true, true, c), string(c))
		assert.False(t, domain.SystemAdminCan(false, true, c), "ordinary user %s", c)
		assert.False(t, domain.SystemAdminCan(true, false, c), "disabled admin %s", c)
	}
	assert.False(t, domain.SystemAdminCan(true, true, domain.SystemCapability("nope")))
}

// SC-008: one cell of the table decides the answer, nothing else does.
func TestAccessTable_FlippingOneCellChangesOnlyThatAnswer(t *testing.T) {
	tb := domain.DefaultAccessTable()
	require.True(t, tb.Can(domain.RoleMember, domain.CapTaskWrite))
	tb.Set(domain.RoleMember, domain.CapTaskWrite, false)
	assert.False(t, tb.Can(domain.RoleMember, domain.CapTaskWrite))
	assert.True(t, tb.Can(domain.RoleMember, domain.CapCommentWrite))
	assert.True(t, tb.Can(domain.RoleAdmin, domain.CapTaskWrite))
	// the shared table is untouched until a test installs the copy
	assert.True(t, domain.RoleCan(domain.RoleMember, domain.CapTaskWrite))

	restore := domain.UseAccessTable(tb)
	assert.False(t, domain.RoleCan(domain.RoleMember, domain.CapTaskWrite))
	assert.NotContains(t, domain.CapabilitiesOf(domain.RoleMember), domain.CapTaskWrite)
	restore()
	assert.True(t, domain.RoleCan(domain.RoleMember, domain.CapTaskWrite))
}

func TestAccessDeniedError(t *testing.T) {
	err := error(&domain.AccessDeniedError{Capability: domain.CapTaskWrite, Role: domain.RoleViewer})
	assert.True(t, errors.Is(err, domain.ErrForbidden))
	var ade *domain.AccessDeniedError
	require.True(t, errors.As(err, &ade))
	assert.Equal(t, "Viewers can't edit tasks", ade.Reason())

	for c := range expectedProject {
		e := &domain.AccessDeniedError{Capability: c, Role: domain.RoleMember}
		assert.NotEmpty(t, e.Reason(), string(c))
		assert.NotContains(t, e.Reason(), "project.", "reason must be plain language: %s", e.Reason())
	}
	assert.Equal(t, "Members can't manage sprints", (&domain.AccessDeniedError{Capability: domain.CapSprintManage, Role: domain.RoleMember}).Reason())
}

func TestErrProjectNotEmptyCarriesTheCount(t *testing.T) {
	err := error(&domain.ProjectNotEmptyError{TaskCount: 12})
	assert.True(t, errors.Is(err, domain.ErrProjectNotEmpty))
	var pe *domain.ProjectNotEmptyError
	require.True(t, errors.As(err, &pe))
	assert.Equal(t, 12, pe.TaskCount)
}
