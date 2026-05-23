package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/service"
)

// LabelHandler exposes label CRUD at /projects/:key/labels and
// issue-label attachment at /issues/:id/labels/:lid.
type LabelHandler struct {
	svc service.LabelService
}

func NewLabelHandler(svc service.LabelService) *LabelHandler {
	return &LabelHandler{svc: svc}
}

// Create godoc
// @Summary      Create a label for a project
// @Tags         labels
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        key  path string                  true "Project key"
// @Param        body body dto.CreateLabelRequest  true "Label payload"
// @Success      201 {object} dto.LabelResponse
// @Failure      400 {object} apierror.APIError
// @Failure      401 {object} apierror.APIError
// @Failure      403 {object} apierror.APIError
// @Router       /projects/{key}/labels [post]
func (h *LabelHandler) Create(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	var req dto.CreateLabelRequest
	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}
	resp, err := h.svc.Create(c.Request().Context(), c.Param("key"), callerID, req)
	if err != nil {
		return mapLabelError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// List godoc
// @Summary      List labels for a project
// @Tags         labels
// @Produce      json
// @Security     BearerAuth
// @Param        key path string true "Project key"
// @Success      200 {array} dto.LabelResponse
// @Failure      401 {object} apierror.APIError
// @Failure      403 {object} apierror.APIError
// @Router       /projects/{key}/labels [get]
func (h *LabelHandler) List(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	labels, err := h.svc.ListByProject(c.Request().Context(), c.Param("key"), callerID)
	if err != nil {
		return mapLabelError(err)
	}
	return c.JSON(http.StatusOK, labels)
}

// Delete godoc
// @Summary      Delete a project label
// @Tags         labels
// @Security     BearerAuth
// @Param        key path string true "Project key"
// @Param        lid path int    true "Label ID"
// @Success      204
// @Failure      401 {object} apierror.APIError
// @Failure      403 {object} apierror.APIError
// @Failure      404 {object} apierror.APIError
// @Router       /projects/{key}/labels/{lid} [delete]
func (h *LabelHandler) Delete(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	lid, err := parseUintParam(c, "lid")
	if err != nil {
		return echo.ErrBadRequest
	}
	if err := h.svc.Delete(c.Request().Context(), c.Param("key"), lid, callerID); err != nil {
		return mapLabelError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ListByIssue godoc
// @Summary      List labels attached to an issue
// @Tags         labels
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Issue ID"
// @Success      200 {array} dto.LabelResponse
// @Failure      401 {object} apierror.APIError
// @Failure      403 {object} apierror.APIError
// @Failure      404 {object} apierror.APIError
// @Router       /issues/{id}/labels [get]
func (h *LabelHandler) ListByIssue(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	issueID, err := parseUintParam(c, "id")
	if err != nil {
		return echo.ErrBadRequest
	}
	labels, err := h.svc.ListByIssue(c.Request().Context(), issueID, callerID)
	if err != nil {
		return mapLabelError(err)
	}
	return c.JSON(http.StatusOK, labels)
}

// Attach godoc
// @Summary      Attach a label to an issue
// @Tags         labels
// @Security     BearerAuth
// @Param        id  path int true "Issue ID"
// @Param        lid path int true "Label ID"
// @Success      204
// @Failure      401 {object} apierror.APIError
// @Failure      403 {object} apierror.APIError
// @Failure      404 {object} apierror.APIError
// @Router       /issues/{id}/labels/{lid} [post]
func (h *LabelHandler) Attach(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	issueID, err := parseUintParam(c, "id")
	if err != nil {
		return echo.ErrBadRequest
	}
	lid, err := parseUintParam(c, "lid")
	if err != nil {
		return echo.ErrBadRequest
	}
	if err := h.svc.AttachToIssue(c.Request().Context(), issueID, lid, callerID); err != nil {
		return mapLabelError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// Detach godoc
// @Summary      Detach a label from an issue
// @Tags         labels
// @Security     BearerAuth
// @Param        id  path int true "Issue ID"
// @Param        lid path int true "Label ID"
// @Success      204
// @Failure      401 {object} apierror.APIError
// @Failure      403 {object} apierror.APIError
// @Failure      404 {object} apierror.APIError
// @Router       /issues/{id}/labels/{lid} [delete]
func (h *LabelHandler) Detach(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	issueID, err := parseUintParam(c, "id")
	if err != nil {
		return echo.ErrBadRequest
	}
	lid, err := parseUintParam(c, "lid")
	if err != nil {
		return echo.ErrBadRequest
	}
	if err := h.svc.DetachFromIssue(c.Request().Context(), issueID, lid, callerID); err != nil {
		return mapLabelError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func mapLabelError(err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return echo.NewHTTPError(http.StatusNotFound, "not found")
	case errors.Is(err, domain.ErrForbidden):
		return echo.NewHTTPError(http.StatusForbidden, "forbidden")
	}
	return err
}
