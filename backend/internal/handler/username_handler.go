package handler

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/sharique/mansooba/internal/service"
)

// UsernameHandler exposes the administrator-only username availability check
// used by the create-user forms (feature 018). It is administrator-only so that
// nobody else can probe which usernames exist.
type UsernameHandler struct {
	svc     service.UsernameService
	userSvc service.UserService
}

// NewUsernameHandler creates a UsernameHandler.
func NewUsernameHandler(svc service.UsernameService, userSvc service.UserService) *UsernameHandler {
	return &UsernameHandler{svc: svc, userSvc: userSvc}
}

// Check godoc
// @Summary      Check whether a username can be used (Admin only)
// @Description  Read-only. Reason is empty when available, else invalid, reserved or taken.
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        username query string true "Username to check"
// @Success      200 {object} dto.UsernameCheckResponse
// @Failure      400 {object} apierror.APIError "Missing username"
// @Failure      401 {object} apierror.APIError "Unauthorized"
// @Failure      403 {object} apierror.APIError "Forbidden"
// @Router       /admin/usernames/check [get]
func (h *UsernameHandler) Check(c *echo.Context) error {
	callerID := c.Get("userID").(uint)
	profile, err := h.userSvc.GetProfile(c.Request().Context(), callerID)
	if err != nil || !profile.IsAdmin {
		return echo.NewHTTPError(http.StatusForbidden, "forbidden")
	}
	username := c.QueryParam("username")
	if username == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "username is required")
	}
	return c.JSON(http.StatusOK, h.svc.Check(c.Request().Context(), username))
}
