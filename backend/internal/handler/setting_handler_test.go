package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/handler"
	"github.com/sharique/mansooba/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── stubSettingService ────────────────────────────────────────────────────────

type stubSettingService struct {
	getAllFn func() (*dto.SettingsResponse, error)
	patchFn func(userID uint, req dto.PatchSettingsRequest) (*dto.SettingsResponse, error)
}

func (s *stubSettingService) GetAll(_ context.Context) (*dto.SettingsResponse, error) {
	return s.getAllFn()
}

func (s *stubSettingService) Patch(_ context.Context, userID uint, req dto.PatchSettingsRequest) (*dto.SettingsResponse, error) {
	return s.patchFn(userID, req)
}

var _ service.SettingService = (*stubSettingService)(nil)

// ── helpers ───────────────────────────────────────────────────────────────────

func newSettingEcho(h *handler.SettingHandler, isAdmin bool) *echo.Echo {
	e := newEcho()
	api := e.Group("/api/v1", func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("userID", uint(1))
			return next(c)
		}
	})
	api.GET("/settings", h.GetAll)
	api.PATCH("/settings", h.Patch)
	_ = isAdmin // isAdmin is determined by the stubUserService profile
	return e
}

func newSettingHandler(getAllFn func() (*dto.SettingsResponse, error), patchFn func(uint, dto.PatchSettingsRequest) (*dto.SettingsResponse, error), isAdmin bool) *handler.SettingHandler {
	settingSvc := &stubSettingService{getAllFn: getAllFn, patchFn: patchFn}
	userSvc := &stubUserService{
		profile: &dto.UserProfileResponse{ID: 1, Name: "User", Email: "user@example.com", IsAdmin: isAdmin},
	}
	return handler.NewSettingHandler(settingSvc, userSvc)
}

func defaultSettings() *dto.SettingsResponse {
	return &dto.SettingsResponse{
		OrganizationName: "Mansooba",
		DateFormat:       "YYYY-MM-DD",
		TimeFormat:       "24h",
		Locale:           "en-US",
		WeekStartDay:     "monday",
	}
}

// ── T009: GET /api/v1/settings ────────────────────────────────────────────────

func TestSettingHandler_GetAll_Returns200WithDefaults(t *testing.T) {
	h := newSettingHandler(func() (*dto.SettingsResponse, error) { return defaultSettings(), nil }, nil, false)
	e := newSettingEcho(h, false)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp dto.SettingsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "Mansooba", resp.OrganizationName)
	assert.Equal(t, "YYYY-MM-DD", resp.DateFormat)
	assert.Equal(t, "24h", resp.TimeFormat)
	assert.Equal(t, "en-US", resp.Locale)
	assert.Equal(t, "monday", resp.WeekStartDay)
}

func TestSettingHandler_GetAll_Returns200WithDBValues(t *testing.T) {
	h := newSettingHandler(func() (*dto.SettingsResponse, error) {
		return &dto.SettingsResponse{
			OrganizationName: "Acme Corp",
			DateFormat:       "DD/MM/YYYY",
			TimeFormat:       "12h",
			Locale:           "en-GB",
			WeekStartDay:     "sunday",
		}, nil
	}, nil, false)
	e := newSettingEcho(h, false)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp dto.SettingsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "Acme Corp", resp.OrganizationName)
	assert.Equal(t, "sunday", resp.WeekStartDay)
}

// ── T009: PATCH /api/v1/settings ─────────────────────────────────────────────

func TestSettingHandler_Patch_ByAdmin_Returns200(t *testing.T) {
	updated := defaultSettings()
	updated.OrganizationName = "Acme Corp"
	h := newSettingHandler(nil, func(_ uint, req dto.PatchSettingsRequest) (*dto.SettingsResponse, error) {
		return updated, nil
	}, true) // isAdmin = true
	e := newSettingEcho(h, true)

	body := `{"organization_name":"Acme Corp"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/settings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp dto.SettingsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "Acme Corp", resp.OrganizationName)
}

func TestSettingHandler_Patch_ByNonAdmin_Returns403(t *testing.T) {
	h := newSettingHandler(nil, func(_ uint, req dto.PatchSettingsRequest) (*dto.SettingsResponse, error) {
		return defaultSettings(), nil
	}, false) // isAdmin = false
	e := newSettingEcho(h, false)

	body := `{"organization_name":"Hacked"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/settings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestSettingHandler_Patch_InvalidDateFormat_Returns400(t *testing.T) {
	h := newSettingHandler(nil, func(_ uint, req dto.PatchSettingsRequest) (*dto.SettingsResponse, error) {
		return nil, service.ErrInvalidSettingValue
	}, true) // isAdmin = true
	e := newSettingEcho(h, true)

	body := `{"date_format":"not-valid"}`
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/settings", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
