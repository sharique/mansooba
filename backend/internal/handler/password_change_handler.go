package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/service"
	"github.com/sharique/mansooba/pkg/logger"
	"go.uber.org/zap"
)

// PasswordChangeHandler exposes PUT /auth/me/password.
type PasswordChangeHandler struct {
	svc     service.PasswordChangeService
	authSvc service.AuthService
}

// NewPasswordChangeHandler returns a PasswordChangeHandler. authSvc is used
// only to reissue a fresh token pair for the acting session on success (see
// ChangePassword) — TokenValidAfter (ADR-032) invalidates every refresh
// token issued before the change, including the one the caller already
// holds, so that session needs a replacement to stay logged in per FR-008's
// "the session used to make the change remains active" requirement.
func NewPasswordChangeHandler(svc service.PasswordChangeService, authSvc service.AuthService) *PasswordChangeHandler {
	return &PasswordChangeHandler{svc: svc, authSvc: authSvc}
}

// ChangePassword godoc
// @Summary      Change the caller's own password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body dto.ChangePasswordRequest true "Current and new password"
// @Success      200 {object} map[string]string
// @Failure      400 {object} apierror.APIError "Validation error or incorrect current password"
// @Router       /auth/me/password [put]
func (h *PasswordChangeHandler) ChangePassword(c echo.Context) error {
	callerID := c.Get("userID").(uint)

	var req dto.ChangePasswordRequest
	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}
	if err := c.Validate(&req); err != nil {
		return echo.ErrBadRequest
	}

	if err := h.svc.ChangePassword(c.Request().Context(), callerID, req); err != nil {
		if errors.Is(err, domain.ErrCurrentPasswordMismatch) {
			return echo.NewHTTPError(http.StatusBadRequest, "current password is incorrect")
		}
		if errors.Is(err, domain.ErrNewPasswordSameAsCurrent) {
			return echo.NewHTTPError(http.StatusBadRequest, "new password must differ from current password")
		}
		return err
	}

	// The password change succeeded — that's final regardless of what
	// happens below. Reissuing the acting session's tokens is best-effort:
	// on failure the caller's current access token is still valid until its
	// natural TTL, it just won't survive its next refresh (same degradation
	// any other client would see, not a security regression).
	resp := map[string]string{"message": "password changed"}
	tokens, err := h.authSvc.IssueTokens(c.Request().Context(), callerID)
	if err != nil {
		logger.Logger.Warn("failed to reissue tokens after password change",
			zap.String("event", "password_change_token_reissue_failed"), zap.Error(err))
		return c.JSON(http.StatusOK, resp)
	}

	setRefreshCookie(c, tokens.RefreshToken)
	resp["access_token"] = tokens.AccessToken
	return c.JSON(http.StatusOK, resp)
}
