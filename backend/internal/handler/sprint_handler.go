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

func (h *SprintHandler) List(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	sprints, err := h.svc.List(c.Request().Context(), c.Param("key"), callerID)
	if err != nil {
		return mapSprintError(err)
	}
	return c.JSON(http.StatusOK, sprints)
}

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
