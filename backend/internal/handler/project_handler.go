package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v5"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/service"
)

// ProjectHandler exposes project CRUD and membership endpoints.
type ProjectHandler struct {
	svc service.ProjectService
}

// NewProjectHandler creates a ProjectHandler backed by the given service.
func NewProjectHandler(svc service.ProjectService) *ProjectHandler {
	return &ProjectHandler{svc: svc}
}

// List godoc
// @Summary      List projects for the current user
// @Tags         projects
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array}  dto.ProjectResponse
// @Failure      401 {object} apierror.APIError "Unauthorized"
// @Router       /projects [get]
func (h *ProjectHandler) List(c *echo.Context) error {
	callerID := c.Get("userID").(uint)
	projects, err := h.svc.List(c.Request().Context(), callerID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, projects)
}

// Create godoc
// @Summary      Create a new project
// @Tags         projects
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body dto.CreateProjectRequest true "Project payload"
// @Success      201 {object} dto.ProjectResponse
// @Failure      400 {object} apierror.APIError "Bad request"
// @Failure      401 {object} apierror.APIError "Unauthorized"
// @Failure      409 {object} apierror.APIError "Key already exists"
// @Router       /projects [post]
func (h *ProjectHandler) Create(c *echo.Context) error {
	callerID := c.Get("userID").(uint)
	var req dto.CreateProjectRequest
	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}
	if err := validateRequest(c, &req); err != nil {
		return err
	}
	resp, err := h.svc.Create(c.Request().Context(), callerID, req)
	if err != nil {
		return mapProjectError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// Get godoc
// @Summary      Get a project by key
// @Tags         projects
// @Produce      json
// @Security     BearerAuth
// @Param        key path string true "Project key"
// @Success      200 {object} dto.ProjectResponse
// @Failure      401 {object} apierror.APIError "Unauthorized"
// @Failure      403 {object} apierror.APIError "Forbidden"
// @Failure      404 {object} apierror.APIError "Not found"
// @Router       /projects/{key} [get]
func (h *ProjectHandler) Get(c *echo.Context) error {
	callerID := c.Get("userID").(uint)
	resp, err := h.svc.FindByKey(c.Request().Context(), c.Param("key"), callerID)
	if err != nil {
		return mapProjectError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// Update godoc
// @Summary      Update a project
// @Tags         projects
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        key  path string                         true "Project key"
// @Param        body body dto.UpdateProjectRequest true "Update payload"
// @Success      200 {object} dto.ProjectResponse
// @Failure      400 {object} apierror.APIError "Bad request"
// @Failure      401 {object} apierror.APIError "Unauthorized"
// @Failure      403 {object} apierror.APIError "Forbidden"
// @Failure      404 {object} apierror.APIError "Not found"
// @Router       /projects/{key} [put]
func (h *ProjectHandler) Update(c *echo.Context) error {
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

// Delete godoc
// @Summary      Delete an empty project
// @Description  Needs the project's exact name in the body and a project with no tasks left.
// @Tags         projects
// @Accept       json
// @Security     BearerAuth
// @Param        key  path string                   true "Project key"
// @Param        body body dto.DeleteProjectRequest true "Confirmation"
// @Success      204
// @Failure      400 {object} apierror.APIError "Name missing or wrong"
// @Failure      401 {object} apierror.APIError "Unauthorized"
// @Failure      403 {object} apierror.APIError "Forbidden"
// @Failure      404 {object} apierror.APIError "Not found"
// @Failure      409 {object} apierror.APIError "Project still has tasks (task_count)"
// @Router       /projects/{key} [delete]
func (h *ProjectHandler) Delete(c *echo.Context) error {
	callerID := c.Get("userID").(uint)
	var req dto.DeleteProjectRequest
	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}
	if err := h.svc.Delete(c.Request().Context(), c.Param("key"), callerID, req.ConfirmName); err != nil {
		return mapProjectError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ListMembers godoc
// @Summary      List members of a project
// @Tags         projects
// @Produce      json
// @Security     BearerAuth
// @Param        key path string true "Project key"
// @Success      200 {array}  dto.MemberResponse
// @Failure      401 {object} apierror.APIError "Unauthorized"
// @Failure      403 {object} apierror.APIError "Forbidden"
// @Failure      404 {object} apierror.APIError "Not found"
// @Router       /projects/{key}/members [get]
func (h *ProjectHandler) ListMembers(c *echo.Context) error {
	callerID := c.Get("userID").(uint)
	members, err := h.svc.ListMembers(c.Request().Context(), c.Param("key"), callerID)
	if err != nil {
		return mapProjectError(err)
	}
	return c.JSON(http.StatusOK, members)
}

// AddMember godoc
// @Summary      Add a member to a project
// @Tags         projects
// @Accept       json
// @Security     BearerAuth
// @Param        key  path string              true "Project key"
// @Param        body body dto.AddMemberRequest true "Member payload"
// @Success      201
// @Failure      400 {object} apierror.APIError "Bad request"
// @Failure      401 {object} apierror.APIError "Unauthorized"
// @Failure      403 {object} apierror.APIError "Forbidden"
// @Failure      404 {object} apierror.APIError "Not found"
// @Router       /projects/{key}/members [post]
func (h *ProjectHandler) AddMember(c *echo.Context) error {
	callerID := c.Get("userID").(uint)
	var req dto.AddMemberRequest
	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}
	if err := validateRequest(c, &req); err != nil {
		return err
	}
	if err := h.svc.AddMember(c.Request().Context(), c.Param("key"), callerID, req); err != nil {
		return mapProjectError(err)
	}
	return c.NoContent(http.StatusCreated)
}

// ChangeMemberRole godoc
// @Summary      Change a member's role
// @Tags         projects
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        key    path string                true "Project key"
// @Param        userId path string                true "User ID"
// @Param        body   body dto.ChangeRoleRequest true "New role"
// @Success      200 {object} dto.MemberResponse
// @Failure      400 {object} apierror.APIError "Bad request"
// @Failure      401 {object} apierror.APIError "Unauthorized"
// @Failure      403 {object} apierror.APIError "Forbidden"
// @Failure      404 {object} apierror.APIError "Not found"
// @Failure      409 {object} apierror.APIError "Would leave the project without an active admin"
// @Router       /projects/{key}/members/{userId} [patch]
func (h *ProjectHandler) ChangeMemberRole(c *echo.Context) error {
	callerID := c.Get("userID").(uint)
	targetID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil {
		return echo.ErrBadRequest
	}
	var req dto.ChangeRoleRequest
	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}
	if err := validateRequest(c, &req); err != nil {
		return err
	}
	resp, err := h.svc.ChangeMemberRole(c.Request().Context(), c.Param("key"), callerID, uint(targetID), req.Role)
	if err != nil {
		return mapProjectError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// RemoveMember godoc
// @Summary      Remove a member from a project
// @Tags         projects
// @Security     BearerAuth
// @Param        key    path string true "Project key"
// @Param        userId path string true "User ID"
// @Success      204
// @Failure      400 {object} apierror.APIError "Bad request"
// @Failure      401 {object} apierror.APIError "Unauthorized"
// @Failure      403 {object} apierror.APIError "Forbidden"
// @Failure      404 {object} apierror.APIError "Not found"
// @Router       /projects/{key}/members/{userId} [delete]
func (h *ProjectHandler) RemoveMember(c *echo.Context) error {
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
		return forbiddenError(err)
	case errors.Is(err, domain.ErrConflict):
		return echo.NewHTTPError(http.StatusConflict, "conflict")
	}
	// Everything else (invalid input, last admin, self-removal, project not empty)
	// is worded and given its status by the central error handler.
	return err
}
