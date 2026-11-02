package httpserver_test

// Behaviour that must not change while project roles are tightened: these pass
// before the role work starts and must keep passing at every step.

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRolesBaseline_CreatorIsListedAsAdmin(t *testing.T) {
	env := newAccessEnv(t)
	rec := call(env.e, "GET", "/api/v1/projects/acc/members", env.owner, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"owner@example.com"`)
	assert.Contains(t, rec.Body.String(), `"role":"admin"`)
}

func TestRolesBaseline_MemberCanReadTheProject(t *testing.T) {
	env := newAccessEnv(t)
	for _, p := range []string{
		"/api/v1/projects/acc",
		"/api/v1/projects/acc/issues",
		"/api/v1/projects/acc/board",
		"/api/v1/projects/acc/members",
	} {
		rec := call(env.e, "GET", p, env.member, "")
		assert.Equal(t, http.StatusOK, rec.Code, "%s: %s", p, rec.Body.String())
	}
}

func TestRolesBaseline_AdminCanAddAndRemoveMembers(t *testing.T) {
	env := newAccessEnv(t)
	rec := call(env.e, "POST", "/api/v1/projects/acc/members", env.owner, `{"email":"outsider@example.com","role":"viewer"}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	rec = call(env.e, "DELETE", "/api/v1/projects/acc/members/"+env.outsiderID, env.owner, "")
	assert.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
}

func TestRolesBaseline_ReporterCanDeleteOwnTask(t *testing.T) {
	env := newAccessEnv(t)
	rec := call(env.e, "POST", "/api/v1/projects/acc/issues", env.member, `{"title":"mine","type":"task","priority":"low"}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	id := jsonField(t, rec, "id")
	rec = call(env.e, "DELETE", "/api/v1/projects/acc/issues/"+id, env.member, "")
	assert.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
}

func TestRolesBaseline_MemberCannotManageTheProject(t *testing.T) {
	env := newAccessEnv(t)
	rec := call(env.e, "PUT", "/api/v1/projects/acc", env.member, `{"name":"nope"}`)
	assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	rec = call(env.e, "POST", "/api/v1/projects/acc/sprints", env.member, `{"name":"nope"}`)
	assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
}

// US2: any signed-in user (not only a system administrator) can create a
// project and is its admin from the first moment.
func TestRoles_AnyUserCreatesAProjectAndIsItsAdmin(t *testing.T) {
	env := newRolesEnv(t) // Ada is an ordinary user and created project ROL
	rec := call(env.e, "POST", "/api/v1/projects", env.ben, `{"name":"Bens","key":"BEN"}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	assert.Equal(t, "admin", jsonField(t, rec, "my_role"))

	rec = call(env.e, "GET", "/api/v1/projects/ben/members", env.ben, "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"role":"admin"`)

	// every admin action works straight away
	assert.Equal(t, http.StatusCreated, call(env.e, "POST", "/api/v1/projects/ben/members", env.ben, `{"email":"cy@example.com","role":"viewer"}`).Code)
	assert.Equal(t, http.StatusCreated, call(env.e, "POST", "/api/v1/projects/ben/sprints", env.ben, `{"name":"S1"}`).Code)
	assert.Equal(t, http.StatusOK, call(env.e, "PUT", "/api/v1/projects/ben", env.ben, `{"description":"mine"}`).Code)
}

// Being the creator gives nothing extra once another admin has changed the role.
func TestRoles_DemotedCreatorLosesAdminPowers(t *testing.T) {
	env := newRolesEnv(t)
	adaID := jsonField(t, call(env.e, "GET", "/api/v1/auth/me", env.ada, ""), "id")
	// Ben becomes a second admin, then demotes the creator.
	require.Equal(t, http.StatusOK, call(env.e, "PATCH", "/api/v1/projects/rol/members/"+env.benID, env.ada, `{"role":"admin"}`).Code)
	require.Equal(t, http.StatusOK, call(env.e, "PATCH", "/api/v1/projects/rol/members/"+adaID, env.ben, `{"role":"member"}`).Code)

	assert.Equal(t, http.StatusForbidden, call(env.e, "PUT", "/api/v1/projects/rol", env.ada, `{"name":"Mine"}`).Code)
	assert.Equal(t, http.StatusForbidden, call(env.e, "POST", "/api/v1/projects/rol/sprints", env.ada, `{"name":"x"}`).Code)
	// and the project cannot be left without an admin
	rec := call(env.e, "PATCH", "/api/v1/projects/rol/members/"+env.benID, env.ben, `{"role":"member"}`)
	assert.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
}

// FR-011: a role change applies on the person's very next request, with no new sign-in.
func TestRoles_ChangeTakesEffectOnTheNextRequest(t *testing.T) {
	env := newRolesEnv(t)
	issue := "/api/v1/projects/rol/issues/" + env.issue1
	assert.Equal(t, http.StatusOK, call(env.e, "PUT", issue, env.ben, `{"title":"a"}`).Code)
	require.Equal(t, http.StatusOK, call(env.e, "PATCH", "/api/v1/projects/rol/members/"+env.benID, env.ada, `{"role":"viewer"}`).Code)
	assert.Equal(t, http.StatusForbidden, call(env.e, "PUT", issue, env.ben, `{"title":"b"}`).Code)
	require.Equal(t, http.StatusOK, call(env.e, "PATCH", "/api/v1/projects/rol/members/"+env.benID, env.ada, `{"role":"member"}`).Code)
	assert.Equal(t, http.StatusOK, call(env.e, "PUT", issue, env.ben, `{"title":"c"}`).Code)
}

// FR-005b through the whole stack: tasks block deletion, the name must match.
func TestRoles_DeletingAProjectNeedsItEmptyAndTheNameTyped(t *testing.T) {
	env := newRolesEnv(t)
	p := "/api/v1/projects/rol"
	rec := call(env.e, "DELETE", p, env.ada, `{"confirm_name":"Roles"}`)
	require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
	assert.Contains(t, rec.Body.String(), `"task_count":2`)

	for _, id := range []string{env.issue1, env.issue2} {
		require.Equal(t, http.StatusNoContent, call(env.e, "DELETE", p+"/issues/"+id, env.ada, "").Code)
	}
	assert.Equal(t, http.StatusBadRequest, call(env.e, "DELETE", p, env.ada, `{"confirm_name":"roles"}`).Code)
	assert.Equal(t, http.StatusBadRequest, call(env.e, "DELETE", p, env.ada, ``).Code)
	assert.Equal(t, http.StatusNoContent, call(env.e, "DELETE", p, env.ada, `{"confirm_name":"Roles"}`).Code)
	assert.Equal(t, http.StatusNotFound, call(env.e, "GET", p, env.ada, "").Code)
}

// FR-010a through the whole stack.
func TestRoles_AnAdminCannotRemoveThemselves(t *testing.T) {
	env := newRolesEnv(t)
	adaID := jsonField(t, call(env.e, "GET", "/api/v1/auth/me", env.ada, ""), "id")
	rec := call(env.e, "DELETE", "/api/v1/projects/rol/members/"+adaID, env.ada, "")
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "Ask another admin to remove you")
}
