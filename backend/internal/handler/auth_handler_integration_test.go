package handler_test

// T007: Integration tests verifying the refresh token cookie fix.
// Uses a real SQLite DB, real authService, and real JWTs — no mocks for this path
// (Constitution Principle III: security paths require real infrastructure).

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/handler"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/sharique/mansooba/internal/service"
	"github.com/sharique/mansooba/pkg/logger"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	if err := db.AutoMigrate(&domain.User{}, &domain.RevokedToken{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func newIntegrationSetup(t *testing.T) (*echo.Echo, *handler.AuthHandler, *gorm.DB) {
	t.Helper()
	db := openAuthTestDB(t)
	log := logger.Logger
	userRepo := repository.NewUserRepository(db)
	revokedRepo := repository.NewRevokedTokenRepository(db)
	authSvc := service.NewAuthService(userRepo, revokedRepo, log, "test-secret-key", "15m", "168h")
	userSvc := service.NewUserService(userRepo)
	h := handler.NewAuthHandler(authSvc, userSvc)
	e := newEcho()
	return e, h, db
}

func seedUser(t *testing.T, db *gorm.DB, email, password string) *domain.User {
	t.Helper()
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	user := &domain.User{Name: "Test User", Email: email, Password: string(hash), IsActive: true}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return user
}

func loginAndGetCookie(t *testing.T, e *echo.Echo, h *handler.AuthHandler, email, password string) *http.Cookie {
	t.Helper()
	body := `{"email":"` + email + `","password":"` + password + `"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.POST("/auth/login", h.Login)
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login failed: %d %s", rec.Code, rec.Body.String())
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == "refresh_token" {
			return c
		}
	}
	t.Fatal("no refresh_token cookie in login response")
	return nil
}

func TestLogin_SetsNonEmptyRefreshCookie(t *testing.T) {
	e, h, db := newIntegrationSetup(t)
	seedUser(t, db, "alice@test.com", "Password1")

	body := `{"email":"alice@test.com","password":"Password1"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.POST("/auth/login", h.Login)
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var found bool
	for _, c := range rec.Result().Cookies() {
		if c.Name == "refresh_token" {
			if c.Value == "" {
				t.Fatal("refresh_token cookie is empty")
			}
			found = true
		}
	}
	if !found {
		t.Fatal("refresh_token cookie not set")
	}
}

func TestLogin_RefreshCookieHasJTI(t *testing.T) {
	e, h, db := newIntegrationSetup(t)
	seedUser(t, db, "bob@test.com", "Password1")

	cookie := loginAndGetCookie(t, e, h, "bob@test.com", "Password1")

	claims := &jwt.RegisteredClaims{}
	_, err := jwt.ParseWithClaims(cookie.Value, claims, func(token *jwt.Token) (any, error) {
		return []byte("test-secret-key"), nil
	})
	if err != nil {
		t.Fatalf("parse refresh token: %v", err)
	}
	if claims.ID == "" {
		t.Error("refresh token has no JTI (claims.ID is empty)")
	}
}

func TestRefresh_SucceedsWithCookieFromLogin(t *testing.T) {
	e, h, db := newIntegrationSetup(t)
	seedUser(t, db, "carol@test.com", "Password1")

	cookie := loginAndGetCookie(t, e, h, "carol@test.com", "Password1")

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	e.POST("/auth/refresh", h.Refresh)
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp) //nolint:errcheck
	if resp["access_token"] == "" {
		t.Error("expected non-empty access_token in refresh response")
	}
}

func TestRefresh_Returns401_AfterLogout(t *testing.T) {
	e, h, db := newIntegrationSetup(t)
	seedUser(t, db, "dave@test.com", "Password1")

	cookie := loginAndGetCookie(t, e, h, "dave@test.com", "Password1")

	// Logout
	logoutReq := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	logoutReq.AddCookie(cookie)
	logoutRec := httptest.NewRecorder()
	e.POST("/auth/logout", h.Logout)
	e.POST("/auth/refresh", h.Refresh)
	e.ServeHTTP(logoutRec, logoutReq)
	if logoutRec.Code != http.StatusOK {
		t.Fatalf("logout failed: %d", logoutRec.Code)
	}

	// Attempt refresh with revoked token
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after logout, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLogin_ResponseBodyContainsRefreshToken(t *testing.T) {
	e, h, db := newIntegrationSetup(t)
	seedUser(t, db, "eve@test.com", "Password1")

	body := `{"email":"eve@test.com","password":"Password1"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.POST("/auth/login", h.Login)
	e.ServeHTTP(rec, req)

	var resp dto.AuthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.RefreshToken == "" {
		t.Error("expected refresh_token in response body")
	}
}
