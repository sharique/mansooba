package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/service"
)

var validSystemLogCategories = map[string]bool{
	domain.SystemLogCategoryAuthentication: true,
	domain.SystemLogCategoryAdminAction:    true,
	domain.SystemLogCategorySettingsChange: true,
	domain.SystemLogCategoryDBLifecycle:    true,
}

// SystemLogHandler exposes the admin-only System Logs read endpoint
// (011-system-logs).
type SystemLogHandler struct {
	svc     service.SystemLogService
	userSvc service.UserService
}

// NewSystemLogHandler creates a SystemLogHandler backed by the given services.
func NewSystemLogHandler(svc service.SystemLogService, userSvc service.UserService) *SystemLogHandler {
	return &SystemLogHandler{svc: svc, userSvc: userSvc}
}

// List godoc
// @Summary      List system log entries (Admin only)
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        category query string false "authentication|admin_action|settings_change|db_lifecycle"
// @Param        from query string false "RFC3339 lower bound"
// @Param        to query string false "RFC3339 upper bound"
// @Param        actor query string false "Actor filter"
// @Param        q query string false "Free-text search"
// @Param        page query int false "Page (1-based)" default(1)
// @Param        size query int false "Page size (max 100)" default(20)
// @Success      200 {object} dto.SystemLogListResponse
// @Failure      400 {object} apierror.APIError "Invalid query params"
// @Failure      403 {object} apierror.APIError "Forbidden"
// @Router       /admin/system-logs [get]
func (h *SystemLogHandler) List(c echo.Context) error {
	callerID := c.Get("userID").(uint)
	profile, err := h.userSvc.GetProfile(c.Request().Context(), callerID)
	if err != nil || !profile.IsAdmin {
		return echo.NewHTTPError(http.StatusForbidden, "forbidden")
	}

	query, err := parseSystemLogListQuery(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	result, err := h.svc.List(c.Request().Context(), domain.SystemLogListFilter{
		Category: query.Category,
		From:     query.From,
		To:       query.To,
		Actor:    query.Actor,
		Q:        query.Q,
		Page:     query.Page,
		Size:     query.Size,
	})
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, toSystemLogListResponse(result, query.Page, query.Size))
}

func parseSystemLogListQuery(c echo.Context) (dto.SystemLogListQuery, error) {
	q := dto.SystemLogListQuery{Page: 1, Size: 20}

	if cat := c.QueryParam("category"); cat != "" {
		if !validSystemLogCategories[cat] {
			return q, errors.New("invalid category")
		}
		q.Category = cat
	}

	if from := c.QueryParam("from"); from != "" {
		t, err := time.Parse(time.RFC3339, from)
		if err != nil {
			return q, errors.New("from must be a valid RFC3339 timestamp")
		}
		q.From = &t
	}
	if to := c.QueryParam("to"); to != "" {
		t, err := time.Parse(time.RFC3339, to)
		if err != nil {
			return q, errors.New("to must be a valid RFC3339 timestamp")
		}
		q.To = &t
	}
	if q.From != nil && q.To != nil && q.From.After(*q.To) {
		return q, errors.New("from must not be after to")
	}

	q.Actor = c.QueryParam("actor")
	q.Q = c.QueryParam("q")

	if p := c.QueryParam("page"); p != "" {
		page, err := strconv.Atoi(p)
		if err != nil || page < 1 {
			return q, errors.New("page must be a positive integer")
		}
		q.Page = page
	}
	if s := c.QueryParam("size"); s != "" {
		size, err := strconv.Atoi(s)
		if err != nil || size < 1 || size > 100 {
			return q, errors.New("size must be between 1 and 100")
		}
		q.Size = size
	}

	return q, nil
}

func toSystemLogListResponse(result domain.SystemLogListResult, page, size int) dto.SystemLogListResponse {
	entries := make([]dto.SystemLogEntryDTO, 0, len(result.Entries))
	for _, e := range result.Entries {
		var target *string
		if e.Target != "" {
			t := e.Target
			target = &t
		}
		entries = append(entries, dto.SystemLogEntryDTO{
			ID:            e.ID,
			EventCategory: e.Category,
			Action:        e.Action,
			Outcome:       e.Outcome,
			ActorLabel:    e.Actor,
			TargetLabel:   target,
			Detail:        e.Detail,
			CreatedAt:     e.CreatedAt,
		})
	}
	return dto.SystemLogListResponse{
		Entries: entries,
		Total:   result.Total,
		Page:    page,
		Size:    size,
	}
}
