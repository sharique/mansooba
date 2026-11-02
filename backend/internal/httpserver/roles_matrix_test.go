package httpserver_test

// Role matrix (SC-001, SC-002, SC-010): every project and issue route is tried
// as a viewer, a member, an admin, a stranger, and a system administrator who
// is not in the project. What is allowed or refused is not written here: it is
// read from the one rule table in domain/access.go, so this test also proves the
// routes and the table agree.
//
//   - a role lacking the route's capability gets 403 with a reason;
//   - someone who is not a member (a stranger, or a system administrator who has
//     not joined) gets 404, the same answer as for a project that does not exist;
//   - a role holding the capability is never refused (anything but 401/403/404).

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type rolesEnv struct {
	e                                      *echo.Echo
	sys, ada, ben, cy, dee                 string // tokens: system admin, project admin, member, viewer, stranger
	benID, cyID, deeID                     string
	issue1, issue2, comment, label, sprint string
	relation                               string
}

func newRolesEnv(t *testing.T) *rolesEnv {
	t.Helper()
	env := &rolesEnv{e: newServer(t, nil)}
	e := env.e
	pw := "Passw0rd!x"
	rec := call(e, "POST", "/api/v1/setup/admin", "", `{"full_name":"Sys","email":"sys@example.com","password":"`+pw+`"}`)
	require.Equal(t, 201, rec.Code, rec.Body.String())
	env.sys = jsonField(t, rec, "access_token")
	reg := func(name string) (token, id string) {
		rec := call(e, "POST", "/api/v1/auth/register", env.sys, `{"full_name":"`+name+`","email":"`+strings.ToLower(name)+`@example.com","password":"`+pw+`"}`)
		require.Equal(t, 201, rec.Code, rec.Body.String())
		return jsonField(t, rec, "access_token"), jsonField(t, rec, "user", "id")
	}
	env.ada, _ = reg("Ada")
	env.ben, env.benID = reg("Ben")
	env.cy, env.cyID = reg("Cy")
	env.dee, env.deeID = reg("Dee")

	// Ada is an ordinary user (not a system administrator) who creates the project.
	rec = call(e, "POST", "/api/v1/projects", env.ada, `{"name":"Roles","key":"ROL"}`)
	require.Equal(t, 201, rec.Code, rec.Body.String())
	for _, m := range []struct{ email, role string }{{"ben@example.com", "member"}, {"cy@example.com", "viewer"}} {
		rec = call(e, "POST", "/api/v1/projects/rol/members", env.ada, `{"email":"`+m.email+`","role":"`+m.role+`"}`)
		require.Equal(t, 201, rec.Code, rec.Body.String())
	}
	mk := func(title string) string {
		rec := call(e, "POST", "/api/v1/projects/rol/issues", env.ada, `{"title":"`+title+`","type":"task","priority":"low"}`)
		require.Equal(t, 201, rec.Code, rec.Body.String())
		return jsonField(t, rec, "id")
	}
	env.issue1, env.issue2 = mk("one"), mk("two")
	rec = call(e, "POST", "/api/v1/issues/"+env.issue1+"/comments", env.ada, `{"body":"hello"}`)
	require.Equal(t, 201, rec.Code, rec.Body.String())
	env.comment = jsonField(t, rec, "id")
	rec = call(e, "POST", "/api/v1/projects/rol/labels", env.ada, `{"name":"bug","color":"#e11d48"}`)
	require.Equal(t, 201, rec.Code, rec.Body.String())
	env.label = jsonField(t, rec, "id")
	rec = call(e, "POST", "/api/v1/issues/"+env.issue1+"/labels/"+env.label, env.ada, "")
	require.Less(t, rec.Code, 300, rec.Body.String())
	rec = call(e, "POST", "/api/v1/projects/rol/sprints", env.ada, `{"name":"S1"}`)
	require.Equal(t, 201, rec.Code, rec.Body.String())
	env.sprint = jsonField(t, rec, "id")
	rec = call(e, "POST", "/api/v1/issues/"+env.issue1+"/relations", env.ada, `{"target_issue_id":`+env.issue2+`,"relation_type":"blocks"}`)
	require.Equal(t, 201, rec.Code, rec.Body.String())
	env.relation = jsonField(t, rec, "id")
	return env
}

func (env *rolesEnv) token(role string) string {
	switch role {
	case "admin":
		return env.ada
	case "member":
		return env.ben
	case "viewer":
		return env.cy
	case "stranger":
		return env.dee
	case "sysadmin":
		return env.sys
	}
	panic("unknown role " + role)
}

