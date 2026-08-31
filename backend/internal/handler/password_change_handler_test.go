package handler_test

// T002, T015: PasswordChangeHandler tests.
//
// newEcho() (with the password_complexity validator registered) is defined
// in auth_handler_test.go (same package).
//
// This file will not compile until handler.NewPasswordChangeHandler,
// handler.PasswordChangeHandler, service.PasswordChangeService,
// dto.ChangePasswordRequest, domain.ErrCurrentPasswordMismatch, and
// domain.ErrNewPasswordSameAsCurrent all exist — that is the expected TDD
// red state. See specs/012-change-password/IMPLEMENTATION_GUIDE.md.

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
)

// stubPasswordChangeService is a controllable stand-in for
// service.PasswordChangeService.
type stubPasswordChangeService struct {
	changePasswordFn func(ctx context.Context, userID uint, req dto.ChangePasswordRequest) error
	// lastCallerID captures the userID the handler actually passed through,
	// so tests can assert the caller identity always comes from the JWT
	// context, never from the request body.
	lastCallerID uint
}

func (s *stubPasswordChangeService) ChangePassword(ctx context.Context, userID uint, req dto.ChangePasswordRequest) error {
	s.lastCallerID = userID
	return s.changePasswordFn(ctx, userID, req)
}

// setupPasswordChangeRoute registers PUT /auth/me/password on a fresh echo
// instance (with the password_complexity validator registered), injecting
// userID into the context the same way the real JWTAuth middleware would —
// matching this package's established convention (see setupUserEcho in
// user_handler_test.go) of exercising the handler directly rather than the
// real middleware chain. Reuses stubAuthService (auth_handler_test.go, same
// package) with its default IssueTokens stub, since most of these tests
// don't care about token reissuance specifically.
func setupPasswordChangeRoute(svc *stubPasswordChangeService, userID uint) *echo.Echo {
	return setupPasswordChangeRouteWithAuth(svc, &stubAuthService{}, userID)
}

func setupPasswordChangeRouteWithAuth(svc *stubPasswordChangeService, authSvc *stubAuthService, userID uint) *echo.Echo {
	e := newEcho()
	h := handler.NewPasswordChangeHandler(svc, authSvc)
	e.PUT("/auth/me/password", func(c echo.Context) error {
		c.Set("userID", userID)
		return h.ChangePassword(c)
	})
	return e
}

