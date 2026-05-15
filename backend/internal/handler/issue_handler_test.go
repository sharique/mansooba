package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/sharique/jira-go/internal/domain"
	"github.com/sharique/jira-go/internal/dto"
	"github.com/sharique/jira-go/internal/handler"
)

// stubIssueService controls service responses in handler tests.
type stubIssueService struct {
	listFn     func(ctx context.Context, projectKey string, callerID uint, q dto.IssueListQuery) ([]*dto.IssueResponse, error)
	createFn   func(ctx context.Context, projectKey string, callerID uint, req dto.CreateIssueRequest) (*dto.IssueResponse, error)
	findByIDFn func(ctx context.Context, projectKey string, id uint, callerID uint) (*dto.IssueResponse, error)
	updateFn   func(ctx context.Context, projectKey string, id uint, callerID uint, req dto.UpdateIssueRequest) (*dto.IssueResponse, error)
	deleteFn   func(ctx context.Context, projectKey string, id uint, callerID uint) error
}

func (s *stubIssueService) ListByProject(ctx context.Context, projectKey string, callerID uint, q dto.IssueListQuery) ([]*dto.IssueResponse, error) {
	return s.listFn(ctx, projectKey, callerID, q)
}
func (s *stubIssueService) Create(ctx context.Context, projectKey string, callerID uint, req dto.CreateIssueRequest) (*dto.IssueResponse, error) {
	return s.createFn(ctx, projectKey, callerID, req)
}
func (s *stubIssueService) FindByID(ctx context.Context, projectKey string, id uint, callerID uint) (*dto.IssueResponse, error) {
	return s.findByIDFn(ctx, projectKey, id, callerID)
}
func (s *stubIssueService) Update(ctx context.Context, projectKey string, id uint, callerID uint, req dto.UpdateIssueRequest) (*dto.IssueResponse, error) {
	return s.updateFn(ctx, projectKey, id, callerID, req)
}
func (s *stubIssueService) Delete(ctx context.Context, projectKey string, id uint, callerID uint) error {
	return s.deleteFn(ctx, projectKey, id, callerID)
}

// newIssueEcho wires the issue handler into a test Echo instance with userID pre-set.
func newIssueEcho(h *handler.IssueHandler) *echo.Echo {
	e := newEcho()
	api := e.Group("/api/v1", func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("userID", uint(1))
			return next(c)
		}
	})
	issues := api.Group("/projects/:key/issues")
	issues.GET("", h.List)
	issues.POST("", h.Create)
	issues.GET("/:id", h.Get)
	issues.PUT("/:id", h.Update)
	issues.DELETE("/:id", h.Delete)
	return e
}

func TestIssueHandler_List_Returns200(t *testing.T) {
	svc := &stubIssueService{
		listFn: func(_ context.Context, _ string, _ uint, _ dto.IssueListQuery) ([]*dto.IssueResponse, error) {
			return []*dto.IssueResponse{{ID: 1, Key: "PROJ-1", Title: "Test Issue"}}, nil
		},
	}
	e := newIssueEcho(handler.NewIssueHandler(svc))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/PROJ/issues", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestIssueHandler_Create_Returns201(t *testing.T) {
	svc := &stubIssueService{
		createFn: func(_ context.Context, _ string, _ uint, req dto.CreateIssueRequest) (*dto.IssueResponse, error) {
			return &dto.IssueResponse{ID: 1, Key: "PROJ-1", Title: req.Title, Type: req.Type}, nil
		},
	}
	e := newIssueEcho(handler.NewIssueHandler(svc))

	body := `{"title":"Fix login bug","type":"bug","priority":"high"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/PROJ/issues", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestIssueHandler_Get_Returns200(t *testing.T) {
	svc := &stubIssueService{
		findByIDFn: func(_ context.Context, _ string, id uint, _ uint) (*dto.IssueResponse, error) {
			return &dto.IssueResponse{ID: id, Key: "PROJ-1", Title: "Test Issue"}, nil
		},
	}
	e := newIssueEcho(handler.NewIssueHandler(svc))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/PROJ/issues/1", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestIssueHandler_Get_Returns404_ForMissing(t *testing.T) {
	svc := &stubIssueService{
		findByIDFn: func(_ context.Context, _ string, _ uint, _ uint) (*dto.IssueResponse, error) {
			return nil, domain.ErrNotFound
		},
	}
	e := newIssueEcho(handler.NewIssueHandler(svc))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/PROJ/issues/999", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestIssueHandler_Update_Returns200(t *testing.T) {
	svc := &stubIssueService{
		updateFn: func(_ context.Context, _ string, id uint, _ uint, _ dto.UpdateIssueRequest) (*dto.IssueResponse, error) {
			return &dto.IssueResponse{ID: id, Key: "PROJ-1", Status: "in_progress"}, nil
		},
	}
	e := newIssueEcho(handler.NewIssueHandler(svc))

	body := `{"status":"in_progress"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/PROJ/issues/1", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestIssueHandler_Delete_Returns204(t *testing.T) {
	svc := &stubIssueService{
		deleteFn: func(_ context.Context, _ string, _ uint, _ uint) error { return nil },
	}
	e := newIssueEcho(handler.NewIssueHandler(svc))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/PROJ/issues/1", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestIssueHandler_Delete_Returns403_ForNonReporter(t *testing.T) {
	svc := &stubIssueService{
		deleteFn: func(_ context.Context, _ string, _ uint, _ uint) error {
			return domain.ErrForbidden
		},
	}
	e := newIssueEcho(handler.NewIssueHandler(svc))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/PROJ/issues/1", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}
