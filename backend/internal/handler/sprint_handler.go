package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/sharique/jira-go/internal/domain"
	"github.com/sharique/jira-go/internal/dto"
	"github.com/sharique/jira-go/internal/service"
)

type SprintHandler struct {
	svc service.SprintService
}

func NewSprintHandler(svc service.SprintService) *SprintHandler {
	return &SprintHandler{svc: svc}
}

// List godoc
// @Summary      List sprints
// @Description  Returns all sprints for a project ordered by created_at ASC.
// @Tags         sprints
// @Produce      json
// @Param        key path string true "Project key (e.g. PROJ)"
// @Success      200 {array}  dto.SprintResponse
// @Failure      404 {object} apierror.APIError "Project not found"
// @Failure      401 {object} apierror.APIError "Unauthorized"
// @Router       /projects/{key}/sprints [get]
// @Security     BearerAuth
func (h *SprintHandler) List(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	sprints, err := h.svc.List(c.Request().Context(), c.Param("key"), callerID)
	if err != nil {
		return mapSprintError(err)
	}
	return c.JSON(http.StatusOK, sprints)
}

// Create godoc
// @Summary      Create a sprint
// @Description  Creates a new sprint in Planning state. Requires admin or owner role.
// @Tags         sprints
// @Accept       json
// @Produce      json
// @Param        key  path string                  true "Project key"
// @Param        body body dto.CreateSprintRequest true "Sprint details"
// @Success      201 {object} dto.SprintResponse
// @Failure      400 {object} apierror.APIError "Validation error"
// @Failure      403 {object} apierror.APIError "Insufficient role"
// @Failure      404 {object} apierror.APIError "Project not found"
// @Router       /projects/{key}/sprints [post]
// @Security     BearerAuth
func (h *SprintHandler) Create(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	var req dto.CreateSprintRequest
	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}
	sprint, err := h.svc.Create(c.Request().Context(), c.Param("key"), callerID, req)
	if err != nil {
		return mapSprintError(err)
	}
	return c.JSON(http.StatusCreated, sprint)
}

// Get godoc
// @Summary      Get a sprint
// @Description  Returns a single sprint by ID.
// @Tags         sprints
// @Produce      json
// @Param        key path string true "Project key"
// @Param        id  path int    true "Sprint ID"
// @Success      200 {object} dto.SprintResponse
// @Failure      404 {object} apierror.APIError "Sprint or project not found"
// @Failure      401 {object} apierror.APIError "Unauthorized"
// @Router       /projects/{key}/sprints/{id} [get]
// @Security     BearerAuth
func (h *SprintHandler) Get(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	id, err := parseSprintID(c)
	if err != nil {
		return echo.ErrBadRequest
	}
	sprint, err := h.svc.Get(c.Request().Context(), c.Param("key"), id, callerID)
	if err != nil {
		return mapSprintError(err)
	}
	return c.JSON(http.StatusOK, sprint)
}

// Update godoc
// @Summary      Update a sprint
// @Description  Updates name, goal, or dates. Blocked for Completed sprints. Requires admin or owner role.
// @Tags         sprints
// @Accept       json
// @Produce      json
// @Param        key  path string                  true  "Project key"
// @Param        id   path int                     true  "Sprint ID"
// @Param        body body dto.UpdateSprintRequest true  "Fields to update (all optional)"
// @Success      200 {object} dto.SprintResponse
// @Failure      409 {object} apierror.APIError "Sprint is Completed (not editable)"
// @Failure      403 {object} apierror.APIError "Insufficient role"
// @Failure      404 {object} apierror.APIError "Sprint or project not found"
// @Router       /projects/{key}/sprints/{id} [put]
// @Security     BearerAuth
func (h *SprintHandler) Update(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	id, err := parseSprintID(c)
	if err != nil {
		return echo.ErrBadRequest
	}
	var req dto.UpdateSprintRequest
	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}
	sprint, err := h.svc.Update(c.Request().Context(), c.Param("key"), id, callerID, req)
	if err != nil {
		return mapSprintError(err)
	}
	return c.JSON(http.StatusOK, sprint)
}

