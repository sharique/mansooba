package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/service"
)

// AdminUserHandler exposes admin-only user management endpoints.
type AdminUserHandler struct {
	svc     service.AdminUserService
	userSvc service.UserService
}

// NewAdminUserHandler creates an AdminUserHandler.
func NewAdminUserHandler(svc service.AdminUserService, userSvc service.UserService) *AdminUserHandler {
	return &AdminUserHandler{svc: svc, userSvc: userSvc}
}

// ListUsers godoc
// @Summary      List all users (admin only)
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        page query int false "Page (1-based)" default(1)
// @Param        size query int false "Page size (max 100)" default(20)
// @Success      200 {object} dto.AdminUserListResponse
// @Failure      400 {object} apierror.APIError "Invalid query params"
// @Failure      403 {object} apierror.APIError "Forbidden"
// @Router       /admin/users [get]
func (h *AdminUserHandler) ListUsers(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	profile, err := h.userSvc.GetProfile(c.Request().Context(), callerID)
	if err != nil || !profile.IsAdmin {
		return echo.NewHTTPError(http.StatusForbidden, "forbidden")
	}

	page, size, err := parsePagination(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	resp, err := h.svc.ListUsers(c.Request().Context(), page, size)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, resp)
}

// PatchUser godoc
// @Summary      Update user role or status (admin only)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "User ID"
// @Param        body body dto.AdminUserPatchRequest true "Fields to update"
// @Success      200 {object} dto.AdminUserDTO
// @Failure      400 {object} apierror.APIError "Invalid body or params"
// @Failure      403 {object} apierror.APIError "Forbidden"
// @Failure      404 {object} apierror.APIError "User not found"
// @Failure      409 {object} apierror.APIError "Cannot remove last admin"
// @Router       /admin/users/{id} [patch]
func (h *AdminUserHandler) PatchUser(c echo.Context) error {
	// Step 1: caller must be admin.
	callerID := c.Get("userID").(uint)
	profile, err := h.userSvc.GetProfile(c.Request().Context(), callerID)
	if err != nil || !profile.IsAdmin {
		return echo.NewHTTPError(http.StatusForbidden, "forbidden")
	}

	// Step 2: parse target ID.
	targetID, err := parseUintParam(c, "id")
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}

	// Step 3: bind and validate body.
	var req dto.AdminUserPatchRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.IsAdmin == nil && req.IsActive == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "at least one field (is_admin, is_active) is required")
	}

	ctx := c.Request().Context()

	// Step 4: apply changes (last-admin guard inside service).
	if req.IsAdmin != nil {
		if err := h.svc.SetRole(ctx, callerID, targetID, *req.IsAdmin); err != nil {
			if errors.Is(err, domain.ErrLastAdmin) {
				return echo.NewHTTPError(http.StatusConflict, map[string]string{
					"code":    "LAST_ADMIN",
					"message": "Cannot remove the last active admin. Promote another user first.",
				})
			}
			if errors.Is(err, domain.ErrNotFound) {
				return echo.NewHTTPError(http.StatusNotFound, "user not found")
			}
			return err
		}
	}
	if req.IsActive != nil {
		if err := h.svc.SetActive(ctx, callerID, targetID, *req.IsActive); err != nil {
			if errors.Is(err, domain.ErrLastAdmin) {
				return echo.NewHTTPError(http.StatusConflict, map[string]string{
					"code":    "LAST_ADMIN",
					"message": "Cannot remove the last active admin. Promote another user first.",
				})
			}
			if errors.Is(err, domain.ErrNotFound) {
				return echo.NewHTTPError(http.StatusNotFound, "user not found")
			}
			return err
		}
	}

	// Step 5: return updated user.
	updated, err := h.svc.GetUser(ctx, targetID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return err
	}
	return c.JSON(http.StatusOK, updated)
}

func parsePagination(c echo.Context) (page, size int, err error) {
	page = 1
	size = 20

	if p := c.QueryParam("page"); p != "" {
		if page, err = strconv.Atoi(p); err != nil || page < 1 {
			return 0, 0, errors.New("page must be a positive integer")
		}
	}
	if s := c.QueryParam("size"); s != "" {
		if size, err = strconv.Atoi(s); err != nil || size < 1 || size > 100 {
			return 0, 0, errors.New("size must be between 1 and 100")
		}
	}
	return page, size, nil
}

