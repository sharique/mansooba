package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/service"
)

// IssueRelationHandler exposes GET/POST/DELETE endpoints for task-to-task relations.
type IssueRelationHandler struct {
	svc service.IssueRelationService
}

// NewIssueRelationHandler creates an IssueRelationHandler backed by the given service.
func NewIssueRelationHandler(svc service.IssueRelationService) *IssueRelationHandler {
	return &IssueRelationHandler{svc: svc}
}

// List godoc
// @Summary      List relations for an issue
// @Tags         issue-relations
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int true "Issue ID"
// @Success      200 {array}  dto.RelationResponse
// @Failure      401 {object} apierror.APIError
// @Failure      404 {object} apierror.APIError
// @Router       /issues/{id}/relations [get]
func (h *IssueRelationHandler) List(c echo.Context) error {
	issueID, err := parseUintParam(c, "id")
	if err != nil {
		return echo.ErrBadRequest
	}
	relations, err := h.svc.List(c.Request().Context(), issueID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return echo.ErrNotFound
		}
		return err
	}
	if relations == nil {
		relations = []*dto.RelationResponse{}
	}
	return c.JSON(http.StatusOK, relations)
}

// Create godoc
// @Summary      Create a relation between two issues
// @Tags         issue-relations
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int true "Issue ID"
// @Param        body body dto.CreateRelationRequest true "Relation payload"
// @Success      201 {object} dto.RelationResponse
// @Failure      400 {object} apierror.APIError
// @Failure      401 {object} apierror.APIError
// @Router       /issues/{id}/relations [post]
func (h *IssueRelationHandler) Create(c echo.Context) error {
	issueID, err := parseUintParam(c, "id")
	if err != nil {
		return echo.ErrBadRequest
	}
	callerID := c.Get("userID").(uint)
	var req dto.CreateRelationRequest
	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}

	rel, err := h.svc.Create(c.Request().Context(), issueID, callerID, req)
	if err != nil {
		return mapRelationError(err)
	}
	return c.JSON(http.StatusCreated, rel)
}

// Delete godoc
// @Summary      Remove a relation
// @Tags         issue-relations
// @Security     BearerAuth
// @Param        id  path int true "Issue ID"
// @Param        rid path int true "Relation ID"
// @Success      204
// @Failure      401 {object} apierror.APIError
// @Failure      404 {object} apierror.APIError
// @Router       /issues/{id}/relations/{rid} [delete]
func (h *IssueRelationHandler) Delete(c echo.Context) error {
	relationID, err := parseUintParam(c, "rid")
	if err != nil {
		return echo.ErrBadRequest
	}
	callerID := c.Get("userID").(uint)

	if err := h.svc.Delete(c.Request().Context(), relationID, callerID); err != nil {
		if errors.Is(err, service.ErrRelationNotFound) {
			return echo.ErrNotFound
		}
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func mapRelationError(err error) error {
	switch {
	case errors.Is(err, service.ErrSelfRelation):
		return echo.NewHTTPError(http.StatusBadRequest, "self_relation")
	case errors.Is(err, service.ErrCircularRelation):
		return echo.NewHTTPError(http.StatusBadRequest, "circular_relation")
	case errors.Is(err, service.ErrCrossProjectRelation):
		return echo.NewHTTPError(http.StatusBadRequest, "cross_project_relation")
	case errors.Is(err, service.ErrDuplicateRelation):
		return echo.NewHTTPError(http.StatusBadRequest, "duplicate_relation")
	case errors.Is(err, service.ErrInvalidRelationType):
		return echo.NewHTTPError(http.StatusBadRequest, "invalid_relation_type")
	case errors.Is(err, domain.ErrNotFound):
		return echo.ErrNotFound
	default:
		return err
	}
}
