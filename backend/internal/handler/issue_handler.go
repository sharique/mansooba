package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/service"
)

// IssueHandler exposes issue CRUD endpoints nested under /projects/:key/issues.
type IssueHandler struct {
	svc     service.IssueService
	relRepo domain.IssueRelationRepository // optional; when set, cascades relation delete on issue delete
}

// NewIssueHandler creates an IssueHandler backed by the given service.
func NewIssueHandler(svc service.IssueService) *IssueHandler {
	return &IssueHandler{svc: svc}
}

// WithRelationRepo attaches an IssueRelationRepository for cascade-delete on issue removal.
func (h *IssueHandler) WithRelationRepo(repo domain.IssueRelationRepository) *IssueHandler {
	h.relRepo = repo
	return h
}

// List godoc
// @Summary      List issues in a project
// @Tags         issues
// @Produce      json
// @Security     BearerAuth
// @Param        key         path  string false "Project key"
// @Param        q           query string false "Full-text search on title and description"
// @Param        type        query string false "Filter by type (task|story|bug|epic)"
// @Param        status      query string false "Filter by status (backlog|todo|in_progress|in_review|done)"
// @Param        priority    query string false "Filter by priority (low|medium|high|critical)"
// @Param        assignee_id query int    false "Filter by assignee user ID"
// @Param        label_id    query int    false "Filter by label ID"
// @Param        page        query int    false "Page number"
// @Param        limit       query int    false "Page size"
// @Success      200 {array}  dto.IssueResponse
// @Failure      401 {object} apierror.APIError "Unauthorized"
// @Failure      403 {object} apierror.APIError "Forbidden"
// @Failure      404 {object} apierror.APIError "Not found"
// @Router       /projects/{key}/issues [get]
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

// Create godoc
// @Summary      Create an issue in a project
// @Tags         issues
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        key  path string                  true "Project key"
// @Param        body body dto.CreateIssueRequest  true "Issue payload"
// @Success      201 {object} dto.IssueResponse
// @Failure      400 {object} apierror.APIError "Bad request"
// @Failure      401 {object} apierror.APIError "Unauthorized"
// @Failure      403 {object} apierror.APIError "Forbidden"
// @Failure      404 {object} apierror.APIError "Project not found"
// @Router       /projects/{key}/issues [post]
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

// Get godoc
// @Summary      Get an issue by ID
// @Tags         issues
// @Produce      json
// @Security     BearerAuth
// @Param        key path string true "Project key"
// @Param        id  path int    true "Issue ID"
// @Success      200 {object} dto.IssueResponse
// @Failure      400 {object} apierror.APIError "Bad request"
// @Failure      401 {object} apierror.APIError "Unauthorized"
// @Failure      403 {object} apierror.APIError "Forbidden"
// @Failure      404 {object} apierror.APIError "Not found"
// @Router       /projects/{key}/issues/{id} [get]
func (h *IssueHandler) Get(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	id, err := parseUintParam(c, "id")
	if err != nil {
		return echo.ErrBadRequest
	}
	issue, err := h.svc.FindByID(c.Request().Context(), c.Param("key"), id, callerID)
	if err != nil {
		return mapIssueError(err)
	}
	return c.JSON(http.StatusOK, issue)
}

// Update godoc
// @Summary      Update an issue
// @Tags         issues
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        key  path string                 true "Project key"
// @Param        id   path int                    true "Issue ID"
// @Param        body body dto.UpdateIssueRequest true "Update payload"
// @Success      200 {object} dto.IssueResponse
// @Failure      400 {object} apierror.APIError "Bad request"
// @Failure      401 {object} apierror.APIError "Unauthorized"
// @Failure      403 {object} apierror.APIError "Forbidden"
// @Failure      404 {object} apierror.APIError "Not found"
// @Router       /projects/{key}/issues/{id} [put]
func (h *IssueHandler) Update(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	id, err := parseUintParam(c, "id")
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

// Delete godoc
// @Summary      Delete an issue
// @Tags         issues
// @Security     BearerAuth
// @Param        key path string true "Project key"
// @Param        id  path int    true "Issue ID"
// @Success      204
// @Failure      400 {object} apierror.APIError "Bad request"
// @Failure      401 {object} apierror.APIError "Unauthorized"
// @Failure      403 {object} apierror.APIError "Forbidden"
// @Failure      404 {object} apierror.APIError "Not found"
// @Router       /projects/{key}/issues/{id} [delete]
func (h *IssueHandler) Delete(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	id, err := parseUintParam(c, "id")
	if err != nil {
		return echo.ErrBadRequest
	}
	if h.relRepo != nil {
		_ = h.relRepo.DeleteByIssueID(c.Request().Context(), id)
	}
	if err := h.svc.Delete(c.Request().Context(), c.Param("key"), id, callerID); err != nil {
		return mapIssueError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func mapIssueError(err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return echo.NewHTTPError(http.StatusNotFound, "not found")
	case errors.Is(err, domain.ErrForbidden):
		return echo.NewHTTPError(http.StatusForbidden, "forbidden")
	case errors.Is(err, domain.ErrConflict):
		return echo.NewHTTPError(http.StatusConflict, "conflict")
	case errors.Is(err, domain.ErrAttachmentStorageUnavailable):
		return echo.NewHTTPError(http.StatusBadGateway, "storage temporarily unavailable, please try again")
	}
	return err
}
