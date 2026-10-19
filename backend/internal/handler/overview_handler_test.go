package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/handler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubOverviewService struct {
	err       error
	gotKey    string
	gotCaller uint
	gotScope  string
}

func (s *stubOverviewService) Sprint(_ context.Context, key string, caller uint) (*dto.OverviewSprintResponse, error) {
	s.gotKey, s.gotCaller = key, caller
	if s.err != nil {
		return nil, s.err
	}
	return &dto.OverviewSprintResponse{Sprint: &dto.SprintSummaryResponse{ID: 3, Name: "Sprint 3"}}, nil
}

func (s *stubOverviewService) Tasks(_ context.Context, key string, caller uint) (*dto.OverviewTasksResponse, error) {
	s.gotKey, s.gotCaller = key, caller
	if s.err != nil {
		return nil, s.err
	}
	return &dto.OverviewTasksResponse{SprintActive: true, Items: []dto.OpenTaskResponse{{Key: "OVR-1"}}}, nil
}

func (s *stubOverviewService) Activity(_ context.Context, key string, caller uint, scope string) (*dto.OverviewActivityResponse, error) {
	s.gotKey, s.gotCaller, s.gotScope = key, caller, scope
	if s.err != nil {
		return nil, s.err
	}
	return &dto.OverviewActivityResponse{Items: []dto.RecentActivityResponse{}}, nil
}

func newOverviewEcho(svc *stubOverviewService) *echo.Echo {
	h := handler.NewOverviewHandler(svc)
	e := newEcho()
	api := e.Group("/api/v1", func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("userID", uint(42))
			return next(c)
		}
	})
	api.GET("/projects/:key/overview/sprint", h.Sprint)
	api.GET("/projects/:key/overview/tasks", h.Tasks)
	api.GET("/projects/:key/overview/activity", h.Activity)
	return e
}

func getOverview(e *echo.Echo, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestOverviewHandler_ReturnsContractShapes(t *testing.T) {
	svc := &stubOverviewService{}
	e := newOverviewEcho(svc)

	rec := getOverview(e, "/api/v1/projects/OVR/overview/sprint")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var sprint map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &sprint))
	assert.Equal(t, "Sprint 3", sprint["sprint"].(map[string]any)["name"])
	assert.Equal(t, "OVR", svc.gotKey)
	assert.Equal(t, uint(42), svc.gotCaller)

	rec = getOverview(e, "/api/v1/projects/OVR/overview/tasks")
	require.Equal(t, http.StatusOK, rec.Code)
	var tasks map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tasks))
	assert.Equal(t, true, tasks["sprint_active"])
	assert.Len(t, tasks["items"], 1)

	rec = getOverview(e, "/api/v1/projects/OVR/overview/activity")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"sprint_active":false,"items":[]}`, rec.Body.String())
}

func TestOverviewHandler_ActivityScope(t *testing.T) {
	svc := &stubOverviewService{}
	e := newOverviewEcho(svc)

	getOverview(e, "/api/v1/projects/OVR/overview/activity")
	assert.Equal(t, "", svc.gotScope, "no scope is passed through as empty; the service defaults it")
	getOverview(e, "/api/v1/projects/OVR/overview/activity?scope=sprint")
	assert.Equal(t, "sprint", svc.gotScope)
	getOverview(e, "/api/v1/projects/OVR/overview/activity?scope=all")
	assert.Equal(t, "all", svc.gotScope)
}

func TestOverviewHandler_MapsErrors(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"not found", domain.ErrNotFound, http.StatusNotFound},
		{"forbidden", domain.ErrForbidden, http.StatusForbidden},
		{"bad scope", domain.ErrInvalidInput, http.StatusBadRequest},
	}
	for _, tc := range cases {
		for _, path := range []string{"sprint", "tasks", "activity"} {
			t.Run(tc.name+"/"+path, func(t *testing.T) {
				e := newOverviewEcho(&stubOverviewService{err: tc.err})
				rec := getOverview(e, "/api/v1/projects/OVR/overview/"+path)
				assert.Equal(t, tc.want, rec.Code)
			})
		}
	}
}

func TestOverviewHandler_DoesNotLeakInfrastructureErrors(t *testing.T) {
	e := newOverviewEcho(&stubOverviewService{err: errors.New("pq: relation \"issues\" does not exist")})
	rec := getOverview(e, "/api/v1/projects/OVR/overview/tasks")
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), "pq:")
	assert.NotContains(t, rec.Body.String(), "relation")
}
