package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/handler"
	"github.com/sharique/mansooba/internal/service"
)

// stubSetupService is a controllable stand-in for service.SetupService.
type stubSetupService struct {
	setupRequiredFn func(ctx context.Context) (bool, error)
	createAdminFn   func(ctx context.Context, req dto.SetupAdminRequest) (*dto.AuthResponse, error)
	createUserFn    func(ctx context.Context, req dto.SetupUserRequest) (*dto.SetupUserResponse, error)
	createProjectFn func(ctx context.Context, adminID uint, req dto.SetupProjectRequest) (*dto.SetupProjectResponse, error)
	seedDataFn      func(ctx context.Context, adminID uint) (*dto.SetupSeedResponse, error)
}

func (s *stubSetupService) SetupRequired(ctx context.Context) (bool, error) {
	return s.setupRequiredFn(ctx)
}
func (s *stubSetupService) CreateAdmin(ctx context.Context, req dto.SetupAdminRequest) (*dto.AuthResponse, error) {
	return s.createAdminFn(ctx, req)
}
func (s *stubSetupService) CreateUser(ctx context.Context, req dto.SetupUserRequest) (*dto.SetupUserResponse, error) {
	return s.createUserFn(ctx, req)
}
func (s *stubSetupService) CreateProject(ctx context.Context, adminID uint, req dto.SetupProjectRequest) (*dto.SetupProjectResponse, error) {
	return s.createProjectFn(ctx, adminID, req)
}
func (s *stubSetupService) SeedData(ctx context.Context, adminID uint) (*dto.SetupSeedResponse, error) {
	if s.seedDataFn != nil {
		return s.seedDataFn(ctx, adminID)
	}
	return nil, nil
}

// --- GET /setup/status ---

func TestSetupHandler_Status_Returns200_WhenRequired(t *testing.T) {
	svc := &stubSetupService{
		setupRequiredFn: func(_ context.Context) (bool, error) { return true, nil },
	}
	e := newEcho()
	h := handler.NewSetupHandler(svc)
	e.GET("/setup/status", h.Status)

	req := httptest.NewRequest(http.MethodGet, "/setup/status", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp dto.SetupStatusResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode: %v", err)
	}
	if !resp.SetupRequired {
		t.Error("expected setup_required=true")
	}
}

