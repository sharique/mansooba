package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/service"
)

// AuthHandler exposes register, login, and refresh endpoints.
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

	h.setRefreshCookie(c, "")
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
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}

	h.setRefreshCookie(c, "")
	return c.JSON(http.StatusOK, resp)
}

// Refresh godoc
// @Summary      Refresh access token using refresh_token cookie
// @Tags         auth
// @Produce      json
// @Success      200 {object} map[string]string
// @Failure      401 {object} apierror.APIError "Unauthorized"
// @Router       /auth/refresh [post]
func (h *AuthHandler) Refresh(c echo.Context) error {
	cookie, err := c.Cookie("refresh_token")
	if err != nil {
		return echo.ErrUnauthorized
	}

	accessToken, err := h.svc.Refresh(c.Request().Context(), cookie.Value)
	if err != nil {
		return echo.ErrUnauthorized
	}

	return c.JSON(http.StatusOK, map[string]string{"access_token": accessToken})
}

func (h *AuthHandler) setRefreshCookie(c echo.Context, value string) {
	c.SetCookie(&http.Cookie{
		Name:     "refresh_token",
		Value:    value,
		HttpOnly: true,
		Path:     "/",
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
	})
}
