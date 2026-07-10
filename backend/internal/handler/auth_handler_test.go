package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/handler"
)

// stubAuthService is a controllable stand-in for service.AuthService.
type stubAuthService struct {
	registerFn func(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error)
	loginFn    func(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error)
	refreshFn  func(ctx context.Context, token string) (string, error)
	logoutFn   func(ctx context.Context, token string) error
}

func (s *stubAuthService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error) {
	return s.registerFn(ctx, req)
}
func (s *stubAuthService) Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error) {
	return s.loginFn(ctx, req)
}
func (s *stubAuthService) Refresh(ctx context.Context, token string) (string, error) {
	return s.refreshFn(ctx, token)
}
func (s *stubAuthService) Logout(ctx context.Context, token string) error {
	if s.logoutFn != nil {
		return s.logoutFn(ctx, token)
	}
	return nil
}

type testValidator struct{ v *validator.Validate }

func (tv *testValidator) Validate(i any) error { return tv.v.Struct(i) }

func newEcho() *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	v := validator.New()
	v.RegisterValidation("password_complexity", func(fl validator.FieldLevel) bool { //nolint:errcheck
		pw := fl.Field().String()
		var hasUpper, hasLower, hasDigit bool
		for _, r := range pw {
			switch {
			case r >= 'A' && r <= 'Z':
				hasUpper = true
			case r >= 'a' && r <= 'z':
				hasLower = true
			case r >= '0' && r <= '9':
				hasDigit = true
			}
		}
		return len(pw) >= 8 && hasUpper && hasLower && hasDigit
	})
	e.Validator = &testValidator{v}
	return e
}

func TestAuthHandler_Register_Returns201(t *testing.T) {
	svc := &stubAuthService{
		registerFn: func(_ context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error) {
			return &dto.AuthResponse{
				AccessToken: "test.access.token",
				User:        dto.UserDTO{ID: 1, Name: req.FullName, Email: req.Email},
			}, nil
		},
	}
	adminUserSvc := &stubAuthUserService{
		getProfileFn: func(_ context.Context, _ uint) (*dto.UserProfileResponse, error) {
			return &dto.UserProfileResponse{ID: 1, IsAdmin: true}, nil
		},
	}

	e := newEcho()
	h := handler.NewAuthHandler(svc, adminUserSvc)
	e.POST("/auth/register", func(c echo.Context) error {
		c.Set("userID", uint(1))
		return h.Register(c)
	})

	body := `{"full_name":"Alice","email":"alice@example.com","password":"Password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp dto.AuthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if resp.AccessToken != "test.access.token" {
		t.Errorf("unexpected access token: %s", resp.AccessToken)
	}
}

func TestAuthHandler_Login_Returns200(t *testing.T) {
	svc := &stubAuthService{
		loginFn: func(_ context.Context, req dto.LoginRequest) (*dto.AuthResponse, error) {
			return &dto.AuthResponse{
				AccessToken: "login.access.token",
				User:        dto.UserDTO{ID: 1, Email: req.Email},
			}, nil
		},
	}

	e := newEcho()
	h := handler.NewAuthHandler(svc, nil)
	e.POST("/auth/login", h.Login)

	body := `{"email":"alice@example.com","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAuthHandler_Refresh_Returns200(t *testing.T) {
	svc := &stubAuthService{
		refreshFn: func(_ context.Context, token string) (string, error) {
			return "new.access.token", nil
		},
	}

	e := newEcho()
	h := handler.NewAuthHandler(svc, nil)
	e.POST("/auth/refresh", h.Refresh)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "some.refresh.token"})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["access_token"] != "new.access.token" {
		t.Errorf("unexpected token in response: %v", resp)
	}
}

// stubAuthUserService satisfies service.UserService for admin check tests.
type stubAuthUserService struct {
	getProfileFn func(ctx context.Context, userID uint) (*dto.UserProfileResponse, error)
}

func (s *stubAuthUserService) GetProfile(ctx context.Context, userID uint) (*dto.UserProfileResponse, error) {
	if s.getProfileFn != nil {
		return s.getProfileFn(ctx, userID)
	}
	return &dto.UserProfileResponse{ID: userID, IsAdmin: false}, nil
}
func (s *stubAuthUserService) UpdateProfile(_ context.Context, _ uint, _ dto.UpdateProfileRequest) (*dto.UserProfileResponse, error) {
	return nil, nil
}
func (s *stubAuthUserService) UploadAvatar(_ context.Context, _ uint, _ string, _ []byte, _ string) (*dto.UserProfileResponse, error) {
	return nil, nil
}
func (s *stubAuthUserService) DeleteAvatar(_ context.Context, _ uint) (*dto.UserProfileResponse, error) {
	return nil, nil
}