// Delete godoc
// @Summary      Delete a sprint
// @Description  Removes a sprint. Only allowed when status is Planning. Requires admin or owner role.
// @Tags         sprints
// @Param        key path string true "Project key"
// @Param        id  path int    true "Sprint ID"
// @Success      204
// @Failure      409 {object} apierror.APIError "Sprint is not in Planning state"
// @Failure      403 {object} apierror.APIError "Insufficient role"
// @Failure      404 {object} apierror.APIError "Sprint or project not found"
// @Router       /projects/{key}/sprints/{id} [delete]
// @Security     BearerAuth
func (h *SprintHandler) Delete(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	id, err := parseSprintID(c)
	if err != nil {
		return echo.ErrBadRequest
	}
	if err := h.svc.Delete(c.Request().Context(), c.Param("key"), id, callerID); err != nil {
		return mapSprintError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// Start godoc
// @Summary      Start a sprint
// @Description  Transitions a sprint from Planning to Active. Only one Active sprint is allowed per project. Requires admin or owner role.
// @Tags         sprints
// @Produce      json
// @Param        key path string true "Project key"
// @Param        id  path int    true "Sprint ID"
// @Success      200 {object} dto.SprintResponse
// @Failure      409 {object} apierror.APIError "Sprint already active or invalid state"
// @Failure      403 {object} apierror.APIError "Insufficient role"
// @Failure      404 {object} apierror.APIError "Sprint or project not found"
// @Router       /projects/{key}/sprints/{id}/start [post]
// @Security     BearerAuth
func (h *SprintHandler) Start(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	id, err := parseSprintID(c)
	if err != nil {
		return echo.ErrBadRequest
	}
	sprint, err := h.svc.Start(c.Request().Context(), c.Param("key"), id, callerID)
	if err != nil {
		return mapSprintError(err)
	}
	return c.JSON(http.StatusOK, sprint)
}

// Complete godoc
// @Summary      Complete a sprint
// @Description  Transitions an Active sprint to Completed. Unfinished issues move to next_sprint_id or the backlog if omitted. Requires admin or owner role.
// @Tags         sprints
// @Accept       json
// @Produce      json
// @Param        key  path string                    true  "Project key"
// @Param        id   path int                       true  "Sprint ID"
// @Param        body body dto.CompleteSprintRequest false "Optional: next sprint for unfinished issues"
// @Success      200 {object} dto.SprintResponse
// @Failure      409 {object} apierror.APIError "Sprint is not Active"
// @Failure      404 {object} apierror.APIError "next_sprint_id not found"
// @Failure      403 {object} apierror.APIError "Insufficient role"
// @Router       /projects/{key}/sprints/{id}/complete [post]
// @Security     BearerAuth
func (h *SprintHandler) Complete(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	id, err := parseSprintID(c)
	if err != nil {
		return echo.ErrBadRequest
	}
	var req dto.CompleteSprintRequest
	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}
	sprint, err := h.svc.Complete(c.Request().Context(), c.Param("key"), id, callerID, req)
	if err != nil {
		return mapSprintError(err)
	}
	return c.JSON(http.StatusOK, sprint)
}

// Backlog godoc
// @Summary      Get project backlog
// @Description  Returns issues with no sprint assigned, sorted by priority (Critical first) then created_at.
// @Tags         sprints
// @Produce      json
// @Param        key path string true "Project key"
// @Success      200 {array}  dto.IssueResponse
// @Failure      404 {object} apierror.APIError "Project not found"
// @Failure      401 {object} apierror.APIError "Unauthorized"
// @Router       /projects/{key}/backlog [get]
// @Security     BearerAuth
func (h *SprintHandler) Backlog(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	issues, err := h.svc.Backlog(c.Request().Context(), c.Param("key"), callerID)
	if err != nil {
		return mapSprintError(err)
	}
	resp := make([]*dto.IssueResponse, len(issues))
	for i, issue := range issues {
		resp[i] = &dto.IssueResponse{
			ID:          issue.ID,
			Key:         issue.Key,
			ProjectID:   issue.ProjectID,
			Title:       issue.Title,
			Description: issue.Description,
			Type:        issue.Type,
			Status:      issue.Status,
			Priority:    issue.Priority,
			AssigneeID:  issue.AssigneeID,
			ReporterID:  issue.ReporterID,
			SprintID:    issue.SprintID,
			StoryPoints: issue.StoryPoints,
		}
	}
	return c.JSON(http.StatusOK, resp)
}

// GetIssues godoc
// @Summary      Get sprint issues
// @Description  Returns all issues assigned to the given sprint.
// @Tags         sprints
// @Produce      json
// @Param        key path string true "Project key"
// @Param        id  path int    true "Sprint ID"
// @Success      200 {array}  dto.IssueResponse
// @Failure      404 {object} apierror.APIError "Sprint or project not found"
// @Failure      401 {object} apierror.APIError "Unauthorized"
// @Router       /projects/{key}/sprints/{id}/issues [get]
// @Security     BearerAuth
func (h *SprintHandler) GetIssues(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	id, err := parseSprintID(c)
	if err != nil {
		return echo.ErrBadRequest
	}
	issues, err := h.svc.GetIssues(c.Request().Context(), c.Param("key"), id, callerID)
	if err != nil {
		return mapSprintError(err)
	}
	resp := make([]*dto.IssueResponse, len(issues))
	for i, issue := range issues {
		resp[i] = &dto.IssueResponse{
			ID:          issue.ID,
			Key:         issue.Key,
			ProjectID:   issue.ProjectID,
			Title:       issue.Title,
			Description: issue.Description,
			Type:        issue.Type,
			Status:      issue.Status,
			Priority:    issue.Priority,
			AssigneeID:  issue.AssigneeID,
			ReporterID:  issue.ReporterID,
			SprintID:    issue.SprintID,
			StoryPoints: issue.StoryPoints,
		}
	}
	return c.JSON(http.StatusOK, resp)
}

// Burndown godoc
// @Summary      Sprint burndown data
// @Description  Returns story points remaining per day from sprint start to today (or end date). Computed on-the-fly from issue data.
// @Tags         sprints
// @Produce      json
// @Param        key path string true "Project key"
// @Param        id  path int    true "Sprint ID"
// @Success      200 {object} dto.BurndownResponse
// @Failure      400 {object} apierror.APIError "Sprint has not been started"
// @Failure      404 {object} apierror.APIError "Sprint or project not found"
// @Router       /projects/{key}/sprints/{id}/burndown [get]
// @Security     BearerAuth
func (h *SprintHandler) Burndown(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	id, err := parseSprintID(c)
	if err != nil {
		return echo.ErrBadRequest
	}
	data, err := h.svc.Burndown(c.Request().Context(), c.Param("key"), id, callerID)
	if err != nil {
		return mapSprintError(err)
	}
	return c.JSON(http.StatusOK, data)
}

func parseSprintID(c echo.Context) (uint, error) {
	raw, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(raw), nil
}

func mapSprintError(err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return echo.NewHTTPError(http.StatusNotFound, "not found")
	case errors.Is(err, domain.ErrForbidden):
		return echo.NewHTTPError(http.StatusForbidden, "forbidden")
	case errors.Is(err, domain.ErrSprintAlreadyActive):
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrSprintNotDeletable):
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrSprintNotEditable):
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrSprintInvalidTransition):
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrSprintNotStarted):
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return err
}
