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

// ProjectHandler exposes project CRUD and membership endpoints.
type ProjectHandler struct {
	svc service.ProjectService
}

// NewProjectHandler creates a ProjectHandler backed by the given service.
func NewProjectHandler(svc service.ProjectService) *ProjectHandler {
	return &ProjectHandler{svc: svc}
}

func (h *ProjectHandler) List(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	projects, err := h.svc.List(c.Request().Context(), callerID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, projects)
}

func (h *ProjectHandler) Create(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	var req dto.CreateProjectRequest
	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}
	resp, err := h.svc.Create(c.Request().Context(), callerID, req)
	if err != nil {
		return mapProjectError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

func (h *ProjectHandler) Get(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	resp, err := h.svc.FindByKey(c.Request().Context(), c.Param("key"), callerID)
	if err != nil {
		return mapProjectError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *ProjectHandler) Update(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	var req dto.UpdateProjectRequest
	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}
	resp, err := h.svc.Update(c.Request().Context(), c.Param("key"), callerID, req)
	if err != nil {
		return mapProjectError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *ProjectHandler) Delete(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	if err := h.svc.Delete(c.Request().Context(), c.Param("key"), callerID); err != nil {
		return mapProjectError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *ProjectHandler) ListMembers(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	members, err := h.svc.ListMembers(c.Request().Context(), c.Param("key"), callerID)
	if err != nil {
		return mapProjectError(err)
	}
	return c.JSON(http.StatusOK, members)
}

func (h *ProjectHandler) AddMember(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	var req dto.AddMemberRequest
	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}
	if err := h.svc.AddMember(c.Request().Context(), c.Param("key"), callerID, req); err != nil {
		return mapProjectError(err)
	}
	return c.NoContent(http.StatusCreated)
}

func (h *ProjectHandler) RemoveMember(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	targetID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil {
		return echo.ErrBadRequest
	}
	if err := h.svc.RemoveMember(c.Request().Context(), c.Param("key"), callerID, uint(targetID)); err != nil {
		return mapProjectError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func mapProjectError(err error) error {
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
