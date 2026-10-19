package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/service"
)

// OverviewHandler exposes the read-only project overview: sprint progress, open
// tasks and recent activity. Every route is JWT-protected and member-only; the
// service enforces membership.
type OverviewHandler struct {
	svc service.OverviewService
}

// NewOverviewHandler creates an OverviewHandler backed by the given service.
func NewOverviewHandler(svc service.OverviewService) *OverviewHandler {
	return &OverviewHandler{svc: svc}
}

// Sprint godoc
// @Summary      Progress of the project's active sprint
// @Description  Task and story-point counts in the four board columns, the sprint's dates and whether it is overdue. "sprint" is null when no sprint is active.
// @Tags         overview
// @Produce      json
// @Security     BearerAuth
// @Param        key path string true "Project key"
// @Success      200 {object} dto.OverviewSprintResponse
// @Failure      401 {object} apierror.APIError "Unauthorized"
// @Failure      403 {object} apierror.APIError "Not a member of the project"
// @Failure      404 {object} apierror.APIError "Not found"
// @Router       /projects/{key}/overview/sprint [get]
func (h *OverviewHandler) Sprint(c *echo.Context) error {
	resp, err := h.svc.Sprint(c.Request().Context(), c.Param("key"), c.Get("userID").(uint))
	if err != nil {
		return mapOverviewError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// Tasks godoc
// @Summary      Open tasks of the project's active sprint
// @Description  Up to 10 tasks that are not done, ordered by priority, then status (in review, in progress, to do), then most recently updated.
// @Tags         overview
// @Produce      json
// @Security     BearerAuth
// @Param        key path string true "Project key"
// @Success      200 {object} dto.OverviewTasksResponse
// @Failure      401 {object} apierror.APIError "Unauthorized"
// @Failure      403 {object} apierror.APIError "Not a member of the project"
// @Failure      404 {object} apierror.APIError "Not found"
// @Router       /projects/{key}/overview/tasks [get]
func (h *OverviewHandler) Tasks(c *echo.Context) error {
	resp, err := h.svc.Tasks(c.Request().Context(), c.Param("key"), c.Get("userID").(uint))
	if err != nil {
		return mapOverviewError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// Activity godoc
// @Summary      Recent progress and comments in the project
// @Description  Up to 10 events from the last 5 days, newest first: tasks moved into In Progress, In Review or Done, and added comments (never the comment text). scope=sprint limits it to tasks in the active sprint.
// @Tags         overview
// @Produce      json
// @Security     BearerAuth
// @Param        key path string true "Project key"
// @Param        scope query string false "all (default) or sprint" Enums(all, sprint)
// @Success      200 {object} dto.OverviewActivityResponse
// @Failure      400 {object} apierror.APIError "Invalid scope"
// @Failure      401 {object} apierror.APIError "Unauthorized"
// @Failure      403 {object} apierror.APIError "Not a member of the project"
// @Failure      404 {object} apierror.APIError "Not found"
// @Router       /projects/{key}/overview/activity [get]
func (h *OverviewHandler) Activity(c *echo.Context) error {
	resp, err := h.svc.Activity(c.Request().Context(), c.Param("key"), c.Get("userID").(uint), c.QueryParam("scope"))
	if err != nil {
		return mapOverviewError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func mapOverviewError(err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return echo.NewHTTPError(http.StatusNotFound, "not found")
	case errors.Is(err, domain.ErrForbidden):
		return echo.NewHTTPError(http.StatusForbidden, "forbidden")
	case errors.Is(err, domain.ErrInvalidInput):
		return echo.NewHTTPError(http.StatusBadRequest, "scope must be all or sprint")
	}
	return err
}
