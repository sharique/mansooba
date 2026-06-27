package handler_test

// newEcho() is defined in auth_handler_test.go (same package).

import (
	"context"
	"encoding/json"
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

// stubPasswordResetService is a controllable stand-in for service.PasswordResetService.
type stubPasswordResetService struct {
	forgotPasswordFn func(ctx context.Context, req dto.ForgotPasswordRequest) (*dto.ForgotPasswordResponse, error)
	resetPasswordFn  func(ctx context.Context, req dto.ResetPasswordRequest) error
}

func (s *stubPasswordResetService) ForgotPassword(ctx context.Context, req dto.ForgotPasswordRequest) (*dto.ForgotPasswordResponse, error) {
	return s.forgotPasswordFn(ctx, req)
}

func (s *stubPasswordResetService) ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) error {
	return s.resetPasswordFn(ctx, req)
}

func setupPasswordResetRoutes(e *echo.Echo, svc service.PasswordResetService) {
	h := handler.NewPasswordResetHandler(svc)
	e.POST("/auth/forgot-password", h.ForgotPassword)
	e.POST("/auth/reset-password", h.ResetPassword)
}

// valid64 is a well-formed 64-char hex token (all zeros — not a real token).
const valid64 = "0000000000000000000000000000000000000000000000000000000000000000"

// ─── ForgotPassword endpoint ─────────────────────────────────────────────────

func TestPasswordResetHandler_ForgotPassword_KnownEmail_Returns200WithToken(t *testing.T) {
	svc := &stubPasswordResetService{
		forgotPasswordFn: func(_ context.Context, _ dto.ForgotPasswordRequest) (*dto.ForgotPasswordResponse, error) {
			return &dto.ForgotPasswordResponse{
				Token:     valid64,
				ExpiresAt: "2026-01-01T00:15:00Z",
				Message:   "If that email is registered, a reset token has been generated.",
			}, nil
		},
	}
	e := newEcho()
	setupPasswordResetRoutes(e, svc)

	body := `{"email":"alice@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/forgot-password", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp dto.ForgotPasswordResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Token != valid64 {
		t.Errorf("expected token=%q, got %q", valid64, resp.Token)
	}
}

func TestPasswordResetHandler_ForgotPassword_UnknownEmail_Returns200WithEmptyToken(t *testing.T) {
	svc := &stubPasswordResetService{
		forgotPasswordFn: func(_ context.Context, _ dto.ForgotPasswordRequest) (*dto.ForgotPasswordResponse, error) {
			return &dto.ForgotPasswordResponse{
				Token:     "",
				ExpiresAt: "",
				Message:   "If that email is registered, a reset token has been generated.",
			}, nil
		},
	}
	e := newEcho()
	setupPasswordResetRoutes(e, svc)

	body := `{"email":"nobody@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/forgot-password", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 even for unknown email, got %d", rec.Code)
	}
	var resp dto.ForgotPasswordResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Token != "" {
		t.Error("token must be empty for unknown email")
	}
}

func TestPasswordResetHandler_ForgotPassword_MalformedJSON_Returns400(t *testing.T) {
	svc := &stubPasswordResetService{
		forgotPasswordFn: func(_ context.Context, _ dto.ForgotPasswordRequest) (*dto.ForgotPasswordResponse, error) {
			t.Error("service should not be called on malformed JSON")
			return nil, nil
		},
	}
	e := newEcho()
	setupPasswordResetRoutes(e, svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/forgot-password", strings.NewReader("{bad json"))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for malformed JSON, got %d", rec.Code)
	}
}

func TestPasswordResetHandler_ForgotPassword_MissingEmail_Returns400(t *testing.T) {
	svc := &stubPasswordResetService{
		forgotPasswordFn: func(_ context.Context, _ dto.ForgotPasswordRequest) (*dto.ForgotPasswordResponse, error) {
			t.Error("service should not be called when email is missing")
			return nil, nil
		},
	}
	e := newEcho()
	setupPasswordResetRoutes(e, svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/forgot-password", strings.NewReader(`{}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing email, got %d", rec.Code)
	}
}

// ─── ResetPassword endpoint ───────────────────────────────────────────────────

func TestPasswordResetHandler_ResetPassword_ValidToken_Returns200(t *testing.T) {
	svc := &stubPasswordResetService{
		resetPasswordFn: func(_ context.Context, _ dto.ResetPasswordRequest) error {
			return nil
		},
	}
	e := newEcho()
	setupPasswordResetRoutes(e, svc)

	body := `{"token":"` + valid64 + `","password":"newPassword8"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/reset-password", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPasswordResetHandler_ResetPassword_InvalidToken_Returns400(t *testing.T) {
	svc := &stubPasswordResetService{
		resetPasswordFn: func(_ context.Context, _ dto.ResetPasswordRequest) error {
			return domain.ErrNotFound
		},
	}
	e := newEcho()
	setupPasswordResetRoutes(e, svc)

	body := `{"token":"` + valid64 + `","password":"newPassword8"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/reset-password", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	var body2 map[string]string
	json.NewDecoder(rec.Body).Decode(&body2)
	if body2["message"] != "invalid or expired token" {
		t.Errorf("unexpected error message: %v", body2)
	}
}

func TestPasswordResetHandler_ResetPassword_ExpiredToken_Returns400(t *testing.T) {
	svc := &stubPasswordResetService{
		resetPasswordFn: func(_ context.Context, _ dto.ResetPasswordRequest) error {
			return service.ErrTokenExpired
		},
	}
	e := newEcho()
	setupPasswordResetRoutes(e, svc)

	body := `{"token":"` + valid64 + `","password":"newPassword8"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/reset-password", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["message"] != "invalid or expired token" {
		t.Errorf("unexpected error message: %v", resp)
	}
}

func TestPasswordResetHandler_ResetPassword_ShortPassword_Returns400(t *testing.T) {
	svc := &stubPasswordResetService{
		resetPasswordFn: func(_ context.Context, _ dto.ResetPasswordRequest) error {
			t.Error("service should not be called when password is too short")
			return nil
		},
	}
	e := newEcho()
	setupPasswordResetRoutes(e, svc)

	body := `{"token":"` + valid64 + `","password":"short"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/reset-password", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for short password, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPasswordResetHandler_ResetPassword_WrongTokenLength_Returns400(t *testing.T) {
	svc := &stubPasswordResetService{
		resetPasswordFn: func(_ context.Context, _ dto.ResetPasswordRequest) error {
			t.Error("service should not be called when token length is wrong")
			return nil
		},
	}
	e := newEcho()
	setupPasswordResetRoutes(e, svc)

	body := `{"token":"tooshort","password":"newPassword8"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/reset-password", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for wrong-length token, got %d: %s", rec.Code, rec.Body.String())
	}
}
