package handler_test

// T003: Integration test for the change-password endpoint against a real
// SQLite DB, real PasswordChangeService, real AuthService, real bcrypt
// hashing, and real AuthHandler.Login/Refresh calls — no mocks for this
// auth-critical path (Constitution Principle III).
//
// openAuthTestDB, seedUser, loginAndGetCookie, and newEcho are defined in
// auth_handler_integration_test.go / auth_handler_test.go (same package).

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sharique/mansooba/internal/handler"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/sharique/mansooba/internal/service"
	"github.com/sharique/mansooba/pkg/logger"
	"gorm.io/gorm"
)

// stubIntegrationEmailSender is a no-op EmailSender — this test exercises
// real DB/bcrypt/JWT behavior, not email delivery (covered separately by
// password_change_service_test.go and smtp_sender_test.go).
type stubIntegrationEmailSender struct{}

func (stubIntegrationEmailSender) SendPasswordReset(_ context.Context, _, _ string) error {
	return nil
}
func (stubIntegrationEmailSender) SendPasswordChanged(_ context.Context, _ string) error {
	return nil
}
func (stubIntegrationEmailSender) SendSuspiciousActivityAlert(_ context.Context, _ string) error {
	return nil
}

// testUserIDHeader carries the caller identity for the PUT route below,
// since there's no real JWTAuth middleware in this test setup — the route
// wrapper reads it and calls c.Set("userID", ...) the same way JWTAuth
// would, matching this package's established "inject userID directly"
// convention (see setupUserEcho in user_handler_test.go).
const testUserIDHeader = "X-Test-User-ID"

func newPasswordChangeIntegrationSetup(t *testing.T) (*echo.Echo, *handler.AuthHandler, *gorm.DB) {
	t.Helper()
	db := openAuthTestDB(t)
	log := logger.Logger
	userRepo := repository.NewUserRepository(db)
	revokedRepo := repository.NewRevokedTokenRepository(db)

	authSvc := service.NewAuthService(userRepo, revokedRepo, &stubSystemLogService{}, log, "test-secret-key", "15m", "168h")
	userSvc := service.NewUserService(userRepo)
	authHandler := handler.NewAuthHandler(authSvc, userSvc)

	pcSvc := service.NewPasswordChangeService(userRepo, &stubSystemLogService{}, stubIntegrationEmailSender{})
	pcHandler := handler.NewPasswordChangeHandler(pcSvc, authSvc)

	e := newEcho()
	e.POST("/auth/login", authHandler.Login)
	e.POST("/auth/refresh", authHandler.Refresh)
	e.PUT("/auth/me/password", func(c echo.Context) error {
		id, err := strconv.ParseUint(c.Request().Header.Get(testUserIDHeader), 10, 64)
		if err != nil {
			return echo.ErrUnauthorized
		}
		c.Set("userID", uint(id))
		return pcHandler.ChangePassword(c)
	})
	return e, authHandler, db
}

// doChangePasswordAs issues PUT /auth/me/password as the given user. If
// sessionCookie is non-nil it's attached to the request (simulating "this
// is the session that's making the change") purely so the response's
// Set-Cookie can be inspected by the caller — the handler itself never
// reads the incoming cookie, it only writes a new one.
func doChangePasswordAs(e *echo.Echo, userID uint, sessionCookie *http.Cookie, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPut, "/auth/me/password", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set(testUserIDHeader, strconv.FormatUint(uint64(userID), 10))
	if sessionCookie != nil {
		req.AddCookie(sessionCookie)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func doRefresh(e *echo.Echo, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func doLogin(e *echo.Echo, email, password string) int {
	body := `{"email":"` + email + `","password":"` + password + `"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec.Code
}

func TestPasswordChangeHandler_Integration_ChangeThenLogin(t *testing.T) {
	e, _, db := newPasswordChangeIntegrationSetup(t)
	user := seedUser(t, db, "alice@test.com", "OldPassword1")

	rec := doChangePasswordAs(e, user.ID, nil, `{"current_password":"OldPassword1","new_password":"NewPassword2"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from change-password, got %d: %s", rec.Code, rec.Body.String())
	}

	if code := doLogin(e, "alice@test.com", "OldPassword1"); code == http.StatusOK {
		t.Error("old password must no longer authenticate after a successful change")
	}
	if code := doLogin(e, "alice@test.com", "NewPassword2"); code != http.StatusOK {
		t.Errorf("new password must authenticate after a successful change, got status %d", code)
	}
}

func TestPasswordChangeHandler_Integration_WrongCurrentPassword_OldPasswordStillWorks(t *testing.T) {
	e, _, db := newPasswordChangeIntegrationSetup(t)
	user := seedUser(t, db, "bob@test.com", "OldPassword1")

	rec := doChangePasswordAs(e, user.ID, nil, `{"current_password":"WrongPassword9","new_password":"NewPassword2"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for wrong current password, got %d: %s", rec.Code, rec.Body.String())
	}

	if code := doLogin(e, "bob@test.com", "OldPassword1"); code != http.StatusOK {
		t.Errorf("original password must still authenticate after a rejected attempt, got status %d", code)
	}
}

// TestPasswordChangeHandler_Integration_OtherSessionRevoked_ActingSessionSurvives
// is the real-infrastructure regression test for a bug caught during manual
// verification (specs/012-change-password/quickstart.md §2): two sessions
// are logged in, the password is changed from session A, and then — against
// the real AuthService, real JWT signing/parsing, and a real DB-backed
// TokenValidAfter check — session B's refresh must fail while session A's
// (reissued) refresh must still succeed.
func TestPasswordChangeHandler_Integration_OtherSessionRevoked_ActingSessionSurvives(t *testing.T) {
	e, authHandler, db := newPasswordChangeIntegrationSetup(t)
	user := seedUser(t, db, "carol@test.com", "OldPassword1")

	cookieA := loginAndGetCookie(t, e, authHandler, "carol@test.com", "OldPassword1")
	cookieB := loginAndGetCookie(t, e, authHandler, "carol@test.com", "OldPassword1")

	// TokenValidAfter is truncated to whole seconds (password_change_service.go)
	// to match jwt.NewNumericDate's own truncation. In real usage a password
	// change happens well over a second after any prior login, so this is a
	// non-issue in production — but this test logs both sessions in and
	// changes the password within the same test function, so without this
	// sleep session B's login could truncate into the *same* second as the
	// change and incorrectly appear to survive it (a real, accepted, narrow
	// precision limitation of the design — see ADR-032 — not a test bug to
	// paper over by loosening the assertion below).
	time.Sleep(1100 * time.Millisecond)

	rec := doChangePasswordAs(e, user.ID, cookieA, `{"current_password":"OldPassword1","new_password":"NewPassword2"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from change-password, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["access_token"] == "" {
		t.Fatal("expected a non-empty access_token in the change-password response")
	}
	var reissuedCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "refresh_token" {
			reissuedCookie = c
		}
	}
	if reissuedCookie == nil {
		t.Fatal("expected a refresh_token cookie on the change-password response")
	}

	if code := doRefresh(e, cookieB).Code; code != http.StatusUnauthorized {
		t.Errorf("session B (not the one that changed the password) must be rejected on refresh, got %d", code)
	}

	if code := doRefresh(e, reissuedCookie).Code; code != http.StatusOK {
		t.Errorf("session A's reissued refresh token (from the change-password response) must still work, got %d", code)
	}
}