// roleRoute is one endpoint and the capability a caller who is NOT the author
// of the target needs for it. Authorship rules ("delete your own") are covered
// in the service tests.
type roleRoute struct {
	method, path, body string
	cap                domain.Capability
}

func (env *rolesEnv) routes() []roleRoute {
	i1, i2, c, l, s, r := env.issue1, env.issue2, env.comment, env.label, env.sprint, env.relation
	const view, tw = domain.CapProjectView, domain.CapTaskWrite
	p := "/api/v1/projects/rol"
	return []roleRoute{
		{"GET", p, "", view},
		{"GET", p + "/members", "", view},
		{"GET", p + "/labels", "", view},
		{"GET", p + "/issues", "", view},
		{"GET", p + "/issues/" + i1, "", view},
		{"GET", p + "/board", "", view},
		{"GET", p + "/backlog", "", view},
		{"GET", p + "/velocity", "", view},
		{"GET", p + "/sprints", "", view},
		{"GET", p + "/sprints/" + s, "", view},
		{"GET", p + "/sprints/" + s + "/issues", "", view},
		{"GET", p + "/sprints/" + s + "/burndown", "", view},
		{"GET", p + "/overview/sprint", "", view},
		{"GET", p + "/overview/tasks", "", view},
		{"GET", p + "/overview/activity", "", view},
		{"GET", "/api/v1/issues/" + i1 + "/comments", "", view},
		{"GET", "/api/v1/issues/" + i1 + "/activity", "", view},
		{"GET", "/api/v1/issues/" + i1 + "/labels", "", view},
		{"GET", "/api/v1/issues/" + i1 + "/relations", "", view},
		{"GET", "/api/v1/issues/" + i1 + "/attachments", "", view},

		{"POST", p + "/issues", `{"title":"x","type":"task","priority":"low"}`, tw},
		{"PUT", p + "/issues/" + i1, `{"title":"renamed"}`, tw},
		{"POST", "/api/v1/issues/" + i1 + "/comments", `{"body":"x"}`, domain.CapCommentWrite},
		{"POST", p + "/labels", `{"name":"x","color":"#3b82f6"}`, domain.CapLabelWrite},
		{"POST", "/api/v1/issues/" + i2 + "/labels/" + l, "", domain.CapLabelWrite},
		{"POST", "/api/v1/issues/" + i2 + "/relations", `{"target_issue_id":` + i1 + `,"relation_type":"relates_to"}`, tw},
		{"POST", p + "/sprints", `{"name":"x"}`, domain.CapSprintManage},
		{"PUT", p + "/sprints/" + s, `{"goal":"g"}`, domain.CapSprintManage},
		{"POST", p + "/sprints/" + s + "/start", "", domain.CapSprintManage},
		{"POST", p + "/sprints/" + s + "/complete", `{}`, domain.CapSprintManage},
		{"PUT", p, `{"name":"Renamed"}`, domain.CapProjectEdit},
		{"POST", p + "/members", `{"email":"dee@example.com","role":"member"}`, domain.CapMemberManage},
		{"PATCH", p + "/members/" + env.benID, `{"role":"member"}`, domain.CapMemberManage},

		{"DELETE", "/api/v1/issues/" + i1 + "/labels/" + l, "", domain.CapLabelWrite},
		{"DELETE", "/api/v1/issues/" + i1 + "/relations/" + r, "", tw},
		{"DELETE", "/api/v1/issues/" + i1 + "/comments/" + c, "", domain.CapCommentDeleteAny},
		{"DELETE", p + "/labels/" + l, "", domain.CapLabelDelete},
		{"DELETE", p + "/sprints/" + s, "", domain.CapSprintManage},
		{"DELETE", p + "/issues/" + i2, "", domain.CapTaskDeleteAny},
		{"DELETE", p + "/members/" + env.benID, "", domain.CapMemberManage},
		{"DELETE", p, `{"confirm_name":"Roles"}`, domain.CapProjectDelete},
	}
}

func rowName(role string, rt roleRoute) string {
	return fmt.Sprintf("%s %s %s", role, rt.method, rt.path)
}

// shortPath names ids by what they are, replacing whole path segments only.
func shortPath(env *rolesEnv, p string) string {
	names := map[string]string{env.issue1: "{i1}", env.issue2: "{i2}", env.comment: "{c}", env.label: "{l}", env.sprint: "{s}", env.relation: "{r}", env.benID: "{ben}"}
	segs := strings.Split(p, "/")
	for i, sg := range segs {
		// only the segments after "projects/rol" or an "issues"/"labels"... position hold ids
		if n, ok := names[sg]; ok && i > 0 && segs[i-1] != "v1" {
			segs[i] = n
		}
	}
	return strings.Join(segs, "/")
}

