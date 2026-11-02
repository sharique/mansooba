package httpserver_test

// The system administrator and project roles stay separate (US4): the flag
// governs users, settings and logs and nothing inside a project, there is always
// an active system administrator, and an abandoned project can be taken over by
// an explicit, visible step.

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func meID(t *testing.T, env *rolesEnv, token string) string {
	t.Helper()
	return jsonField(t, call(env.e, "GET", "/api/v1/auth/me", token, ""), "id")
}

// A project admin is not a system administrator.
func TestSystem_ProjectAdminCannotUseSystemPages(t *testing.T) {
	env := newRolesEnv(t)
	for _, rt := range []struct{ method, path, body string }{
		{"GET", "/api/v1/admin/users", ""},
		{"PATCH", "/api/v1/admin/users/" + env.benID, `{"is_active":false}`},
		{"GET", "/api/v1/admin/system-logs", ""},
		{"GET", "/api/v1/admin/usernames/check?username=zed", ""},
		{"PATCH", "/api/v1/settings", `{"demo_banner_enabled":true}`},
		{"POST", "/api/v1/auth/register", `{"full_name":"X","email":"x@example.com","password":"Passw0rd!x"}`},
		{"GET", "/api/v1/admin/projects/without-admin", ""},
		{"POST", "/api/v1/admin/projects/rol/claim-admin", ""},
	} {
		rec := call(env.e, rt.method, rt.path, env.ada, rt.body)
		assert.Equal(t, http.StatusForbidden, rec.Code, "%s %s as a project admin: %s", rt.method, rt.path, rec.Body.String())
	}
}

// ...and the system administrator still can.
func TestSystem_SystemAdministratorKeepsTheirOwnPages(t *testing.T) {
	env := newRolesEnv(t)
	assert.Equal(t, http.StatusOK, call(env.e, "GET", "/api/v1/admin/users", env.sys, "").Code)
	// the in-memory server has no Loki, so the logs page cannot succeed here; it must not be refused
	assert.NotEqual(t, http.StatusForbidden, call(env.e, "GET", "/api/v1/admin/system-logs", env.sys, "").Code)
	assert.Equal(t, http.StatusOK, call(env.e, "GET", "/api/v1/admin/projects/without-admin", env.sys, "").Code)
}

// FR-007c through the whole stack.
func TestSystem_TheLastSystemAdministratorCannotBeRemoved(t *testing.T) {
	env := newRolesEnv(t)
	sysID := meID(t, env, env.sys)
	adaID := meID(t, env, env.ada)

	for _, body := range []string{`{"is_admin":false}`, `{"is_active":false}`} {
		rec := call(env.e, "PATCH", "/api/v1/admin/users/"+sysID, env.sys, body)
		assert.Equal(t, http.StatusConflict, rec.Code, "%s: %s", body, rec.Body.String())
	}

	// with a second administrator the first may step down...
	require.Equal(t, http.StatusOK, call(env.e, "PATCH", "/api/v1/admin/users/"+adaID, env.sys, `{"is_admin":true}`).Code)
	require.Equal(t, http.StatusOK, call(env.e, "PATCH", "/api/v1/admin/users/"+sysID, env.sys, `{"is_admin":false}`).Code)
	// ...and the one left cannot follow
	rec := call(env.e, "PATCH", "/api/v1/admin/users/"+adaID, env.ada, `{"is_admin":false}`)
	assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	assert.Equal(t, http.StatusConflict, call(env.e, "PATCH", "/api/v1/admin/users/"+adaID, env.ada, `{"is_active":false}`).Code)
}

// FR-007d: an abandoned project is found, then taken over by an explicit step.
func TestSystem_AbandonedProjectCanBeTakenOverByTheSystemAdministrator(t *testing.T) {
	env := newRolesEnv(t)
	adaID := meID(t, env, env.ada)
	list := func() string {
		rec := call(env.e, "GET", "/api/v1/admin/projects/without-admin", env.sys, "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		return rec.Body.String()
	}
	assert.JSONEq(t, `[]`, list(), "a project with an active admin is not listed")
	assert.Equal(t, http.StatusConflict, call(env.e, "POST", "/api/v1/admin/projects/rol/claim-admin", env.sys, "").Code,
		"cannot take over a project that has an active admin")

	// the only admin's account is disabled
	require.Equal(t, http.StatusOK, call(env.e, "PATCH", "/api/v1/admin/users/"+adaID, env.sys, `{"is_active":false}`).Code)
	got := list()
	assert.Contains(t, got, `"key":"rol"`)
	assert.Contains(t, got, `"name":"Roles"`)
	assert.Contains(t, got, `"member_count":3`)
	assert.NotContains(t, got, "issue", "no contents are shown")
	assert.NotContains(t, got, "ben@example.com", "no member details are shown")

	// listing it does not open the project
	assert.Equal(t, http.StatusNotFound, call(env.e, "GET", "/api/v1/projects/rol", env.sys, "").Code)
	assert.Equal(t, http.StatusNotFound, call(env.e, "GET", "/api/v1/projects/rol/issues", env.sys, "").Code)

	// the explicit step
	rec := call(env.e, "POST", "/api/v1/admin/projects/rol/claim-admin", env.sys, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "admin", jsonField(t, rec, "role"))
	rec = call(env.e, "GET", "/api/v1/projects/rol", env.sys, "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "admin", jsonField(t, rec, "my_role"))
	assert.JSONEq(t, `[]`, list(), "no longer abandoned")
	assert.Equal(t, http.StatusConflict, call(env.e, "POST", "/api/v1/admin/projects/rol/claim-admin", env.sys, "").Code)

	// the people in the project were told
	rec = call(env.e, "GET", "/api/v1/notifications", env.ben, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), "project_admin_claimed")

	// unknown project
	assert.Equal(t, http.StatusNotFound, call(env.e, "POST", "/api/v1/admin/projects/nosuch/claim-admin", env.sys, "").Code)
}

// A system administrator who joins a project has exactly that member's rights, no more.
func TestSystem_AddedAsAMemberHasOnlyThatMembersRights(t *testing.T) {
	env := newRolesEnv(t)
	require.Equal(t, http.StatusCreated, call(env.e, "POST", "/api/v1/projects/rol/members", env.ada, `{"email":"sys@example.com","role":"viewer"}`).Code)
	assert.Equal(t, http.StatusOK, call(env.e, "GET", "/api/v1/projects/rol/issues", env.sys, "").Code)
	rec := call(env.e, "PUT", "/api/v1/projects/rol/issues/"+env.issue1, env.sys, `{"title":"x"}`)
	assert.Equal(t, http.StatusForbidden, rec.Code, "being a system administrator does not lift a viewer's limits: %s", rec.Body.String())
}
