package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sharique/jira-go/internal/domain"
	"github.com/sharique/jira-go/internal/dto"
	"github.com/sharique/jira-go/internal/service"
)

// AuthHandler exposes register, login, and refresh endpoints.
type AuthHandler struct {
	svc service.AuthService
}

// NewAuthHandler creates an AuthHandler backed by the given service.
func NewAuthHandler(svc service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Register(c echo.Context) error {
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
