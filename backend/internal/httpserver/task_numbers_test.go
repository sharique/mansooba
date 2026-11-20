package httpserver_test

// The reproduced failure, end to end: delete a task in the middle of a project
// and create another, delete the newest and create another. A task number is
// issued once and never again.

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

func apiCall(t *testing.T, e *echo.Echo, method, path, token, body string) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	var out map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func TestTaskNumbers_NeverReusedAndCreateNeverFailsAfterADelete(t *testing.T) {
	e := newServer(t, nil)
	code, admin := apiCall(t, e, http.MethodPost, "/api/v1/setup/admin", "", `{"full_name":"Num Admin","email":"num@example.com","password":"Passw0rd!Long1","username":"numadmin"}`)
	require.Equal(t, http.StatusCreated, code)
	token := admin["access_token"].(string)
	code, _ = apiCall(t, e, http.MethodPost, "/api/v1/projects", token, `{"name":"Numbers","key":"NUM","description":"x"}`)
	require.Equal(t, http.StatusCreated, code)

	create := func(title string) (int, string, float64) {
		code, body := apiCall(t, e, http.MethodPost, "/api/v1/projects/num/issues", token, fmt.Sprintf(`{"title":%q,"type":"task","priority":"low"}`, title))
		if code != http.StatusCreated {
			return code, "", 0
		}
		assert.NotContains(t, body, "task_counter", "the counter is not part of the API")
		return code, body["key"].(string), body["id"].(float64)
	}
	remove := func(id float64) {
		code, _ := apiCall(t, e, http.MethodDelete, fmt.Sprintf("/api/v1/projects/num/issues/%d", int(id)), token, "")
		require.Equal(t, http.StatusNoContent, code)
	}

	_, k1, _ := create("one")
	_, k2, id2 := create("two")
	_, k3, id3 := create("three")
	assert.Equal(t, []string{"num-1", "num-2", "num-3"}, []string{k1, k2, k3})

	// Scenario A: delete a task in the middle. This used to answer 500.
	remove(id2)
	code, k4, id4 := create("four")
	assert.Equal(t, http.StatusCreated, code, "creating after deleting a middle task must not fail")
	assert.Equal(t, "num-4", k4)

	// Scenario B: delete the newest. The next task must not take its key.
	remove(id4)
	remove(id3)
	code, k5, _ := create("five")
	assert.Equal(t, http.StatusCreated, code)
	assert.Equal(t, "num-5", k5, "numbers 2, 3 and 4 belonged to deleted tasks and must not come back")

	// A task created into a sprint, as the backlog's Create Issue dialog does, is numbered the same way.
	code, sprint := apiCall(t, e, http.MethodPost, "/api/v1/projects/num/sprints", token, `{"name":"Sprint 1","goal":"g"}`)
	require.Equal(t, http.StatusCreated, code, "%v", sprint)
	code, withSprint := apiCall(t, e, http.MethodPost, "/api/v1/projects/num/issues", token, fmt.Sprintf(`{"title":"in a sprint","type":"task","priority":"low","sprint_id":%d}`, int(sprint["id"].(float64))))
	assert.Equal(t, http.StatusCreated, code, "%v", withSprint)
	assert.Equal(t, "num-6", withSprint["key"])
}
