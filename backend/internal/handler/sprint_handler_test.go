package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/sharique/jira-go/internal/domain"
	"github.com/sharique/jira-go/internal/dto"
	"github.com/sharique/jira-go/internal/handler"
	"github.com/sharique/jira-go/internal/service"
)

type stubSprintService struct {
	listFn       func(ctx context.Context, projectKey string, callerID uint) ([]*dto.SprintResponse, error)
	createFn     func(ctx context.Context, projectKey string, callerID uint, req dto.CreateSprintRequest) (*dto.SprintResponse, error)
	getFn        func(ctx context.Context, projectKey string, id uint, callerID uint) (*dto.SprintResponse, error)
	updateFn     func(ctx context.Context, projectKey string, id uint, callerID uint, req dto.UpdateSprintRequest) (*dto.SprintResponse, error)
	deleteFn     func(ctx context.Context, projectKey string, id uint, callerID uint) error
	startFn      func(ctx context.Context, projectKey string, id uint, callerID uint) (*dto.SprintResponse, error)
	completeFn   func(ctx context.Context, projectKey string, id uint, callerID uint, req dto.CompleteSprintRequest) (*dto.SprintResponse, error)
	backlogFn    func(ctx context.Context, projectKey string, callerID uint) ([]*domain.Issue, error)
	getIssuesFn  func(ctx context.Context, projectKey string, id uint, callerID uint) ([]*domain.Issue, error)
	burndownFn   func(ctx context.Context, projectKey string, id uint, callerID uint) (*dto.BurndownResponse, error)
	velocityFn   func(ctx context.Context, projectKey string, callerID uint) ([]dto.VelocityDataPoint, error)
}

func (s *stubSprintService) List(ctx context.Context, projectKey string, callerID uint) ([]*dto.SprintResponse, error) {
	return s.listFn(ctx, projectKey, callerID)
}
func (s *stubSprintService) Create(ctx context.Context, projectKey string, callerID uint, req dto.CreateSprintRequest) (*dto.SprintResponse, error) {
	return s.createFn(ctx, projectKey, callerID, req)
}
func (s *stubSprintService) Get(ctx context.Context, projectKey string, id uint, callerID uint) (*dto.SprintResponse, error) {
	return s.getFn(ctx, projectKey, id, callerID)
}
func (s *stubSprintService) Update(ctx context.Context, projectKey string, id uint, callerID uint, req dto.UpdateSprintRequest) (*dto.SprintResponse, error) {
	return s.updateFn(ctx, projectKey, id, callerID, req)
}
func (s *stubSprintService) Delete(ctx context.Context, projectKey string, id uint, callerID uint) error {
	return s.deleteFn(ctx, projectKey, id, callerID)
}
func (s *stubSprintService) Start(ctx context.Context, projectKey string, id uint, callerID uint) (*dto.SprintResponse, error) {
	return s.startFn(ctx, projectKey, id, callerID)
}
func (s *stubSprintService) Complete(ctx context.Context, projectKey string, id uint, callerID uint, req dto.CompleteSprintRequest) (*dto.SprintResponse, error) {
	return s.completeFn(ctx, projectKey, id, callerID, req)
}
func (s *stubSprintService) Backlog(ctx context.Context, projectKey string, callerID uint) ([]*domain.Issue, error) {
	return s.backlogFn(ctx, projectKey, callerID)
}
func (s *stubSprintService) GetIssues(ctx context.Context, projectKey string, id uint, callerID uint) ([]*domain.Issue, error) {
	return s.getIssuesFn(ctx, projectKey, id, callerID)
}
func (s *stubSprintService) Burndown(ctx context.Context, projectKey string, id uint, callerID uint) (*dto.BurndownResponse, error) {
	return s.burndownFn(ctx, projectKey, id, callerID)
}
func (s *stubSprintService) Velocity(ctx context.Context, projectKey string, callerID uint) ([]dto.VelocityDataPoint, error) {
	return s.velocityFn(ctx, projectKey, callerID)
}

var _ service.SprintService = (*stubSprintService)(nil)

func newSprintEcho(h *handler.SprintHandler) *echo.Echo {
	e := newEcho()
	api := e.Group("/api/v1", func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("userID", uint(1))
			return next(c)
		}
	})
	sprints := api.Group("/projects/:key/sprints")
	sprints.GET("", h.List)
	sprints.POST("", h.Create)
	sprints.GET("/:id", h.Get)
	sprints.PUT("/:id", h.Update)
	sprints.DELETE("/:id", h.Delete)
	sprints.POST("/:id/start", h.Start)
	sprints.POST("/:id/complete", h.Complete)
	sprints.GET("/:id/burndown", h.Burndown)
	sprints.GET("/:id/issues", h.GetIssues)
	api.GET("/projects/:key/backlog", h.Backlog)
	api.GET("/projects/:key/velocity", h.Velocity)
	return e
}