func TestSetupHandler_Status_Returns200_WhenNotRequired(t *testing.T) {
	svc := &stubSetupService{
		setupRequiredFn: func(_ context.Context) (bool, error) { return false, nil },
	}
	e := newEcho()
	h := handler.NewSetupHandler(svc)
	e.GET("/setup/status", h.Status)

	req := httptest.NewRequest(http.MethodGet, "/setup/status", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp dto.SetupStatusResponse
	json.NewDecoder(rec.Body).Decode(&resp) //nolint:errcheck
	if resp.SetupRequired {
		t.Error("expected setup_required=false")
	}
}

// --- POST /setup/admin ---

func TestSetupHandler_CreateAdmin_Returns201(t *testing.T) {
	svc := &stubSetupService{
		createAdminFn: func(_ context.Context, req dto.SetupAdminRequest) (*dto.AuthResponse, error) {
			return &dto.AuthResponse{
				AccessToken: "wizard.jwt.token",
				User:        dto.UserDTO{ID: 1, Name: req.FullName, Email: req.Email},
			}, nil
		},
	}
	e := newEcho()
	h := handler.NewSetupHandler(svc)
	e.POST("/setup/admin", h.CreateAdmin)

	body := `{"full_name":"Alice Admin","email":"alice@example.com","password":"Secret123"}`
	req := httptest.NewRequest(http.MethodPost, "/setup/admin", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp dto.AuthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode: %v", err)
	}
	if resp.AccessToken != "wizard.jwt.token" {
		t.Errorf("unexpected token: %s", resp.AccessToken)
	}
}

func TestSetupHandler_CreateAdmin_Returns400_OnMissingFields(t *testing.T) {
	svc := &stubSetupService{
		createAdminFn: func(_ context.Context, req dto.SetupAdminRequest) (*dto.AuthResponse, error) {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "validation failed")
		},
	}
	e := newEcho()
	h := handler.NewSetupHandler(svc)
	e.POST("/setup/admin", h.CreateAdmin)

	body := `{"email":"alice@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/setup/admin", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSetupHandler_CreateAdmin_Returns409_WhenAdminExists(t *testing.T) {
	svc := &stubSetupService{
		createAdminFn: func(_ context.Context, req dto.SetupAdminRequest) (*dto.AuthResponse, error) {
			return nil, service.ErrSetupComplete
		},
	}
	e := newEcho()
	h := handler.NewSetupHandler(svc)
	e.POST("/setup/admin", h.CreateAdmin)

	body := `{"full_name":"Alice","email":"alice@example.com","password":"Secret123"}`
	req := httptest.NewRequest(http.MethodPost, "/setup/admin", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- POST /setup/user ---

func TestSetupHandler_CreateUser_Returns201(t *testing.T) {
	svc := &stubSetupService{
		createUserFn: func(_ context.Context, req dto.SetupUserRequest) (*dto.SetupUserResponse, error) {
			return &dto.SetupUserResponse{UserID: 2, Name: req.FullName, Email: req.Email}, nil
		},
	}
	e := newEcho()
	h := handler.NewSetupHandler(svc)
	e.POST("/setup/user", h.CreateUser)

	body := `{"full_name":"Bob Member","email":"bob@example.com","password":"Secret456"}`
	req := httptest.NewRequest(http.MethodPost, "/setup/user", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp dto.SetupUserResponse
	json.NewDecoder(rec.Body).Decode(&resp) //nolint:errcheck
	if resp.UserID != 2 {
		t.Errorf("unexpected user_id: %d", resp.UserID)
	}
}

func TestSetupHandler_CreateUser_Returns400_OnMissingFields(t *testing.T) {
	svc := &stubSetupService{
		createUserFn: func(_ context.Context, req dto.SetupUserRequest) (*dto.SetupUserResponse, error) {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "validation failed")
		},
	}
	e := newEcho()
	h := handler.NewSetupHandler(svc)
	e.POST("/setup/user", h.CreateUser)

	body := `{"email":"bob@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/setup/user", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSetupHandler_CreateUser_Returns409_OnDuplicateEmail(t *testing.T) {
	svc := &stubSetupService{
		createUserFn: func(_ context.Context, req dto.SetupUserRequest) (*dto.SetupUserResponse, error) {
			return nil, domain.ErrConflict
		},
	}
	e := newEcho()
	h := handler.NewSetupHandler(svc)
	e.POST("/setup/user", h.CreateUser)

	body := `{"full_name":"Bob","email":"bob@example.com","password":"Secret456"}`
	req := httptest.NewRequest(http.MethodPost, "/setup/user", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- POST /setup/project ---

func TestSetupHandler_CreateProject_Returns201(t *testing.T) {
	svc := &stubSetupService{
		createProjectFn: func(_ context.Context, adminID uint, req dto.SetupProjectRequest) (*dto.SetupProjectResponse, error) {
			return &dto.SetupProjectResponse{ProjectID: 1, ProjectKey: "MFP", Name: req.Name}, nil
		},
	}
	e := newEcho()
	h := handler.NewSetupHandler(svc)
	e.POST("/setup/project", func(c echo.Context) error {
		c.Set("userID", uint(1))
		return h.CreateProject(c)
	})

	body := `{"name":"My First Project"}`
	req := httptest.NewRequest(http.MethodPost, "/setup/project", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp dto.SetupProjectResponse
	json.NewDecoder(rec.Body).Decode(&resp) //nolint:errcheck
	if resp.ProjectKey != "MFP" {
		t.Errorf("unexpected key: %s", resp.ProjectKey)
	}
}

func TestSetupHandler_CreateProject_Returns201_NoMembership(t *testing.T) {
	svc := &stubSetupService{
		createProjectFn: func(_ context.Context, adminID uint, req dto.SetupProjectRequest) (*dto.SetupProjectResponse, error) {
			if req.AddUserID != 0 {
				t.Errorf("expected add_user_id=0, got %d", req.AddUserID)
			}
			return &dto.SetupProjectResponse{ProjectID: 1, ProjectKey: "MFP", Name: req.Name}, nil
		},
	}
	e := newEcho()
	h := handler.NewSetupHandler(svc)
	e.POST("/setup/project", func(c echo.Context) error {
		c.Set("userID", uint(1))
		return h.CreateProject(c)
	})

	body := `{"name":"My First Project","add_user_id":0}`
	req := httptest.NewRequest(http.MethodPost, "/setup/project", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSetupHandler_CreateProject_Returns404_WhenUserNotFound(t *testing.T) {
	svc := &stubSetupService{
		createProjectFn: func(_ context.Context, adminID uint, req dto.SetupProjectRequest) (*dto.SetupProjectResponse, error) {
			return nil, echo.NewHTTPError(http.StatusNotFound, "user not found")
		},
	}
	e := newEcho()
	h := handler.NewSetupHandler(svc)
	e.POST("/setup/project", func(c echo.Context) error {
		c.Set("userID", uint(1))
		return h.CreateProject(c)
	})

	body := `{"name":"My Project","add_user_id":999}`
	req := httptest.NewRequest(http.MethodPost, "/setup/project", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// --- POST /setup/seed ---

func TestSetupHandler_SeedData_Returns200OnSuccess(t *testing.T) {
	svc := &stubSetupService{
		seedDataFn: func(_ context.Context, _ uint) (*dto.SetupSeedResponse, error) {
			return &dto.SetupSeedResponse{Skipped: false, ProjectKey: "DEMO", ProjectName: "Mansooba Demo"}, nil
		},
	}
	e := newEcho()
	h := handler.NewSetupHandler(svc)
	e.POST("/setup/seed", func(c echo.Context) error {
		c.Set("userID", uint(1))
		return h.SeedData(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/setup/seed", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp dto.SetupSeedResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Skipped {
		t.Error("expected skipped=false")
	}
	if resp.ProjectKey != "DEMO" {
		t.Errorf("expected project_key=DEMO, got %s", resp.ProjectKey)
	}
}

func TestSetupHandler_SeedData_Returns200WhenSkipped(t *testing.T) {
	svc := &stubSetupService{
		seedDataFn: func(_ context.Context, _ uint) (*dto.SetupSeedResponse, error) {
			return &dto.SetupSeedResponse{Skipped: true, ProjectKey: "DEMO", ProjectName: "Mansooba Demo"}, nil
		},
	}
	e := newEcho()
	h := handler.NewSetupHandler(svc)
	e.POST("/setup/seed", func(c echo.Context) error {
		c.Set("userID", uint(1))
		return h.SeedData(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/setup/seed", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp dto.SetupSeedResponse
	json.NewDecoder(rec.Body).Decode(&resp) //nolint:errcheck
	if !resp.Skipped {
		t.Error("expected skipped=true")
	}
}

func TestSetupHandler_SeedData_Returns500OnError(t *testing.T) {
	svc := &stubSetupService{
		seedDataFn: func(_ context.Context, _ uint) (*dto.SetupSeedResponse, error) {
			return nil, errors.New("db error")
		},
	}
	e := newEcho()
	h := handler.NewSetupHandler(svc)
	e.POST("/setup/seed", func(c echo.Context) error {
		c.Set("userID", uint(1))
		return h.SeedData(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/setup/seed", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSetupHandler_CreateProject_Returns400_OnMissingName(t *testing.T) {
	svc := &stubSetupService{
		createProjectFn: func(_ context.Context, adminID uint, req dto.SetupProjectRequest) (*dto.SetupProjectResponse, error) {
			return nil, echo.NewHTTPError(http.StatusBadRequest, "validation failed")
		},
	}
	e := newEcho()
	h := handler.NewSetupHandler(svc)
	e.POST("/setup/project", func(c echo.Context) error {
		c.Set("userID", uint(1))
		return h.CreateProject(c)
	})

	body := `{"description":"no name"}`
	req := httptest.NewRequest(http.MethodPost, "/setup/project", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}
