package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
)

// NotificationHandler exposes notification endpoints.
type NotificationHandler struct {
	repo domain.NotificationRepository
}

func NewNotificationHandler(repo domain.NotificationRepository) *NotificationHandler {
	return &NotificationHandler{repo: repo}
}

// List godoc
// @Summary      List unread notifications for the current user
// @Tags         notifications
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} dto.NotificationResponse
// @Failure      401 {object} apierror.APIError
// @Router       /notifications [get]
func (h *NotificationHandler) List(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	notifications, err := h.repo.FindUnreadByRecipientID(c.Request().Context(), callerID)
	if err != nil {
		return err
	}
	result := make([]*dto.NotificationResponse, 0, len(notifications))
	for _, n := range notifications {
		result = append(result, &dto.NotificationResponse{
			ID: n.ID, RecipientID: n.RecipientID, ActorID: n.ActorID,
			IssueID: n.IssueID, IssueKey: n.IssueKey, ProjectKey: n.ProjectKey,
			CommentID: n.CommentID, Read: n.Read, CreatedAt: n.CreatedAt,
		})
	}
	return c.JSON(http.StatusOK, result)
}

// MarkRead godoc
// @Summary      Mark a notification as read
// @Tags         notifications
// @Security     BearerAuth
// @Param        id path int true "Notification ID"
// @Success      204
// @Failure      401 {object} apierror.APIError
// @Failure      404 {object} apierror.APIError
// @Router       /notifications/{id}/read [put]
func (h *NotificationHandler) MarkRead(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	id, err := parseUintParam(c, "id")
	if err != nil {
		return echo.ErrBadRequest
	}
	if err := h.repo.MarkRead(c.Request().Context(), id, callerID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "not found")
		}
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
