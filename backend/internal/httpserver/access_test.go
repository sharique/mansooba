package httpserver_test

// Access matrix: every project- and issue-scoped route must refuse someone who
// is not a member of the project (403, or 404 where the resource itself is
// hidden) and must refuse an unauthenticated caller (401). It was added with
// the fix for GET /issues/:id/activity, which had no membership check, and
// guards against the same gap in any other route (feature 018, story 4).

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type accessEnv struct {
	e                                      *echo.Echo
	owner, member, outsider                string // bearer tokens
	issue1, issue2, comment, label, sprint string
	relation                               string
	memberID, outsiderID                   string
}

func call(e *echo.Echo, method, path, token, body string) *httptest.ResponseRecorder {
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	return do(e, r)
}

func jsonField(t *testing.T, rec *httptest.ResponseRecorder, path ...string) string {
	t.Helper()
	var v any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &v), rec.Body.String())
	for _, p := range path {
		m, ok := v.(map[string]any)
		require.True(t, ok, rec.Body.String())
		v = m[p]
	}
	switch x := v.(type) {
	case float64:
		return fmt.Sprintf("%d", int64(x))
	case string:
		return x
	}
	require.Fail(t, "field not found", rec.Body.String())
	return ""
}

func newAccessEnv(t *testing.T) *accessEnv {
	t.Helper()
	env := &accessEnv{e: newServer(t, nil)}
	e := env.e
	pw := "Passw0rd!x"

	rec := call(e, "POST", "/api/v1/setup/admin", "", `{"full_name":"Owner","email":"owner@example.com","password":"`+pw+`"}`)
	require.Equal(t, 201, rec.Code, rec.Body.String())
	env.owner = jsonField(t, rec, "access_token")

	for _, u := range []struct{ name, email string }{{"Member", "member@example.com"}, {"Outsider", "outsider@example.com"}} {
		rec = call(e, "POST", "/api/v1/auth/register", env.owner, `{"full_name":"`+u.name+`","email":"`+u.email+`","password":"`+pw+`"}`)
		require.Equal(t, 201, rec.Code, rec.Body.String())
		tok := jsonField(t, rec, "access_token")
		id := jsonField(t, rec, "user", "id")
		if u.name == "Member" {
			env.member, env.memberID = tok, id
		} else {
			env.outsider, env.outsiderID = tok, id
		}
	}

	rec = call(e, "POST", "/api/v1/projects", env.owner, `{"name":"Access","key":"ACC"}`)
	require.Equal(t, 201, rec.Code, rec.Body.String())
	rec = call(e, "POST", "/api/v1/projects/acc/members", env.owner, `{"email":"member@example.com","role":"member"}`)
	require.Equal(t, 201, rec.Code, rec.Body.String())

	mk := func(title string) string {
		rec := call(e, "POST", "/api/v1/projects/acc/issues", env.owner, `{"title":"`+title+`","type":"task","priority":"low"}`)
		require.Equal(t, 201, rec.Code, rec.Body.String())
		return jsonField(t, rec, "id")
	}
	env.issue1, env.issue2 = mk("one"), mk("two")

	rec = call(e, "POST", "/api/v1/issues/"+env.issue1+"/comments", env.owner, `{"body":"hello"}`)
	require.Equal(t, 201, rec.Code, rec.Body.String())
	env.comment = jsonField(t, rec, "id")
	rec = call(e, "POST", "/api/v1/projects/acc/labels", env.owner, `{"name":"bug","color":"#e11d48"}`)
	require.Equal(t, 201, rec.Code, rec.Body.String())
	env.label = jsonField(t, rec, "id")
	rec = call(e, "POST", "/api/v1/projects/acc/sprints", env.owner, `{"name":"S1"}`)
	require.Equal(t, 201, rec.Code, rec.Body.String())
	env.sprint = jsonField(t, rec, "id")
	rec = call(e, "POST", "/api/v1/issues/"+env.issue1+"/relations", env.owner, `{"target_issue_id":`+env.issue2+`,"relation_type":"blocks"}`)
	require.Equal(t, 201, rec.Code, rec.Body.String())
	env.relation = jsonField(t, rec, "id")
	return env
}

type route struct{ method, path, body string }