func TestAuthHandler_Register_Returns403_ForNonAdmin(t *testing.T) {
	svc := &stubAuthService{
		registerFn: func(_ context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error) {
			t.Error("Register service should not be called for non-admin")
			return nil, nil
		},
	}
	userSvc := &stubAuthUserService{
		getProfileFn: func(_ context.Context, _ uint) (*dto.UserProfileResponse, error) {
			return &dto.UserProfileResponse{ID: 1, IsAdmin: false}, nil
		},
	}

	e := newEcho()
	h := handler.NewAuthHandler(svc, userSvc)
	e.POST("/auth/register", func(c echo.Context) error {
		c.Set("userID", uint(1))
		return h.Register(c)
	})

	body := `{"full_name":"Bob","email":"bob@example.com","password":"Password1"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAuthHandler_Register_Returns201_ForAdmin(t *testing.T) {
	svc := &stubAuthService{
		registerFn: func(_ context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error) {
			return &dto.AuthResponse{
				AccessToken: "admin.created.token",
				User:        dto.UserDTO{ID: 2, Name: req.FullName, Email: req.Email},
			}, nil
		},
	}
	userSvc := &stubAuthUserService{
		getProfileFn: func(_ context.Context, _ uint) (*dto.UserProfileResponse, error) {
			return &dto.UserProfileResponse{ID: 1, IsAdmin: true}, nil
		},
	}

	e := newEcho()
	h := handler.NewAuthHandler(svc, userSvc)
	e.POST("/auth/register", func(c echo.Context) error {
		c.Set("userID", uint(1))
		return h.Register(c)
	})

	body := `{"full_name":"Bob","email":"bob@example.com","password":"Password1"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAuthHandler_Register_Returns409_OnDuplicate(t *testing.T) {
	svc := &stubAuthService{
		registerFn: func(_ context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error) {
			return nil, domain.ErrConflict
		},
	}
	adminUserSvc := &stubAuthUserService{
		getProfileFn: func(_ context.Context, _ uint) (*dto.UserProfileResponse, error) {
			return &dto.UserProfileResponse{ID: 1, IsAdmin: true}, nil
		},
	}

	e := newEcho()
	h := handler.NewAuthHandler(svc, adminUserSvc)
	e.POST("/auth/register", func(c echo.Context) error {
		c.Set("userID", uint(1))
		return h.Register(c)
	})

	body := `{"full_name":"Alice","email":"alice@example.com","password":"Password123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// T016: Logout handler and Refresh error-mapping tests
// ──────────────────────────────────────────────────────────────────────────────

func TestAuthHandler_Logout_Returns200_WithCookie(t *testing.T) {
	svc := &stubAuthService{}
	e := newEcho()
	h := handler.NewAuthHandler(svc, nil)
	e.POST("/auth/logout", h.Logout)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "some.refresh.token"})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	// Cookie should be cleared (MaxAge=-1)
	var cleared bool
	for _, c := range rec.Result().Cookies() {
		if c.Name == "refresh_token" && c.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Error("expected refresh_token cookie to be cleared with MaxAge<0")
	}
}

func TestAuthHandler_Logout_Returns200_MissingCookie(t *testing.T) {
	svc := &stubAuthService{}
	e := newEcho()
	h := handler.NewAuthHandler(svc, nil)
	e.POST("/auth/logout", h.Logout)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 even when cookie absent, got %d", rec.Code)
	}
}

func TestAuthHandler_Logout_Returns200_OnAlreadyRevokedToken(t *testing.T) {
	svc := &stubAuthService{
		logoutFn: func(_ context.Context, _ string) error {
			// Simulates already-revoked (idempotent) — service returns nil.
			return nil
		},
	}
	e := newEcho()
	h := handler.NewAuthHandler(svc, nil)
	e.POST("/auth/logout", h.Logout)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "already.revoked.token"})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 for already-revoked token, got %d", rec.Code)
	}
}

func TestAuthHandler_Refresh_Returns503_OnRevocationStoreError(t *testing.T) {
	svc := &stubAuthService{
		refreshFn: func(_ context.Context, _ string) (string, error) {
			return "", domain.ErrRevocationStoreUnavailable
		},
	}
	e := newEcho()
	h := handler.NewAuthHandler(svc, nil)
	e.POST("/auth/refresh", h.Refresh)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "some.token"})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAuthHandler_Refresh_Returns401_OnRevokedToken(t *testing.T) {
	svc := &stubAuthService{
		refreshFn: func(_ context.Context, _ string) (string, error) {
			return "", domain.ErrTokenRevoked
		},
	}
	e := newEcho()
	h := handler.NewAuthHandler(svc, nil)
	e.POST("/auth/refresh", h.Refresh)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "revoked.token"})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAuthHandler_Refresh_Returns401_OnDisabledAccount(t *testing.T) {
	svc := &stubAuthService{
		refreshFn: func(_ context.Context, _ string) (string, error) {
			return "", domain.ErrAccountDisabled
		},
	}
	e := newEcho()
	h := handler.NewAuthHandler(svc, nil)
	e.POST("/auth/refresh", h.Refresh)

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "disabled.user.token"})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}