func checkRow(t *testing.T, env *rolesEnv, role string, rt roleRoute) {
	t.Helper()
	label := rowName(role, rt)
	rec := call(env.e, rt.method, rt.path, env.token(role), rt.body)
	switch role {
	case "stranger", "sysadmin":
		assert.Equal(t, http.StatusNotFound, rec.Code, "%s should look like a missing project: %s", label, rec.Body.String())
	default:
		if domain.RoleCan(role, rt.cap) {
			assert.NotContains(t, []int{401, 403, 404}, rec.Code, "%s must be allowed (%s): %s", label, rt.cap, rec.Body.String())
			assert.Less(t, rec.Code, 500, "%s must not fail on the server: %s", label, rec.Body.String())
		} else {
			assert.Equal(t, http.StatusForbidden, rec.Code, "%s must be refused (%s): %s", label, rt.cap, rec.Body.String())
			assert.NotEmpty(t, messageOf(t, rec.Body.String()), "%s refusal should say why", label)
		}
	}
}

func runRows(t *testing.T, env *rolesEnv, roles ...string) {
	for _, rt := range env.routes() {
		for _, role := range roles {
			name := shortPath(env, rowName(role, rt))
			t.Run(name, func(t *testing.T) { checkRow(t, env, role, rt) })
		}
	}
}

// Roles that are refused something change nothing, so they can share one server.
func TestRolesMatrix_Refusals(t *testing.T) {
	runRows(t, newRolesEnv(t), "viewer", "stranger", "sysadmin")
}

func TestRolesMatrix_MemberHasExactlyTheMemberCapabilities(t *testing.T) {
	runRows(t, newRolesEnv(t), "member")
}

func TestRolesMatrix_AdminHasEveryCapability(t *testing.T) {
	runRows(t, newRolesEnv(t), "admin")
}

// FR-006: a project that does not exist and a project the caller cannot see give
// the same status and the same body.
func TestRolesMatrix_HiddenProjectLooksLikeAMissingOne(t *testing.T) {
	env := newRolesEnv(t)
	for _, tok := range []string{env.dee, env.sys} {
		hidden := call(env.e, "GET", "/api/v1/projects/rol", tok, "")
		missing := call(env.e, "GET", "/api/v1/projects/nosuchproject", tok, "")
		assert.Equal(t, missing.Code, hidden.Code)
		assert.Equal(t, missing.Body.String(), hidden.Body.String())
	}
}

