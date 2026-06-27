package handler

import (
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/service"
)

// AuthHandler exposes register, login, refresh, and logout endpoints.
type AuthHandler struct {
	svc     service.AuthService
	userSvc service.UserService
}

// NewAuthHandler creates an AuthHandler backed by the given services.
func NewAuthHandler(svc service.AuthService, userSvc service.UserService) *AuthHandler {
	return &AuthHandler{svc: svc, userSvc: userSvc}
}

// Register godoc
// @Summary      Register a new user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body dto.RegisterRequest true "Registration payload"
// @Success      201 {object} dto.AuthResponse
// @Failure      400 {object} apierror.APIError "Bad request"
// @Failure      409 {object} apierror.APIError "Email already registered"
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	profile, err := h.userSvc.GetProfile(c.Request().Context(), callerID)
	if err != nil || !profile.IsAdmin {
		return echo.NewHTTPError(http.StatusForbidden, "forbidden")
	}

	var req dto.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}

	resp, err := h.svc.Register(c.Request().Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return echo.NewHTTPError(http.StatusConflict, "email already registered")
		}
		return err
	}

	// Register is an admin-only action — do NOT set the refresh cookie.
	// Setting it would overwrite the calling admin's own session cookie.
	return c.JSON(http.StatusCreated, resp)
}

// Login godoc
// @Summary      Authenticate a user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body dto.LoginRequest true "Login payload"
// @Success      200 {object} dto.AuthResponse
// @Failure      400 {object} apierror.APIError "Bad request"
// @Failure      401 {object} apierror.APIError "Invalid credentials"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c echo.Context) error {
	var req dto.LoginRequest
	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}

	resp, err := h.svc.Login(c.Request().Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrAccountDisabled) {
			return echo.NewHTTPError(http.StatusUnauthorized, "account disabled")
		}
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}

	h.setRefreshCookie(c, resp.RefreshToken)
	return c.JSON(http.StatusOK, resp)
}

// Refresh godoc
// @Summary      Refresh access token using refresh_token cookie
// @Tags         auth
// @Produce      json
// @Success      200 {object} map[string]string
// @Failure      401 {object} apierror.APIError "Unauthorized"
// @Failure      503 {object} apierror.APIError "Revocation store unavailable"
// @Router       /auth/refresh [post]
func (h *AuthHandler) Refresh(c echo.Context) error {
	cookie, err := c.Cookie("refresh_token")
	if err != nil {
		return echo.ErrUnauthorized
	}

	accessToken, err := h.svc.Refresh(c.Request().Context(), cookie.Value)
	if err != nil {
		if errors.Is(err, domain.ErrRevocationStoreUnavailable) {
			return echo.NewHTTPError(http.StatusServiceUnavailable, "REVOCATION_STORE_UNAVAILABLE")
		}
		return echo.ErrUnauthorized
	}

	return c.JSON(http.StatusOK, map[string]string{"access_token": accessToken})
}

// Logout godoc
// @Summary      Revoke the refresh token and clear its cookie
// @Description  Reads the refresh_token cookie. Missing or invalid tokens are ignored — logout is always idempotent (200). Rate-limited per ADR-021.
// @Tags         auth
// @Success      200 "Cookie cleared"
// @Failure      500 {object} apierror.APIError "Unexpected server error"
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c echo.Context) error {
	cookie, err := c.Cookie("refresh_token")
	if err != nil || cookie.Value == "" {
		// Nothing to revoke — clear cookie defensively and succeed.
		h.clearRefreshCookie(c)
		return c.NoContent(http.StatusOK)
	}

	// Logout is idempotent — ignore revocation errors from the service.
	_ = h.svc.Logout(c.Request().Context(), cookie.Value)

	h.clearRefreshCookie(c)
	return c.NoContent(http.StatusOK)
}

func (h *AuthHandler) setRefreshCookie(c echo.Context, value string) {
	secure := os.Getenv("APP_ENV") != "development"
	c.SetCookie(&http.Cookie{
		Name:     "refresh_token",
		Value:    value,
		HttpOnly: true,
		Secure:   secure,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
	})
}

func (h *AuthHandler) clearRefreshCookie(c echo.Context) {
	secure := os.Getenv("APP_ENV") != "development"
	c.SetCookie(&http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		HttpOnly: true,
		Secure:   secure,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})
}
