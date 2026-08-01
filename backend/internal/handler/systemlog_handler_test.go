package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/handler"
	"github.com/sharique/mansooba/internal/service"
)

type stubSystemLogService struct {
	listFn func(ctx context.Context, filter domain.SystemLogListFilter) (domain.SystemLogListResult, error)
}

func (s *stubSystemLogService) Record(_ context.Context, _ domain.SystemLogEntry) {}

func (s *stubSystemLogService) List(ctx context.Context, filter domain.SystemLogListFilter) (domain.SystemLogListResult, error) {
	if s.listFn != nil {
		return s.listFn(ctx, filter)
	}
	return domain.SystemLogListResult{}, nil
}

func (s *stubSystemLogService) SyncRetention(_ context.Context) error { return nil }

var _ service.SystemLogService = (*stubSystemLogService)(nil)

func newSystemLogHandler(svc service.SystemLogService, isAdmin bool) (*echo.Echo, *handler.SystemLogHandler) {
	userSvc := &stubAuthUserService{
		getProfileFn: func(_ context.Context, _ uint) (*dto.UserProfileResponse, error) {
			return &dto.UserProfileResponse{ID: 1, IsAdmin: isAdmin}, nil
		},
	}
	e := newEcho()
	h := handler.NewSystemLogHandler(svc, userSvc)
	e.GET("/admin/system-logs", func(c echo.Context) error {
		c.Set("userID", uint(1))
		return h.List(c)
	})
	return e, h
}

func TestSystemLogHandler_List_Returns403_ForNonAdmin(t *testing.T) {
	e, _ := newSystemLogHandler(&stubSystemLogService{}, false)

	req := httptest.NewRequest(http.MethodGet, "/admin/system-logs", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestSystemLogHandler_List_Returns200_ForAdmin(t *testing.T) {
	svc := &stubSystemLogService{
		listFn: func(_ context.Context, _ domain.SystemLogListFilter) (domain.SystemLogListResult, error) {
			return domain.SystemLogListResult{
				Entries: []domain.SystemLogEntry{
					{ID: "1-abc", Category: "authentication", Action: "login_failed", Outcome: "failure", Actor: "jane@example.com"},
				},
				Total: 1,
			}, nil
		},
	}
	e, _ := newSystemLogHandler(svc, true)

	req := httptest.NewRequest(http.MethodGet, "/admin/system-logs", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp dto.SystemLogListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Total != 1 || len(resp.Entries) != 1 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.Entries[0].ActorLabel != "jane@example.com" {
		t.Errorf("unexpected actor_label: %s", resp.Entries[0].ActorLabel)
	}
	if resp.Page != 1 || resp.Size != 20 {
		t.Errorf("expected default page=1 size=20, got page=%d size=%d", resp.Page, resp.Size)
	}
}

func TestSystemLogHandler_List_EmptyResultIsEmptyArrayNotNull(t *testing.T) {
	e, _ := newSystemLogHandler(&stubSystemLogService{}, true)

	req := httptest.NewRequest(http.MethodGet, "/admin/system-logs", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	// A nil slice would marshal to `"entries":null`, not `"entries":[]`.
	if want := `"entries":[]`; !strings.Contains(rec.Body.String(), want) {
		t.Errorf("expected empty entries to marshal as [], body: %s", rec.Body.String())
	}
}

func TestSystemLogHandler_List_Returns400_ForInvalidPage(t *testing.T) {
	e, _ := newSystemLogHandler(&stubSystemLogService{}, true)

	req := httptest.NewRequest(http.MethodGet, "/admin/system-logs?page=abc", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestSystemLogHandler_List_Returns400_ForInvalidSize(t *testing.T) {
	e, _ := newSystemLogHandler(&stubSystemLogService{}, true)

	req := httptest.NewRequest(http.MethodGet, "/admin/system-logs?size=0", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestSystemLogHandler_List_Returns400_ForUnrecognizedCategory(t *testing.T) {
	e, _ := newSystemLogHandler(&stubSystemLogService{}, true)

	req := httptest.NewRequest(http.MethodGet, "/admin/system-logs?category=bogus", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestSystemLogHandler_List_Returns400_ForMalformedDate(t *testing.T) {
	e, _ := newSystemLogHandler(&stubSystemLogService{}, true)

	req := httptest.NewRequest(http.MethodGet, "/admin/system-logs?from=not-a-date", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestSystemLogHandler_List_Returns400_ForOutOfOrderDateRange(t *testing.T) {
	e, _ := newSystemLogHandler(&stubSystemLogService{}, true)

	req := httptest.NewRequest(http.MethodGet, "/admin/system-logs?from=2026-07-31T00:00:00Z&to=2026-07-01T00:00:00Z", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}
