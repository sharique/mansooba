package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/service"
)

// SettingHandler exposes endpoints for reading and updating global platform settings.
type SettingHandler struct {
	svc     service.SettingService
	userSvc service.UserService
}

// NewSettingHandler creates a SettingHandler backed by the given services.
func NewSettingHandler(svc service.SettingService, userSvc service.UserService) *SettingHandler {
	return &SettingHandler{svc: svc, userSvc: userSvc}
}

// GetAll godoc
// @Summary      Get global settings
// @Tags         settings
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} dto.SettingsResponse
// @Failure      401 {object} apierror.APIError
// @Router       /settings [get]
func (h *SettingHandler) GetAll(c echo.Context) error {
	resp, err := h.svc.GetAll(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

// Patch godoc
// @Summary      Update global settings (Admin only)
// @Tags         settings
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body dto.PatchSettingsRequest true "Settings to update"
// @Success      200 {object} dto.SettingsResponse
// @Failure      400 {object} apierror.APIError
// @Failure      401 {object} apierror.APIError
// @Failure      403 {object} apierror.APIError
// @Router       /settings [patch]
func (h *SettingHandler) Patch(c echo.Context) error {
	callerID := c.Get("userID").(uint)

	profile, err := h.userSvc.GetProfile(c.Request().Context(), callerID)
	if err != nil {
		return err
	}
	if !profile.IsAdmin {
		return echo.NewHTTPError(http.StatusForbidden, "forbidden")
	}

	var req dto.PatchSettingsRequest
	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}

	resp, err := h.svc.Patch(c.Request().Context(), callerID, req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidSettingValue) {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		return err
	}
	return c.JSON(http.StatusOK, resp)
}
