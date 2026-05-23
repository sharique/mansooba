package handler

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/service"
)

// BoardHandler exposes the kanban board aggregation endpoint.
type BoardHandler struct {
	svc service.BoardService
}

// NewBoardHandler creates a BoardHandler backed by the given service.
func NewBoardHandler(svc service.BoardService) *BoardHandler {
	return &BoardHandler{svc: svc}
}

// GetBoard godoc
// @Summary      Get the kanban board for a project
// @Tags         board
// @Produce      json
// @Security     BearerAuth
// @Param        key path string true "Project key"
// @Success      200 {object} dto.BoardResponse
// @Failure      401 {object} apierror.APIError "Unauthorized"
// @Failure      403 {object} apierror.APIError "Forbidden"
// @Failure      404 {object} apierror.APIError "Not found"
// @Router       /projects/{key}/board [get]
func (h *BoardHandler) GetBoard(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	resp, err := h.svc.GetBoard(c.Request().Context(), c.Param("key"), callerID)
	if err != nil {
		return mapBoardError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func mapBoardError(err error) error {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return echo.NewHTTPError(http.StatusNotFound, "not found")
	case errors.Is(err, domain.ErrForbidden):
		return echo.NewHTTPError(http.StatusForbidden, "forbidden")
	}
	return err
}