func doChangePasswordRequest(e *echo.Echo, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPut, "/auth/me/password", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// ─── T002: happy path, malformed body, and domain-error → status mapping ─────

func TestPasswordChangeHandler_ValidRequest_Returns200(t *testing.T) {
	svc := &stubPasswordChangeService{
		changePasswordFn: func(_ context.Context, _ uint, _ dto.ChangePasswordRequest) error { return nil },
	}
	e := setupPasswordChangeRoute(svc, 1)

	rec := doChangePasswordRequest(e, `{"current_password":"OldPassword1","new_password":"NewPassword2"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["message"] != "password changed" {
		t.Errorf(`expected message "password changed", got %q`, resp["message"])
	}
	if svc.lastCallerID != 1 {
		t.Errorf("expected service called with callerID=1 (from context), got %d", svc.lastCallerID)
	}
}

// TestPasswordChangeHandler_Success_ReissuesTokensForActingSession guards
// against a real bug caught during manual verification (specs/012-change-password
// quickstart.md): TokenValidAfter invalidates every refresh token issued
// before the change, including the one the acting session already holds.
// Without reissuing fresh tokens here, the very session that changed the
// password would also get logged out on its next refresh — violating FR-008
// ("the session used to make the change remains active").
func TestPasswordChangeHandler_Success_ReissuesTokensForActingSession(t *testing.T) {
	svc := &stubPasswordChangeService{
		changePasswordFn: func(_ context.Context, _ uint, _ dto.ChangePasswordRequest) error { return nil },
	}
	var issuedForUserID uint
	authSvc := &stubAuthService{
		issueTokensFn: func(_ context.Context, userID uint) (*dto.AuthResponse, error) {
			issuedForUserID = userID
			return &dto.AuthResponse{AccessToken: "fresh.access.token", RefreshToken: "fresh.refresh.token"}, nil
		},
	}
	e := setupPasswordChangeRouteWithAuth(svc, authSvc, 1)

	rec := doChangePasswordRequest(e, `{"current_password":"OldPassword1","new_password":"NewPassword2"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if issuedForUserID != 1 {
		t.Errorf("expected IssueTokens called with the caller's own ID (1), got %d", issuedForUserID)
	}
	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["access_token"] != "fresh.access.token" {
		t.Errorf("expected access_token %q in response, got %q", "fresh.access.token", resp["access_token"])
	}

	var foundRefreshCookie bool
	for _, c := range rec.Result().Cookies() {
		if c.Name == "refresh_token" && c.Value == "fresh.refresh.token" {
			foundRefreshCookie = true
		}
	}
	if !foundRefreshCookie {
		t.Error("expected a refresh_token cookie carrying the freshly-issued refresh token")
	}
}

// TestPasswordChangeHandler_Success_TokenReissueFailure_StillReturns200
// confirms token reissuance is best-effort: the password change itself
// already succeeded and must not be reported as failed just because the
// reissuance step had a problem (mirrors the email-delivery-is-best-effort
// pattern already used elsewhere in this feature).
func TestPasswordChangeHandler_Success_TokenReissueFailure_StillReturns200(t *testing.T) {
	svc := &stubPasswordChangeService{
		changePasswordFn: func(_ context.Context, _ uint, _ dto.ChangePasswordRequest) error { return nil },
	}
	authSvc := &stubAuthService{
		issueTokensFn: func(_ context.Context, _ uint) (*dto.AuthResponse, error) {
			return nil, errors.New("token signing unavailable")
		},
	}
	e := setupPasswordChangeRouteWithAuth(svc, authSvc, 1)

	rec := doChangePasswordRequest(e, `{"current_password":"OldPassword1","new_password":"NewPassword2"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 even when token reissuance fails, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["message"] != "password changed" {
		t.Errorf(`expected message "password changed", got %q`, resp["message"])
	}
	if _, hasToken := resp["access_token"]; hasToken {
		t.Error("expected no access_token field when reissuance failed")
	}
}

func TestPasswordChangeHandler_MalformedBody_Returns400(t *testing.T) {
	svc := &stubPasswordChangeService{
		changePasswordFn: func(_ context.Context, _ uint, _ dto.ChangePasswordRequest) error {
			t.Fatal("service must not be called for a malformed body")
			return nil
		},
	}
	e := setupPasswordChangeRoute(svc, 1)

	rec := doChangePasswordRequest(e, `{"current_password": `) // truncated JSON

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for malformed body, got %d", rec.Code)
	}
}

func TestPasswordChangeHandler_MissingFields_Returns400(t *testing.T) {
	svc := &stubPasswordChangeService{
		changePasswordFn: func(_ context.Context, _ uint, _ dto.ChangePasswordRequest) error {
			t.Fatal("service must not be called when required fields are missing")
			return nil
		},
	}
	e := setupPasswordChangeRoute(svc, 1)

	rec := doChangePasswordRequest(e, `{"current_password":"OldPassword1"}`) // new_password absent

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing new_password, got %d", rec.Code)
	}
}

func TestPasswordChangeHandler_WrongCurrentPassword_Returns400WithGenericMessage(t *testing.T) {
	svc := &stubPasswordChangeService{
		changePasswordFn: func(_ context.Context, _ uint, _ dto.ChangePasswordRequest) error {
			return domain.ErrCurrentPasswordMismatch
		},
	}
	e := setupPasswordChangeRoute(svc, 1)

	rec := doChangePasswordRequest(e, `{"current_password":"WrongPassword9","new_password":"NewPassword2"}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["message"] != "current password is incorrect" {
		t.Errorf(`expected message "current password is incorrect", got %q`, resp["message"])
	}
}

func TestPasswordChangeHandler_NewPasswordSameAsCurrent_Returns400(t *testing.T) {
	svc := &stubPasswordChangeService{
		changePasswordFn: func(_ context.Context, _ uint, _ dto.ChangePasswordRequest) error {
			return domain.ErrNewPasswordSameAsCurrent
		},
	}
	e := setupPasswordChangeRoute(svc, 1)

	rec := doChangePasswordRequest(e, `{"current_password":"SamePassword1","new_password":"SamePassword1"}`)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["message"] != "new password must differ from current password" {
		t.Errorf(`expected message "new password must differ from current password", got %q`, resp["message"])
	}
}

// ─── T015: complexity-policy rejection and no-credential-reflection ──────────

func TestPasswordChangeHandler_WeakNewPassword_Returns400(t *testing.T) {
	svc := &stubPasswordChangeService{
		changePasswordFn: func(_ context.Context, _ uint, _ dto.ChangePasswordRequest) error {
			t.Fatal("service must not be called when new_password fails the complexity validator")
			return nil
		},
	}
	e := setupPasswordChangeRoute(svc, 1)

	// "short1" is 6 chars, no uppercase — fails password_complexity.
	rec := doChangePasswordRequest(e, `{"current_password":"OldPassword1","new_password":"short1"}`)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for a new_password failing the complexity policy, got %d", rec.Code)
	}
}

func TestPasswordChangeHandler_ResponsesNeverEchoSubmittedPasswords(t *testing.T) {
	cases := []struct {
		name string
		body string
		svc  *stubPasswordChangeService
	}{
		{
			name: "wrong current password",
			body: `{"current_password":"MySecretWrong1","new_password":"NewPassword2"}`,
			svc: &stubPasswordChangeService{
				changePasswordFn: func(_ context.Context, _ uint, _ dto.ChangePasswordRequest) error {
					return domain.ErrCurrentPasswordMismatch
				},
			},
		},
		{
			name: "weak new password",
			body: `{"current_password":"OldPassword1","new_password":"short1"}`,
			svc: &stubPasswordChangeService{
				changePasswordFn: func(_ context.Context, _ uint, _ dto.ChangePasswordRequest) error {
					t.Fatal("service must not be called for a weak new_password")
					return nil
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := setupPasswordChangeRoute(tc.svc, 1)
			rec := doChangePasswordRequest(e, tc.body)

			if strings.Contains(rec.Body.String(), "MySecretWrong1") ||
				strings.Contains(rec.Body.String(), "OldPassword1") ||
				strings.Contains(rec.Body.String(), "NewPassword2") ||
				strings.Contains(rec.Body.String(), "short1") {
				t.Errorf("response body must never echo a submitted password value, got: %s", rec.Body.String())
			}
		})
	}
}
