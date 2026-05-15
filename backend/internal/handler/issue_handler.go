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

// IssueHandler exposes issue CRUD endpoints nested under /projects/:key/issues.
type IssueHandler struct {
	svc service.IssueService
}

// NewIssueHandler creates an IssueHandler backed by the given service.
func NewIssueHandler(svc service.IssueService) *IssueHandler {
	return &IssueHandler{svc: svc}
}

func (h *IssueHandler) List(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	var q dto.IssueListQuery
	if err := c.Bind(&q); err != nil {
		return echo.ErrBadRequest
	}
	issues, err := h.svc.ListByProject(c.Request().Context(), c.Param("key"), callerID, q)
	if err != nil {
		return mapIssueError(err)
	}
	return c.JSON(http.StatusOK, issues)
}

func (h *IssueHandler) Create(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	var req dto.CreateIssueRequest
	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}
	issue, err := h.svc.Create(c.Request().Context(), c.Param("key"), callerID, req)
	if err != nil {
		return mapIssueError(err)
	}
	return c.JSON(http.StatusCreated, issue)
}

func (h *IssueHandler) Get(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	id, err := parseIssueID(c)
	if err != nil {
		return echo.ErrBadRequest
	}
	issue, err := h.svc.FindByID(c.Request().Context(), c.Param("key"), id, callerID)
	if err != nil {
		return mapIssueError(err)
	}
	return c.JSON(http.StatusOK, issue)
}

func (h *IssueHandler) Update(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	id, err := parseIssueID(c)
	if err != nil {
		return echo.ErrBadRequest
	}
	var req dto.UpdateIssueRequest
	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}
	issue, err := h.svc.Update(c.Request().Context(), c.Param("key"), id, callerID, req)
	if err != nil {
		return mapIssueError(err)
	}
	return c.JSON(http.StatusOK, issue)
}

func (h *IssueHandler) Delete(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	id, err := parseIssueID(c)
	if err != nil {
		return echo.ErrBadRequest
	}
	if err := h.svc.Delete(c.Request().Context(), c.Param("key"), id, callerID); err != nil {
		return mapIssueError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func parseIssueID(c echo.Context) (uint, error) {
	raw, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(raw), nil
}

func mapIssueError(err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return echo.NewHTTPError(http.StatusNotFound, "not found")
	case errors.Is(err, domain.ErrForbidden):
		return echo.NewHTTPError(http.StatusForbidden, "forbidden")
	case errors.Is(err, domain.ErrConflict):
		return echo.NewHTTPError(http.StatusConflict, "conflict")
	}
	return err
}