// SC-008: change one cell of the rule table and every path that uses that
// capability follows, with no other edit. Here members lose permission to edit
// tasks: the edit is refused, the project reports the change to the interface,
// and unrelated abilities are untouched.
func TestRolesMatrix_OneTableCellDecidesEveryPath(t *testing.T) {
	env := newRolesEnv(t)
	issue := "/api/v1/projects/rol/issues/" + env.issue1

	rec := call(env.e, "PUT", issue, env.ben, `{"title":"before"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	tb := domain.DefaultAccessTable()
	tb.Set(domain.RoleMember, domain.CapTaskWrite, false)
	t.Cleanup(domain.UseAccessTable(tb))

	rec = call(env.e, "PUT", issue, env.ben, `{"title":"after"}`)
	assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())
	rec = call(env.e, "POST", "/api/v1/projects/rol/issues", env.ben, `{"title":"x","type":"task","priority":"low"}`)
	assert.Equal(t, http.StatusForbidden, rec.Code, rec.Body.String())

	rec = call(env.e, "GET", "/api/v1/projects/rol", env.ben, "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.NotContains(t, rec.Body.String(), `"task.write"`, "the interface must be told the new rule")
	assert.Contains(t, rec.Body.String(), `"comment.write"`)

	rec = call(env.e, "POST", "/api/v1/issues/"+env.issue1+"/comments", env.ben, `{"body":"still allowed"}`)
	assert.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
}

// The project fetch tells each caller what they may do (the interface reads
// this instead of keeping its own copy of the rules).
func TestRolesMatrix_ProjectCarriesTheCallersRoleAndPermissions(t *testing.T) {
	env := newRolesEnv(t)
	for _, tc := range []struct{ role, want string }{{"admin", "admin"}, {"member", "member"}, {"viewer", "viewer"}} {
		rec := call(env.e, "GET", "/api/v1/projects/rol", env.token(tc.role), "")
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		assert.Equal(t, tc.want, jsonField(t, rec, "my_role"))
		for _, c := range domain.CapabilitiesOf(tc.want) {
			assert.Contains(t, rec.Body.String(), `"`+string(c)+`"`, "%s should be told about %s", tc.role, c)
		}
	}
	rec := call(env.e, "GET", "/api/v1/projects/rol", env.cy, "")
	assert.NotContains(t, rec.Body.String(), `"task.write"`)
}

// A resource id from another project, used inside this project's URL, must be
// refused even for someone who belongs to both projects (no crossing over).
func TestRolesMatrix_IdsFromAnotherProjectAreNotReachable(t *testing.T) {
	env := newRolesEnv(t)
	// Ada administers ROL and also creates OTH, so she can see both.
	rec := call(env.e, "POST", "/api/v1/projects", env.ada, `{"name":"Other","key":"OTH"}`)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	rec = call(env.e, "POST", "/api/v1/projects/oth/labels", env.ada, `{"name":"theirs","color":"#e11d48"}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	otherLabel := jsonField(t, rec, "id")
	rec = call(env.e, "POST", "/api/v1/projects/oth/sprints", env.ada, `{"name":"theirs"}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	otherSprint := jsonField(t, rec, "id")
	rec = call(env.e, "POST", "/api/v1/projects/oth/issues", env.ada, `{"title":"theirs","type":"task","priority":"low"}`)
	require.Equal(t, http.StatusCreated, rec.Code)
	otherIssue := jsonField(t, rec, "id")

	rec = call(env.e, "POST", "/api/v1/issues/"+env.issue1+"/labels/"+otherLabel, env.ada, "")
	assert.Equal(t, http.StatusNotFound, rec.Code, "another project's label on this project's task: %s", rec.Body.String())
	rec = call(env.e, "PUT", "/api/v1/projects/rol/issues/"+env.issue1, env.ada, `{"sprint_id":`+otherSprint+`}`)
	assert.Equal(t, http.StatusNotFound, rec.Code, "another project's sprint for this project's task: %s", rec.Body.String())
	rec = call(env.e, "GET", "/api/v1/projects/rol/issues/"+otherIssue, env.ada, "")
	assert.Equal(t, http.StatusNotFound, rec.Code, "another project's task under this project's key: %s", rec.Body.String())
	rec = call(env.e, "GET", "/api/v1/projects/rol/sprints/"+otherSprint, env.ada, "")
	assert.Equal(t, http.StatusNotFound, rec.Code, "another project's sprint under this project's key: %s", rec.Body.String())
	rec = call(env.e, "DELETE", "/api/v1/projects/rol/labels/"+otherLabel, env.ada, "")
	assert.Equal(t, http.StatusNotFound, rec.Code, "another project's label under this project's key: %s", rec.Body.String())

	// A relation across projects is refused for her (she sees both) and looks like
	// "no such task" to someone who sees only one.
	rec = call(env.e, "POST", "/api/v1/issues/"+env.issue1+"/relations", env.ada, `{"target_issue_id":`+otherIssue+`,"relation_type":"relates_to"}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	rec = call(env.e, "POST", "/api/v1/issues/"+env.issue1+"/relations", env.ben, `{"target_issue_id":`+otherIssue+`,"relation_type":"relates_to"}`)
	assert.Equal(t, http.StatusNotFound, rec.Code, "ben cannot see project OTH: %s", rec.Body.String())
}

// A task cannot be given to someone who is not in the project.
func TestRoles_AssigneeMustBeAMember(t *testing.T) {
	env := newRolesEnv(t)
	issue := "/api/v1/projects/rol/issues/" + env.issue1
	rec := call(env.e, "PUT", issue, env.ada, `{"assignee_id":`+env.deeID+`}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	rec = call(env.e, "PUT", issue, env.ada, `{"assignee_id":`+env.cyID+`}`)
	assert.Equal(t, http.StatusOK, rec.Code, "a viewer can be the assignee: %s", rec.Body.String())
}

// Someone removed from a project keeps none of its tasks in their own list.
func TestRoles_RemovedMemberLosesTheirAssignedTasksFromTheirList(t *testing.T) {
	env := newRolesEnv(t)
	require.Equal(t, http.StatusOK, call(env.e, "PUT", "/api/v1/projects/rol/issues/"+env.issue1, env.ada, `{"assignee_id":`+env.benID+`}`).Code)
	rec := call(env.e, "GET", "/api/v1/auth/me/issues", env.ben, "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"one"`)

	require.Equal(t, http.StatusNoContent, call(env.e, "DELETE", "/api/v1/projects/rol/members/"+env.benID, env.ada, "").Code)
	rec = call(env.e, "GET", "/api/v1/auth/me/issues", env.ben, "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.NotContains(t, rec.Body.String(), `"one"`, "the removed member must not keep reading the task")
}