// routes lists every project- or issue-scoped endpoint. Read-only routes come
// first and destructive ones last so a missing check in one route cannot hide
// a missing check in another.
func (env *accessEnv) routes() []route {
	i1, i2, c, l, s, r := env.issue1, env.issue2, env.comment, env.label, env.sprint, env.relation
	return []route{
		{"GET", "/api/v1/projects/acc", ""},
		{"GET", "/api/v1/projects/acc/members", ""},
		{"GET", "/api/v1/projects/acc/labels", ""},
		{"GET", "/api/v1/projects/acc/issues", ""},
		{"GET", "/api/v1/projects/acc/issues/" + i1, ""},
		{"GET", "/api/v1/projects/acc/board", ""},
		{"GET", "/api/v1/projects/acc/backlog", ""},
		{"GET", "/api/v1/projects/acc/velocity", ""},
		{"GET", "/api/v1/projects/acc/sprints", ""},
		{"GET", "/api/v1/projects/acc/sprints/" + s, ""},
		{"GET", "/api/v1/projects/acc/sprints/" + s + "/issues", ""},
		{"GET", "/api/v1/projects/acc/sprints/" + s + "/burndown", ""},
		{"GET", "/api/v1/issues/" + i1 + "/comments", ""},
		{"GET", "/api/v1/issues/" + i1 + "/activity", ""},
		{"GET", "/api/v1/issues/" + i1 + "/labels", ""},
		{"GET", "/api/v1/issues/" + i1 + "/relations", ""},
		{"GET", "/api/v1/issues/" + i1 + "/attachments", ""},
		{"GET", "/api/v1/issues/" + i1 + "/attachments/1/download", ""},

		{"POST", "/api/v1/projects/acc/members", `{"email":"outsider@example.com","role":"member"}`},
		{"POST", "/api/v1/projects/acc/labels", `{"name":"x","color":"#3b82f6"}`},
		{"POST", "/api/v1/projects/acc/issues", `{"title":"x","type":"task","priority":"low"}`},
		{"POST", "/api/v1/projects/acc/sprints", `{"name":"x"}`},
		{"POST", "/api/v1/issues/" + i1 + "/comments", `{"body":"x"}`},
		{"POST", "/api/v1/issues/" + i1 + "/labels/" + l, ""},
		{"POST", "/api/v1/issues/" + i1 + "/relations", `{"target_issue_id":` + i2 + `,"relation_type":"relates_to"}`},

		{"PUT", "/api/v1/projects/acc", `{"name":"hijacked"}`},
		{"PUT", "/api/v1/projects/acc/issues/" + i1, `{"title":"hijacked"}`},
		{"PUT", "/api/v1/projects/acc/sprints/" + s, `{"goal":"hijacked"}`},
		{"POST", "/api/v1/projects/acc/sprints/" + s + "/start", ""},
		{"POST", "/api/v1/projects/acc/sprints/" + s + "/complete", `{}`},
		{"PUT", "/api/v1/issues/" + i1 + "/comments/" + c, `{"body":"hijacked"}`},

		{"DELETE", "/api/v1/issues/" + i1 + "/comments/" + c, ""},
		{"DELETE", "/api/v1/issues/" + i1 + "/labels/" + l, ""},
		{"DELETE", "/api/v1/issues/" + i1 + "/relations/" + r, ""},
		{"DELETE", "/api/v1/issues/" + i1 + "/attachments/1", ""},
		{"DELETE", "/api/v1/projects/acc/labels/" + l, ""},
		{"DELETE", "/api/v1/projects/acc/sprints/" + s, ""},
		{"DELETE", "/api/v1/projects/acc/members/" + env.memberID, ""},
		{"DELETE", "/api/v1/projects/acc/issues/" + i1, ""},
		{"DELETE", "/api/v1/projects/acc", ""},
	}
}

func TestAccess_NonMemberIsRefusedEverywhere(t *testing.T) {
	env := newAccessEnv(t)
	for _, rt := range env.routes() {
		rec := call(env.e, rt.method, rt.path, env.outsider, rt.body)
		ok := rec.Code == http.StatusForbidden || rec.Code == http.StatusNotFound
		assert.True(t, ok, "%s %s as a non-member returned %d (want 403 or 404): %s", rt.method, rt.path, rec.Code, rec.Body.String())
	}
}

func TestAccess_AnonymousIsRefusedEverywhere(t *testing.T) {
	env := newAccessEnv(t)
	for _, rt := range env.routes() {
		rec := call(env.e, rt.method, rt.path, "", rt.body)
		assert.Equal(t, http.StatusUnauthorized, rec.Code, "%s %s without a token: %s", rt.method, rt.path, rec.Body.String())
	}
}

// The project must still exist and be unchanged after the non-member's attempts.
func TestAccess_NonMemberAttemptsChangeNothing(t *testing.T) {
	env := newAccessEnv(t)
	for _, rt := range env.routes() {
		call(env.e, rt.method, rt.path, env.outsider, rt.body)
	}
	rec := call(env.e, "GET", "/api/v1/projects/acc", env.owner, "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Access", jsonField(t, rec, "name"))
	rec = call(env.e, "GET", "/api/v1/projects/acc/issues/"+env.issue1, env.owner, "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "one", jsonField(t, rec, "title"))
	rec = call(env.e, "GET", "/api/v1/projects/acc/members", env.owner, "")
	assert.NotContains(t, rec.Body.String(), "outsider@example.com")
}

func messageOf(t *testing.T, body string) string {
	t.Helper()
	var v struct {
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal([]byte(body), &v), body)
	return v.Message
}

func jsonString(t *testing.T, s string) string {
	t.Helper()
	b, err := json.Marshal(s)
	require.NoError(t, err)
	return string(b)
}
