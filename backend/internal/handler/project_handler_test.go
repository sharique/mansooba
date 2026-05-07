package handler_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/sharique/jira-go/internal/domain"
	"github.com/sharique/jira-go/internal/dto"
	"github.com/sharique/jira-go/internal/handler"
)

// stubProjectService controls service responses in handler tests.
type stubProjectService struct {
	listFn         func(ctx context.Context, callerID uint) ([]*dto.ProjectResponse, error)
	createFn       func(ctx context.Context, callerID uint, req dto.CreateProjectRequest) (*dto.ProjectResponse, error)
	findByKeyFn    func(ctx context.Context, key string, callerID uint) (*dto.ProjectResponse, error)
	updateFn       func(ctx context.Context, key string, callerID uint, req dto.UpdateProjectRequest) (*dto.ProjectResponse, error)
	deleteFn       func(ctx context.Context, key string, callerID uint) error
	listMembersFn  func(ctx context.Context, key string, callerID uint) ([]*dto.MemberResponse, error)
	addMemberFn    func(ctx context.Context, key string, callerID uint, req dto.AddMemberRequest) error
	removeMemberFn func(ctx context.Context, key string, callerID uint, targetUserID uint) error
}

func (s *stubProjectService) List(ctx context.Context, callerID uint) ([]*dto.ProjectResponse, error) {
	return s.listFn(ctx, callerID)
}
func (s *stubProjectService) Create(ctx context.Context, callerID uint, req dto.CreateProjectRequest) (*dto.ProjectResponse, error) {
	return s.createFn(ctx, callerID, req)
}
func (s *stubProjectService) FindByKey(ctx context.Context, key string, callerID uint) (*dto.ProjectResponse, error) {
	return s.findByKeyFn(ctx, key, callerID)
}
func (s *stubProjectService) Update(ctx context.Context, key string, callerID uint, req dto.UpdateProjectRequest) (*dto.ProjectResponse, error) {
	return s.updateFn(ctx, key, callerID, req)
}
func (s *stubProjectService) Delete(ctx context.Context, key string, callerID uint) error {
	return s.deleteFn(ctx, key, callerID)
}
func (s *stubProjectService) ListMembers(ctx context.Context, key string, callerID uint) ([]*dto.MemberResponse, error) {
	return s.listMembersFn(ctx, key, callerID)
}
func (s *stubProjectService) AddMember(ctx context.Context, key string, callerID uint, req dto.AddMemberRequest) error {
	return s.addMemberFn(ctx, key, callerID, req)
}
func (s *stubProjectService) RemoveMember(ctx context.Context, key string, callerID uint, targetUserID uint) error {
	return s.removeMemberFn(ctx, key, callerID, targetUserID)
}

// newProjectEcho creates an Echo instance with the userID already set in context (simulates auth middleware).
func newProjectEcho(h *handler.ProjectHandler) *echo.Echo {
	e := newEcho()
	api := e.Group("/api/v1", func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("userID", uint(1))
			return next(c)
		}
	})
	projects := api.Group("/projects")
	projects.GET("", h.List)
	projects.POST("", h.Create)
	projects.GET("/:key", h.Get)
	projects.PUT("/:key", h.Update)
	projects.DELETE("/:key", h.Delete)
	projects.GET("/:key/members", h.ListMembers)
	projects.POST("/:key/members", h.AddMember)
	projects.DELETE("/:key/members/:userId", h.RemoveMember)
	return e
}

func TestProjectHandler_List_Returns200(t *testing.T) {
	svc := &stubProjectService{
		listFn: func(_ context.Context, _ uint) ([]*dto.ProjectResponse, error) {
			return []*dto.ProjectResponse{{ID: 1, Key: "PROJ", Name: "Test"}}, nil
		},
	}
	e := newProjectEcho(handler.NewProjectHandler(svc))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProjectHandler_Create_Returns201(t *testing.T) {
	svc := &stubProjectService{
		createFn: func(_ context.Context, _ uint, req dto.CreateProjectRequest) (*dto.ProjectResponse, error) {
			return &dto.ProjectResponse{ID: 1, Key: "MYPR", Name: req.Name}, nil
		},
	}
	e := newProjectEcho(handler.NewProjectHandler(svc))

	body := `{"name":"My Project"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp dto.ProjectResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Key != "MYPR" {
		t.Errorf("expected key MYPR, got %s", resp.Key)
	}
}

func TestProjectHandler_Get_Returns200(t *testing.T) {
	svc := &stubProjectService{
		findByKeyFn: func(_ context.Context, key string, _ uint) (*dto.ProjectResponse, error) {
			return &dto.ProjectResponse{ID: 1, Key: key, Name: "Test"}, nil
		},
	}
	e := newProjectEcho(handler.NewProjectHandler(svc))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/MYPR", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProjectHandler_Get_Returns403_ForNonMember(t *testing.T) {
	svc := &stubProjectService{
		findByKeyFn: func(_ context.Context, _ string, _ uint) (*dto.ProjectResponse, error) {
			return nil, domain.ErrForbidden
		},
	}
	e := newProjectEcho(handler.NewProjectHandler(svc))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/MYPR", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProjectHandler_Update_Returns200(t *testing.T) {
	svc := &stubProjectService{
		updateFn: func(_ context.Context, key string, _ uint, _ dto.UpdateProjectRequest) (*dto.ProjectResponse, error) {
			return &dto.ProjectResponse{ID: 1, Key: key, Name: "Updated"}, nil
		},
	}
	e := newProjectEcho(handler.NewProjectHandler(svc))

	body := `{"name":"Updated"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/projects/MYPR", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProjectHandler_Delete_Returns204(t *testing.T) {
	svc := &stubProjectService{
		deleteFn: func(_ context.Context, _ string, _ uint) error { return nil },
	}
	e := newProjectEcho(handler.NewProjectHandler(svc))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/MYPR", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProjectHandler_Delete_Returns403_ForNonOwner(t *testing.T) {
	svc := &stubProjectService{
		deleteFn: func(_ context.Context, _ string, _ uint) error { return domain.ErrForbidden },
	}
	e := newProjectEcho(handler.NewProjectHandler(svc))

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/MYPR", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProjectHandler_ListMembers_Returns200(t *testing.T) {
	svc := &stubProjectService{
		listMembersFn: func(_ context.Context, _ string, _ uint) ([]*dto.MemberResponse, error) {
			return []*dto.MemberResponse{{UserID: 1, Name: "Alice", Email: "alice@example.com", Role: "admin"}}, nil
		},
	}
	e := newProjectEcho(handler.NewProjectHandler(svc))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/MYPR/members", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProjectHandler_AddMember_Returns201(t *testing.T) {
	svc := &stubProjectService{
		addMemberFn: func(_ context.Context, _ string, _ uint, _ dto.AddMemberRequest) error { return nil },
	}
	e := newProjectEcho(handler.NewProjectHandler(svc))

	body := `{"email":"bob@example.com","role":"member"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/MYPR/members", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestProjectHandler_RemoveMember_Returns204(t *testing.T) {
	svc := &stubProjectService{
		removeMemberFn: func(_ context.Context, _ string, _ uint, _ uint) error { return nil },
	}
	e := newProjectEcho(handler.NewProjectHandler(svc))

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/projects/MYPR/members/%d", 2), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}
