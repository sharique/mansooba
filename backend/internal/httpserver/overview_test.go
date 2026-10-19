package httpserver_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The overview endpoints answer 200 to members, and its numbers agree with what
// the Sprints and Board pages read for the same project at the same moment.
func TestOverview_MembersSeeItAndNumbersMatchTheSprintEndpoints(t *testing.T) {
	env := newAccessEnv(t)
	e := env.e

	// Put both existing issues into the sprint, give them points and statuses, start it.
	for id, body := range map[string]string{
		env.issue1: `{"sprint_id":` + env.sprint + `,"story_points":3,"status":"in_progress","priority":"critical"}`,
		env.issue2: `{"sprint_id":` + env.sprint + `,"story_points":5,"status":"done"}`,
	} {
		rec := call(e, "PUT", "/api/v1/projects/acc/issues/"+id, env.owner, body)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	}
	rec := call(e, "POST", "/api/v1/projects/acc/sprints/"+env.sprint+"/start", env.owner, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	for _, token := range []string{env.owner, env.member} {
		for _, path := range []string{"sprint", "tasks", "activity", "activity?scope=sprint", "activity?scope=all"} {
			r := call(e, "GET", "/api/v1/projects/acc/overview/"+path, token, "")
			require.Equal(t, http.StatusOK, r.Code, "%s: %s", path, r.Body.String())
		}
	}

	rec = call(e, "GET", "/api/v1/projects/acc/overview/sprint", env.member, "")
	require.Equal(t, http.StatusOK, rec.Code)
	var ov struct {
		Sprint struct {
			Name       string                      `json:"name"`
			InProgress struct{ Tasks, Points int } `json:"in_progress"`
			Done       struct{ Tasks, Points int } `json:"done"`
			Total      struct{ Tasks, Points int } `json:"total"`
		} `json:"sprint"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &ov), rec.Body.String())

	rec = call(e, "GET", "/api/v1/projects/acc/sprints/"+env.sprint, env.member, "")
	require.Equal(t, http.StatusOK, rec.Code)
	var sp struct {
		IssueCount       int `json:"issue_count"`
		TotalStoryPoints int `json:"total_story_points"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &sp))

	assert.Equal(t, sp.IssueCount, ov.Sprint.Total.Tasks)
	assert.Equal(t, sp.TotalStoryPoints, ov.Sprint.Total.Points)
	assert.Equal(t, 1, ov.Sprint.InProgress.Tasks)
	assert.Equal(t, 3, ov.Sprint.InProgress.Points)
	assert.Equal(t, 1, ov.Sprint.Done.Tasks)
	assert.Equal(t, 5, ov.Sprint.Done.Points)

	// The open tasks are exactly the sprint's tasks that are not done.
	rec = call(e, "GET", "/api/v1/projects/acc/overview/tasks", env.member, "")
	var tasks struct {
		SprintActive bool `json:"sprint_active"`
		Items        []struct {
			Key    string `json:"key"`
			Status string `json:"status"`
		} `json:"items"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tasks), rec.Body.String())
	assert.True(t, tasks.SprintActive)
	require.Len(t, tasks.Items, 1)
	assert.Equal(t, "in_progress", tasks.Items[0].Status)

	// Moving a task into progress or done, and the comment from setup, are recent activity.
	rec = call(e, "GET", "/api/v1/projects/acc/overview/activity", env.member, "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"kind":"status_changed"`)
	assert.Contains(t, rec.Body.String(), `"kind":"comment_added"`)
	assert.NotContains(t, rec.Body.String(), "hello", "comment text must never be returned")
}

func TestOverview_BadScopeIs400(t *testing.T) {
	env := newAccessEnv(t)
	rec := call(env.e, "GET", "/api/v1/projects/acc/overview/activity?scope=everything", env.member, "")
	assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
}

func TestOverview_NoSprintMeansNullAndEmpty(t *testing.T) {
	env := newAccessEnv(t) // the sprint exists but is only planned
	rec := call(env.e, "GET", "/api/v1/projects/acc/overview/sprint", env.member, "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"sprint":null}`, rec.Body.String())
	rec = call(env.e, "GET", "/api/v1/projects/acc/overview/tasks", env.member, "")
	assert.JSONEq(t, `{"sprint_active":false,"items":[]}`, rec.Body.String())
	rec = call(env.e, "GET", "/api/v1/projects/acc/overview/activity?scope=sprint", env.member, "")
	assert.JSONEq(t, `{"sprint_active":false,"items":[]}`, rec.Body.String())
}