func TestSprintHandler_List_Returns200(t *testing.T) {
	svc := &stubSprintService{
		listFn: func(_ context.Context, _ string, _ uint) ([]*dto.SprintResponse, error) {
			return []*dto.SprintResponse{{ID: 1, Name: "Sprint 1", Status: domain.SprintStatusPlanning}}, nil
		},
	}
	e := newSprintEcho(handler.NewSprintHandler(svc))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/TEST/sprints", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSprintHandler_Create_Returns201(t *testing.T) {
	svc := &stubSprintService{
		createFn: func(_ context.Context, _ string, _ uint, req dto.CreateSprintRequest) (*dto.SprintResponse, error) {
			return &dto.SprintResponse{ID: 1, Name: req.Name, Status: domain.SprintStatusPlanning}, nil
		},
	}
	e := newSprintEcho(handler.NewSprintHandler(svc))

	body, _ := json.Marshal(dto.CreateSprintRequest{Name: "Sprint 1"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/TEST/sprints", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp dto.SprintResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Name != "Sprint 1" {
		t.Errorf("expected Sprint 1, got %s", resp.Name)
	}
}

func TestSprintHandler_Start_Returns409_WhenAlreadyActive(t *testing.T) {
	svc := &stubSprintService{
		startFn: func(_ context.Context, _ string, _ uint, _ uint) (*dto.SprintResponse, error) {
			return nil, domain.ErrSprintAlreadyActive
		},
	}
	e := newSprintEcho(handler.NewSprintHandler(svc))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/TEST/sprints/1/start", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSprintHandler_Delete_Returns404_ForMissing(t *testing.T) {
	svc := &stubSprintService{
		deleteFn: func(_ context.Context, _ string, _ uint, _ uint) error {
			return domain.ErrNotFound
		},
	}
	e := newSprintEcho(handler.NewSprintHandler(svc))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/TEST/sprints/99", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSprintHandler_Backlog_Returns200(t *testing.T) {
	svc := &stubSprintService{
		backlogFn: func(_ context.Context, _ string, _ uint) ([]*domain.Issue, error) {
			pts := 3
			return []*domain.Issue{
				{ID: 1, Title: "Backlog task", Priority: domain.IssuePriorityHigh, Status: domain.IssueStatusTodo, StoryPoints: &pts},
			}, nil
		},
	}
	e := newSprintEcho(handler.NewSprintHandler(svc))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/TEST/backlog", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSprintHandler_GetIssues_Returns200(t *testing.T) {
	svc := &stubSprintService{
		getIssuesFn: func(_ context.Context, _ string, _ uint, _ uint) ([]*domain.Issue, error) {
			pts := 5
			return []*domain.Issue{
				{ID: 1, Title: "Sprint task", Priority: domain.IssuePriorityMedium, Status: domain.IssueStatusTodo, StoryPoints: &pts},
			}, nil
		},
	}
	e := newSprintEcho(handler.NewSprintHandler(svc))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/TEST/sprints/1/issues", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSprintHandler_Burndown_Returns400_WhenNotStarted(t *testing.T) {
	svc := &stubSprintService{
		burndownFn: func(_ context.Context, _ string, _ uint, _ uint) (*dto.BurndownResponse, error) {
			return nil, domain.ErrSprintNotStarted
		},
	}
	e := newSprintEcho(handler.NewSprintHandler(svc))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/TEST/sprints/1/burndown", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSprintHandler_Velocity_Returns200WithData(t *testing.T) {
	svc := &stubSprintService{
		velocityFn: func(_ context.Context, _ string, _ uint) ([]dto.VelocityDataPoint, error) {
			return []dto.VelocityDataPoint{
				{SprintID: 1, SprintName: "Sprint 1", Committed: 10, Completed: 7},
				{SprintID: 2, SprintName: "Sprint 2", Committed: 8, Completed: 8},
			}, nil
		},
	}
	e := newSprintEcho(handler.NewSprintHandler(svc))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/TEST/velocity", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp []dto.VelocityDataPoint
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp) != 2 {
		t.Errorf("expected 2 data points, got %d", len(resp))
	}
	if resp[0].SprintName != "Sprint 1" || resp[0].Committed != 10 || resp[0].Completed != 7 {
		t.Errorf("unexpected first data point: %+v", resp[0])
	}
}

func TestSprintHandler_Velocity_Returns404_WhenProjectNotFound(t *testing.T) {
	svc := &stubSprintService{
		velocityFn: func(_ context.Context, _ string, _ uint) ([]dto.VelocityDataPoint, error) {
			return nil, domain.ErrNotFound
		},
	}
	e := newSprintEcho(handler.NewSprintHandler(svc))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/NOPE/velocity", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSprintHandler_Velocity_Returns200EmptyArray_WhenNoCompletedSprints(t *testing.T) {
	svc := &stubSprintService{
		velocityFn: func(_ context.Context, _ string, _ uint) ([]dto.VelocityDataPoint, error) {
			return []dto.VelocityDataPoint{}, nil
		},
	}
	e := newSprintEcho(handler.NewSprintHandler(svc))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/TEST/velocity", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp []dto.VelocityDataPoint
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("expected empty array, got %d items", len(resp))
	}
}

func TestSprintHandler_Velocity_Returns403_WhenForbidden(t *testing.T) {
	svc := &stubSprintService{
		velocityFn: func(_ context.Context, _ string, _ uint) ([]dto.VelocityDataPoint, error) {
			return nil, domain.ErrForbidden
		},
	}
	e := newSprintEcho(handler.NewSprintHandler(svc))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/TEST/velocity", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}
