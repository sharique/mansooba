package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sharique/mansooba/internal/service"
)

// ActivityHandler exposes the activity-log endpoint.
type ActivityHandler struct {
	svc service.ActivityService
}

func NewActivityHandler(svc service.ActivityService) *ActivityHandler {
	return &ActivityHandler{svc: svc}
}

// ListByIssue godoc
// @Summary      List activity events for an issue
// @Tags         activity
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Issue ID"
// @Success      200 {array} dto.ActivityEventResponse
// @Failure      401 {object} apierror.APIError
// @Failure      404 {object} apierror.APIError
// @Router       /issues/{id}/activity [get]
func (h *ActivityHandler) ListByIssue(c echo.Context) error {
	issueID, err := parseUintParam(c, "id")
	if err != nil {
		return echo.ErrBadRequest
	}
	events, err := h.svc.ListByIssue(c.Request().Context(), issueID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, events)
}
