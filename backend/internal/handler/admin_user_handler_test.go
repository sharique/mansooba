package handler_test

import (
	"context"
	"encoding/json"
	"errors"
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

// ──────────────────────────────────────────────────────────────────────────────
// Stubs
// ──────────────────────────────────────────────────────────────────────────────

type stubAdminUserService struct {
	listUsersFn func(ctx context.Context, page, size int) (*dto.AdminUserListResponse, error)
	getUserFn   func(ctx context.Context, id uint) (*dto.AdminUserDTO, error)
	setRoleFn   func(ctx context.Context, callerID, targetID uint, isAdmin bool) error
	setActiveFn func(ctx context.Context, callerID, targetID uint, isActive bool) error
}

func (s *stubAdminUserService) ListUsers(ctx context.Context, page, size int) (*dto.AdminUserListResponse, error) {
	if s.listUsersFn != nil {
		return s.listUsersFn(ctx, page, size)
	}
	return &dto.AdminUserListResponse{Users: []dto.AdminUserDTO{}, Total: 0, Page: page, Size: size}, nil
}

func (s *stubAdminUserService) GetUser(ctx context.Context, id uint) (*dto.AdminUserDTO, error) {
	if s.getUserFn != nil {
		return s.getUserFn(ctx, id)
	}
	return &dto.AdminUserDTO{ID: id, Name: "test", Email: "test@test.com", IsAdmin: false, IsActive: true}, nil
}

func (s *stubAdminUserService) SetRole(ctx context.Context, callerID, targetID uint, isAdmin bool) error {
	if s.setRoleFn != nil {
		return s.setRoleFn(ctx, callerID, targetID, isAdmin)
	}
	return nil
}

func (s *stubAdminUserService) SetActive(ctx context.Context, callerID, targetID uint, isActive bool) error {
	if s.setActiveFn != nil {
		return s.setActiveFn(ctx, callerID, targetID, isActive)
	}
	return nil
}

var _ service.AdminUserService = (*stubAdminUserService)(nil)

func newAdminHandler(svc service.AdminUserService, isAdmin bool) (*echo.Echo, *handler.AdminUserHandler) {
	userSvc := &stubAuthUserService{
		getProfileFn: func(_ context.Context, _ uint) (*dto.UserProfileResponse, error) {
			return &dto.UserProfileResponse{ID: 1, IsAdmin: isAdmin}, nil
		},
	}
	e := newEcho()
	h := handler.NewAdminUserHandler(svc, userSvc)
	return e, h
}

func withAdminAuth(e *echo.Echo, h *handler.AdminUserHandler) {
	e.GET("/admin/users", func(c echo.Context) error {
		c.Set("userID", uint(1))
		return h.ListUsers(c)
	})
	e.PATCH("/admin/users/:id", func(c echo.Context) error {
		c.Set("userID", uint(1))
		return h.PatchUser(c)
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// T027: GET /admin/users tests
// ──────────────────────────────────────────────────────────────────────────────

func TestAdminUserHandler_ListUsers_Returns403_ForNonAdmin(t *testing.T) {
	e, h := newAdminHandler(&stubAdminUserService{}, false)
	withAdminAuth(e, h)

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestAdminUserHandler_ListUsers_Returns400_ForInvalidParams(t *testing.T) {
	e, h := newAdminHandler(&stubAdminUserService{}, true)
	withAdminAuth(e, h)

	req := httptest.NewRequest(http.MethodGet, "/admin/users?page=abc", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestAdminUserHandler_ListUsers_Returns200_WithResponseShape(t *testing.T) {
	svc := &stubAdminUserService{
		listUsersFn: func(_ context.Context, page, size int) (*dto.AdminUserListResponse, error) {
			return &dto.AdminUserListResponse{
				Users: []dto.AdminUserDTO{
					{ID: 1, Name: "Alice", Email: "alice@test.com", IsAdmin: true, IsActive: true},
				},
				Total: 1, Page: page, Size: size,
			}, nil
		},
	}
	e, h := newAdminHandler(svc, true)
	withAdminAuth(e, h)

	req := httptest.NewRequest(http.MethodGet, "/admin/users", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp dto.AdminUserListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Total != 1 || len(resp.Users) != 1 {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestAdminUserHandler_ListUsers_OutOfRangePage_Returns200WithEmptyArray(t *testing.T) {
	svc := &stubAdminUserService{
		listUsersFn: func(_ context.Context, page, _ int) (*dto.AdminUserListResponse, error) {
			return &dto.AdminUserListResponse{Users: []dto.AdminUserDTO{}, Total: 5, Page: page, Size: 20}, nil
		},
	}
	e, h := newAdminHandler(svc, true)
	withAdminAuth(e, h)

	req := httptest.NewRequest(http.MethodGet, "/admin/users?page=999", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	var resp dto.AdminUserListResponse
	json.NewDecoder(rec.Body).Decode(&resp) //nolint:errcheck
	if len(resp.Users) != 0 {
		t.Error("expected empty users array for out-of-range page")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// T027: PATCH /admin/users/:id tests
// ──────────────────────────────────────────────────────────────────────────────

func TestAdminUserHandler_PatchUser_Returns403_ForNonAdmin(t *testing.T) {
	e, h := newAdminHandler(&stubAdminUserService{}, false)
	withAdminAuth(e, h)

	body := `{"is_admin":true}`
	req := httptest.NewRequest(http.MethodPatch, "/admin/users/2", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", rec.Code)
	}
}

func TestAdminUserHandler_PatchUser_Returns400_ForEmptyBody(t *testing.T) {
	e, h := newAdminHandler(&stubAdminUserService{}, true)
	withAdminAuth(e, h)

	req := httptest.NewRequest(http.MethodPatch, "/admin/users/2", strings.NewReader(`{}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminUserHandler_PatchUser_Returns409_ForLastAdmin(t *testing.T) {
	svc := &stubAdminUserService{
		setRoleFn: func(_ context.Context, _, _ uint, _ bool) error {
			return domain.ErrLastAdmin
		},
	}
	e, h := newAdminHandler(svc, true)
	withAdminAuth(e, h)

	body := `{"is_admin":false}`
	req := httptest.NewRequest(http.MethodPatch, "/admin/users/1", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminUserHandler_PatchUser_Returns200_OnRoleChange(t *testing.T) {
	svc := &stubAdminUserService{
		getUserFn: func(_ context.Context, id uint) (*dto.AdminUserDTO, error) {
			return &dto.AdminUserDTO{ID: id, Name: "Bob", Email: "bob@test.com", IsAdmin: true, IsActive: true}, nil
		},
	}
	e, h := newAdminHandler(svc, true)
	withAdminAuth(e, h)

	body := `{"is_admin":true}`
	req := httptest.NewRequest(http.MethodPatch, "/admin/users/2", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp dto.AdminUserDTO
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.IsAdmin {
		t.Error("expected is_admin=true in response")
	}
}

func TestAdminUserHandler_PatchUser_Returns200_OnStatusChange(t *testing.T) {
	svc := &stubAdminUserService{
		getUserFn: func(_ context.Context, id uint) (*dto.AdminUserDTO, error) {
			return &dto.AdminUserDTO{ID: id, Name: "Carol", Email: "carol@test.com", IsAdmin: false, IsActive: false}, nil
		},
	}
	e, h := newAdminHandler(svc, true)
	withAdminAuth(e, h)

	body := `{"is_active":false}`
	req := httptest.NewRequest(http.MethodPatch, "/admin/users/3", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminUserHandler_PatchUser_Returns404_WhenUserNotFound(t *testing.T) {
	svc := &stubAdminUserService{
		setRoleFn: func(_ context.Context, _, _ uint, _ bool) error {
			return domain.ErrNotFound
		},
	}
	e, h := newAdminHandler(svc, true)
	withAdminAuth(e, h)

	body := `{"is_admin":true}`
	req := httptest.NewRequest(http.MethodPatch, "/admin/users/999", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminUserHandler_PatchUser_Returns200_BothFields(t *testing.T) {
	var roleSet, activeSet bool
	svc := &stubAdminUserService{
		setRoleFn: func(_ context.Context, _, _ uint, _ bool) error {
			roleSet = true
			return nil
		},
		setActiveFn: func(_ context.Context, _, _ uint, _ bool) error {
			activeSet = true
			return nil
		},
	}
	e, h := newAdminHandler(svc, true)
	withAdminAuth(e, h)

	body := `{"is_admin":true,"is_active":false}`
	req := httptest.NewRequest(http.MethodPatch, "/admin/users/2", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !roleSet || !activeSet {
		t.Errorf("expected both SetRole and SetActive to be called: roleSet=%v activeSet=%v", roleSet, activeSet)
	}
}

// Ensure the handler test uses errors package (suppress lint)
var _ = errors.New
