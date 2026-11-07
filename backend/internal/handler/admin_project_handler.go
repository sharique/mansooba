package handler

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/sharique/mansooba/internal/service"
)

// AdminProjectHandler lets a system administrator find and take over projects
// that have no active admin. Everything else about a project stays hidden from
// them until they have done so (FR-007, FR-007d).
type AdminProjectHandler struct {
	svc service.ProjectService
}

// NewAdminProjectHandler creates an AdminProjectHandler.
func NewAdminProjectHandler(svc service.ProjectService) *AdminProjectHandler {
	return &AdminProjectHandler{svc: svc}
}

// ListWithoutAdmin godoc
// @Summary      List projects that have no active admin (system administrators only)
// @Description  Key, name and number of members only; nothing about the project's contents.
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array}  dto.AbandonedProjectResponse
// @Failure      401 {object} apierror.APIError "Unauthorized"
// @Failure      403 {object} apierror.APIError "Forbidden"
// @Router       /admin/projects/without-admin [get]
func (h *AdminProjectHandler) ListWithoutAdmin(c *echo.Context) error {
	callerID := c.Get("userID").(uint)
	resp, err := h.svc.ListWithoutAdmin(c.Request().Context(), callerID)
	if err != nil {
		return mapProjectError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ClaimAdmin godoc
// @Summary      Take over a project that has no active admin (system administrators only)
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        key path string true "Project key"
// @Success      200 {object} dto.MemberResponse
// @Failure      401 {object} apierror.APIError "Unauthorized"
// @Failure      403 {object} apierror.APIError "Forbidden"
// @Failure      404 {object} apierror.APIError "Not found"
// @Failure      409 {object} apierror.APIError "The project still has an active admin"
// @Router       /admin/projects/{key}/claim-admin [post]
func (h *AdminProjectHandler) ClaimAdmin(c *echo.Context) error {
	callerID := c.Get("userID").(uint)
	resp, err := h.svc.ClaimAdmin(c.Request().Context(), c.Param("key"), callerID)
	if err != nil {
		return mapProjectError(err)
	}
	return c.JSON(http.StatusOK, resp)
}
