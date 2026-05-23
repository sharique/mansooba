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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── stubUserService ───────────────────────────────────────────────────────────

type stubUserService struct {
	profile  *dto.UserProfileResponse
	updateFn func(userID uint, req dto.UpdateProfileRequest) (*dto.UserProfileResponse, error)
}

func (s *stubUserService) GetProfile(_ context.Context, _ uint) (*dto.UserProfileResponse, error) {
	if s.profile == nil {
		return nil, domain.ErrNotFound
	}
	return s.profile, nil
}

func (s *stubUserService) UpdateProfile(_ context.Context, userID uint, req dto.UpdateProfileRequest) (*dto.UserProfileResponse, error) {
	if s.updateFn != nil {
		return s.updateFn(userID, req)
	}
	s.profile.Name = req.FullName
	return s.profile, nil
}

var _ service.UserService = (*stubUserService)(nil)

// ── stubActivityServiceForUser ────────────────────────────────────────────────

type stubActivityServiceForUser struct{}

func (s *stubActivityServiceForUser) Record(_ context.Context, _ *domain.ActivityEvent) error {
	return nil
}
func (s *stubActivityServiceForUser) ListByIssue(_ context.Context, _ uint) ([]*dto.ActivityEventResponse, error) {
	return nil, nil
}
func (s *stubActivityServiceForUser) GetMyActivity(_ context.Context, _ uint, _, _ int) ([]*dto.ActivityEventResponse, error) {
	return []*dto.ActivityEventResponse{
		{ID: 1, ActorName: "Me", Kind: "status_changed"},
	}, nil
}

var _ service.ActivityService = (*stubActivityServiceForUser)(nil)

// ── helpers ───────────────────────────────────────────────────────────────────

func newUserHandler() (*handler.UserHandler, *stubUserService) {
	profile := &dto.UserProfileResponse{ID: 1, Name: "Alice", Email: "alice@example.com"}
	userSvc := &stubUserService{profile: profile}
	activitySvc := &stubActivityServiceForUser{}
	return handler.NewUserHandler(userSvc, activitySvc), userSvc
}

func setupUserEcho(method, path string, body string, userID uint) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userID", userID)
	return c, rec
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestUserHandler_GetProfile_Returns200(t *testing.T) {
	h, _ := newUserHandler()
	c, rec := setupUserEcho(http.MethodGet, "/auth/me", "", 1)

	require.NoError(t, h.GetProfile(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp dto.UserProfileResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "Alice", resp.Name)
}

func TestUserHandler_UpdateProfile_Returns200(t *testing.T) {
	h, _ := newUserHandler()
	c, rec := setupUserEcho(http.MethodPut, "/auth/me", `{"full_name":"Alice B"}`, 1)

	require.NoError(t, h.UpdateProfile(c))
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestUserHandler_GetMyActivity_Returns200(t *testing.T) {
	h, _ := newUserHandler()
	c, rec := setupUserEcho(http.MethodGet, "/auth/me/activity", "", 1)

	require.NoError(t, h.GetMyActivity(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	var events []*dto.ActivityEventResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &events))
	assert.Len(t, events, 1)
	assert.Equal(t, "Me", events[0].ActorName)
}
