package httpserver_test

// Request validation (feature 018, story 4): endpoints whose request types
// carry validation rules must refuse bad bodies with 400 and a message that
// names the field, and must create or change nothing. Before the fix these
// handlers never called Validate: empty bodies created empty projects, issues,
// sprints and comments, and some invalid values reached the service layer and
// came back as 500.

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidation_BadBodiesAreRefusedWith400(t *testing.T) {
	env := newAccessEnv(t)
	i1, s, c := env.issue1, env.sprint, env.comment

	cases := []struct {
		name, method, path, body, wantMsg string
	}{
		{"register: empty", "POST", "/api/v1/auth/register", `{}`, "full_name is required"},
		{"register: bad email", "POST", "/api/v1/auth/register", `{"full_name":"X","email":"nope","password":"Passw0rd!x"}`, "email must be a valid email address"},
		{"register: weak password", "POST", "/api/v1/auth/register", `{"full_name":"X","email":"x@example.com","password":"weak"}`, "password"},
		{"project: empty", "POST", "/api/v1/projects", `{}`, "name is required"},
		{"member: bad email and role", "POST", "/api/v1/projects/acc/members", `{"email":"x","role":"boss"}`, "email must be a valid email address"},
		{"member: bad role", "POST", "/api/v1/projects/acc/members", `{"email":"outsider@example.com","role":"boss"}`, "role must be one of"},
		{"label: empty", "POST", "/api/v1/projects/acc/labels", `{}`, "name is required"},
		{"label: colour outside the palette", "POST", "/api/v1/projects/acc/labels", `{"name":"x","color":"#123456"}`, "color"},
		{"issue: empty", "POST", "/api/v1/projects/acc/issues", `{}`, "title is required"},
		{"issue: bad type", "POST", "/api/v1/projects/acc/issues", `{"title":"x","type":"nope","priority":"low"}`, "type must be one of"},
		{"issue: bad priority", "POST", "/api/v1/projects/acc/issues", `{"title":"x","type":"task","priority":"urgent"}`, "priority must be one of"},
		{"issue update: bad status", "PUT", "/api/v1/projects/acc/issues/" + i1, `{"status":"nope"}`, "status"},
		{"issue update: blank title", "PUT", "/api/v1/projects/acc/issues/" + i1, `{"title":"  "}`, "title must not be empty"},
		{"issue update: bad type", "PUT", "/api/v1/projects/acc/issues/" + i1, `{"type":"nope"}`, "type must be one of"},
		{"issue update: bad priority", "PUT", "/api/v1/projects/acc/issues/" + i1, `{"priority":"nope"}`, "priority must be one of"},
		{"sprint: empty", "POST", "/api/v1/projects/acc/sprints", `{}`, "name is required"},
		{"sprint update: empty name", "PUT", "/api/v1/projects/acc/sprints/" + s, `{"name":""}`, "name must be at least 1 characters"},
		{"sprint update: blank name", "PUT", "/api/v1/projects/acc/sprints/" + s, `{"name":"   "}`, "name must not be empty"},
		{"comment: empty", "POST", "/api/v1/issues/" + i1 + "/comments", `{}`, "body is required"},
		{"comment update: empty", "PUT", "/api/v1/issues/" + i1 + "/comments/" + c, `{"body":""}`, "body is required"},
	}
	for _, tc := range cases {
		rec := call(env.e, tc.method, tc.path, env.owner, tc.body)
		assert.Equal(t, http.StatusBadRequest, rec.Code, "%s: %s", tc.name, rec.Body.String())
		assert.Contains(t, rec.Body.String(), tc.wantMsg, tc.name)
		assert.JSONEq(t, `{"code":"Bad Request","message":`+jsonString(t, messageOf(t, rec.Body.String()))+`}`, rec.Body.String(), tc.name)
	}

	// Nothing was created or changed by the refused requests.
	rec := call(env.e, "GET", "/api/v1/projects", env.owner, "")
	assert.Equal(t, 1, strings.Count(rec.Body.String(), `"key":`), "no extra project: %s", rec.Body.String())
	rec = call(env.e, "GET", "/api/v1/projects/acc/issues", env.owner, "")
	assert.Equal(t, 2, strings.Count(rec.Body.String(), `"key":`), "no extra issue: %s", rec.Body.String())
	rec = call(env.e, "GET", "/api/v1/projects/acc/sprints", env.owner, "")
	assert.Equal(t, 1, strings.Count(rec.Body.String(), `"name":`), "no extra sprint: %s", rec.Body.String())
	rec = call(env.e, "GET", "/api/v1/issues/"+i1+"/comments", env.owner, "")
	assert.Equal(t, 1, strings.Count(rec.Body.String(), `"body":`), "no extra comment: %s", rec.Body.String())
	rec = call(env.e, "GET", "/api/v1/projects/acc/members", env.owner, "")
	assert.NotContains(t, rec.Body.String(), "outsider@example.com")
	rec = call(env.e, "GET", "/api/v1/admin/users", env.owner, "")
	assert.Equal(t, 3, strings.Count(rec.Body.String(), `"email":`), "no extra account: %s", rec.Body.String())
}

func TestValidation_ValidBodiesStillWork(t *testing.T) {
	env := newAccessEnv(t)
	cases := []struct {
		method, path, body string
		want               int
	}{
		{"POST", "/api/v1/auth/register", `{"full_name":"New User","email":"new@example.com","password":"Passw0rd!x"}`, 201},
		{"POST", "/api/v1/projects", `{"name":"Second","key":"SEC","description":"d"}`, 201},
		{"POST", "/api/v1/projects/acc/members", `{"email":"outsider@example.com","role":"viewer"}`, 201},
		{"POST", "/api/v1/projects/acc/labels", `{"name":"feature","color":"#3b82f6"}`, 201},
		{"POST", "/api/v1/projects/acc/issues", `{"title":"ok","type":"story","priority":"high","story_points":3}`, 201},
		{"PUT", "/api/v1/projects/acc/issues/" + env.issue1, `{"status":"in_progress","priority":"critical"}`, 200},
		{"POST", "/api/v1/projects/acc/sprints", `{"name":"Sprint 2","goal":"g"}`, 201},
		{"PUT", "/api/v1/projects/acc/sprints/" + env.sprint, `{"goal":"changed"}`, 200},
		{"POST", "/api/v1/issues/" + env.issue1 + "/comments", `{"body":"fine"}`, 201},
		{"PUT", "/api/v1/issues/" + env.issue1 + "/comments/" + env.comment, `{"body":"edited"}`, 200},
	}
	for _, tc := range cases {
		rec := call(env.e, tc.method, tc.path, env.owner, tc.body)
		assert.Equal(t, tc.want, rec.Code, "%s %s: %s", tc.method, tc.path, rec.Body.String())
	}
}

func TestLastAdminConflictExplainsItself(t *testing.T) {
	env := newAccessEnv(t)
	ownerID := "1"
	for _, body := range []string{`{"is_admin":false}`, `{"is_active":false}`} {
		rec := call(env.e, "PATCH", "/api/v1/admin/users/"+ownerID, env.owner, body)
		require.Equal(t, http.StatusConflict, rec.Code, rec.Body.String())
		// The frontend recognises this sentence (pages/system/users.vue).
		assert.Contains(t, messageOf(t, rec.Body.String()), "last active admin")
		assert.NotContains(t, rec.Body.String(), "internal server error")
	}
}
