package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/service"
)

// PasswordResetHandler exposes the forgot-password and reset-password endpoints.
type PasswordResetHandler struct {
	svc service.PasswordResetService
}

// NewPasswordResetHandler creates a PasswordResetHandler.
func NewPasswordResetHandler(svc service.PasswordResetService) *PasswordResetHandler {
	return &PasswordResetHandler{svc: svc}
}

// ForgotPassword godoc
// @Summary      Request a password reset token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body dto.ForgotPasswordRequest true "Email address"
// @Success      200 {object} dto.ForgotPasswordResponse
// @Failure      400 {object} apierror.APIError "Bad request"
// @Router       /auth/forgot-password [post]
func (h *PasswordResetHandler) ForgotPassword(c echo.Context) error {
	var req dto.ForgotPasswordRequest
	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}
	if err := c.Validate(&req); err != nil {
		return echo.ErrBadRequest
	}

	resp, err := h.svc.ForgotPassword(c.Request().Context(), req)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

// ResetPassword godoc
// @Summary      Reset password using a reset token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body dto.ResetPasswordRequest true "Token and new password"
// @Success      200 {object} map[string]string
// @Failure      400 {object} apierror.APIError "Invalid or expired token / password too short"
// @Router       /auth/reset-password [post]
func (h *PasswordResetHandler) ResetPassword(c echo.Context) error {
	var req dto.ResetPasswordRequest
	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}
	if err := c.Validate(&req); err != nil {
		return echo.ErrBadRequest
	}

	err := h.svc.ResetPassword(c.Request().Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) || errors.Is(err, service.ErrTokenExpired) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid or expired token")
		}
		return err
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "password reset successful"})
}
