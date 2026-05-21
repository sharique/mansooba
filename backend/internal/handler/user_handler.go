package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/sharique/jira-go/internal/dto"
	"github.com/sharique/jira-go/internal/service"
)

// UserHandler exposes profile and my-activity endpoints for the current user.
type UserHandler struct {
	userSvc     service.UserService
	activitySvc service.ActivityService
}

// NewUserHandler creates a UserHandler backed by the given services.
func NewUserHandler(userSvc service.UserService, activitySvc service.ActivityService) *UserHandler {
	return &UserHandler{userSvc: userSvc, activitySvc: activitySvc}
}

// GetProfile godoc
// @Summary      Get current user profile
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} dto.UserProfileResponse
// @Failure      401 {object} apierror.APIError
// @Router       /auth/me [get]
func (h *UserHandler) GetProfile(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	resp, err := h.userSvc.GetProfile(c.Request().Context(), callerID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, resp)
}

// UpdateProfile godoc
// @Summary      Update current user profile
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body dto.UpdateProfileRequest true "Profile update payload"
// @Success      200 {object} dto.UserProfileResponse
// @Failure      400 {object} apierror.APIError
// @Failure      401 {object} apierror.APIError
// @Router       /auth/me [put]
func (h *UserHandler) UpdateProfile(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	var req dto.UpdateProfileRequest
	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}
	resp, err := h.userSvc.UpdateProfile(c.Request().Context(), callerID, req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusOK, resp)
}

// GetMyActivity godoc
// @Summary      Get activity feed for current user
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        limit  query int false "Max results (default 20, max 100)"
// @Param        offset query int false "Pagination offset"
// @Success      200 {array} dto.ActivityEventResponse
// @Failure      401 {object} apierror.APIError
// @Router       /auth/me/activity [get]
func (h *UserHandler) GetMyActivity(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	limit := 20
	offset := 0
	if v := c.QueryParam("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return echo.ErrBadRequest
		}
		limit = n
	}
	if v := c.QueryParam("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return echo.ErrBadRequest
		}
		offset = n
	}
	if limit > 100 {
		limit = 100
	}
	events, err := h.activitySvc.GetMyActivity(c.Request().Context(), callerID, limit, offset)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, events)
}
