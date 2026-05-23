package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/service"
)

// CommentHandler exposes comment CRUD at /issues/:id/comments.
type CommentHandler struct {
	svc service.CommentService
}

func NewCommentHandler(svc service.CommentService) *CommentHandler {
	return &CommentHandler{svc: svc}
}

// Create godoc
// @Summary      Add a comment to an issue
// @Tags         comments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int                      true "Issue ID"
// @Param        body body dto.CreateCommentRequest true "Comment body"
// @Success      201 {object} dto.CommentResponse
// @Failure      400 {object} apierror.APIError
// @Failure      401 {object} apierror.APIError
// @Failure      403 {object} apierror.APIError
// @Failure      404 {object} apierror.APIError
// @Router       /issues/{id}/comments [post]
func (h *CommentHandler) Create(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	issueID, err := parseUintParam(c, "id")
	if err != nil {
		return echo.ErrBadRequest
	}
	var req dto.CreateCommentRequest
	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}
	resp, err := h.svc.Create(c.Request().Context(), issueID, callerID, req)
	if err != nil {
		return mapCommentError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// List godoc
// @Summary      List comments on an issue
// @Tags         comments
// @Produce      json
// @Security     BearerAuth
// @Param        id path int true "Issue ID"
// @Success      200 {array} dto.CommentResponse
// @Failure      401 {object} apierror.APIError
// @Failure      403 {object} apierror.APIError
// @Failure      404 {object} apierror.APIError
// @Router       /issues/{id}/comments [get]
func (h *CommentHandler) List(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	issueID, err := parseUintParam(c, "id")
	if err != nil {
		return echo.ErrBadRequest
	}
	comments, err := h.svc.List(c.Request().Context(), issueID, callerID)
	if err != nil {
		return mapCommentError(err)
	}
	return c.JSON(http.StatusOK, comments)
}

// Update godoc
// @Summary      Edit a comment
// @Tags         comments
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path int                      true "Issue ID"
// @Param        cid  path int                      true "Comment ID"
// @Param        body body dto.UpdateCommentRequest true "Updated body"
// @Success      200 {object} dto.CommentResponse
// @Failure      400 {object} apierror.APIError
// @Failure      401 {object} apierror.APIError
// @Failure      403 {object} apierror.APIError
// @Failure      404 {object} apierror.APIError
// @Router       /issues/{id}/comments/{cid} [put]
func (h *CommentHandler) Update(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	cid, err := parseUintParam(c, "cid")
	if err != nil {
		return echo.ErrBadRequest
	}
	// The :id (issue ID) param is not validated here — the service enforces ownership
	// by looking up the comment directly. A mismatched issue ID returns 200 if the
	// comment exists and the caller owns it.
	var req dto.UpdateCommentRequest
	if err := c.Bind(&req); err != nil {
		return echo.ErrBadRequest
	}
	resp, err := h.svc.Update(c.Request().Context(), cid, callerID, req)
	if err != nil {
		return mapCommentError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// Delete godoc
// @Summary      Delete a comment
// @Tags         comments
// @Security     BearerAuth
// @Param        id  path int true "Issue ID"
// @Param        cid path int true "Comment ID"
// @Success      204
// @Failure      401 {object} apierror.APIError
// @Failure      403 {object} apierror.APIError
// @Failure      404 {object} apierror.APIError
// @Router       /issues/{id}/comments/{cid} [delete]
func (h *CommentHandler) Delete(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	cid, err := parseUintParam(c, "cid")
	if err != nil {
		return echo.ErrBadRequest
	}
	if err := h.svc.Delete(c.Request().Context(), cid, callerID); err != nil {
		return mapCommentError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func mapCommentError(err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return echo.NewHTTPError(http.StatusNotFound, "not found")
	case errors.Is(err, domain.ErrForbidden):
		return echo.NewHTTPError(http.StatusForbidden, "forbidden")
	}
	return err
}
