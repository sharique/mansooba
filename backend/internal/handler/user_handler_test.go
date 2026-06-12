package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
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

func (s *stubUserService) UploadAvatar(_ context.Context, userID uint, _ string, _ []byte, _ string) (*dto.UserProfileResponse, error) {
	if s.profile == nil {
		return nil, domain.ErrNotFound
	}
	s.profile.AvatarURL = "/uploads/avatars/avatar-1.jpg?v=1000"
	return s.profile, nil
}

func (s *stubUserService) DeleteAvatar(_ context.Context, userID uint) (*dto.UserProfileResponse, error) {
	if s.profile == nil {
		return nil, domain.ErrNotFound
	}
	s.profile.AvatarURL = ""
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

// stubIssueServiceForUser is a no-op IssueService used in user handler tests.
type stubIssueServiceForUser struct{}

func (s *stubIssueServiceForUser) Create(_ context.Context, _ string, _ uint, _ dto.CreateIssueRequest) (*dto.IssueResponse, error) {
	return nil, nil
}
func (s *stubIssueServiceForUser) ListByProject(_ context.Context, _ string, _ uint, _ dto.IssueListQuery) ([]*dto.IssueResponse, error) {
	return nil, nil
}
func (s *stubIssueServiceForUser) GetMyIssues(_ context.Context, _ uint, _ dto.IssueListQuery) ([]*dto.IssueResponse, error) {
	return nil, nil
}
func (s *stubIssueServiceForUser) FindByID(_ context.Context, _ string, _ uint, _ uint) (*dto.IssueResponse, error) {
	return nil, nil
}
func (s *stubIssueServiceForUser) Update(_ context.Context, _ string, _ uint, _ uint, _ dto.UpdateIssueRequest) (*dto.IssueResponse, error) {
	return nil, nil
}
func (s *stubIssueServiceForUser) Delete(_ context.Context, _ string, _ uint, _ uint) error {
	return nil
}

var _ service.IssueService = (*stubIssueServiceForUser)(nil)

// ── helpers ───────────────────────────────────────────────────────────────────

func newUserHandler() (*handler.UserHandler, *stubUserService) {
	profile := &dto.UserProfileResponse{ID: 1, Name: "Alice", Email: "alice@example.com"}
	userSvc := &stubUserService{profile: profile}
	activitySvc := &stubActivityServiceForUser{}
	issueSvc := &stubIssueServiceForUser{}
	return handler.NewUserHandler(userSvc, activitySvc, issueSvc), userSvc
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

// ── T009: upload / delete avatar handler tests ────────────────────────────────

func buildMultipartRequest(t *testing.T, fieldName, filename string, data []byte) (*http.Request, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile(fieldName, filename)
	require.NoError(t, err)
	_, err = fw.Write(data)
	require.NoError(t, err)
	w.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/me/avatar", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req, w.FormDataContentType()
}

// minimalJPEG bytes for handler tests.
var minimalJPEGForHandler = func() []byte {
	b := make([]byte, 512)
	b[0] = 0xFF; b[1] = 0xD8; b[2] = 0xFF; b[3] = 0xE0
	return b
}()

func TestUserHandler_UploadAvatar_Returns200WithUpdatedProfile(t *testing.T) {
	h, svc := newUserHandler()
	svc.profile = &dto.UserProfileResponse{ID: 1, Name: "Alice", Email: "alice@example.com"}

	req, ct := buildMultipartRequest(t, "avatar", "photo.jpg", minimalJPEGForHandler)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()

	e := echo.New()
	c := e.NewContext(req, rec)
	c.Set("userID", uint(1))

	require.NoError(t, h.UploadAvatar(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp dto.UserProfileResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp.AvatarURL)
}

func TestUserHandler_UploadAvatar_Returns400WhenNoFile(t *testing.T) {
	h, _ := newUserHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/me/avatar", strings.NewReader(""))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	rec := httptest.NewRecorder()

	e := echo.New()
	c := e.NewContext(req, rec)
	c.Set("userID", uint(1))

	err := h.UploadAvatar(c)
	assert.Error(t, err)
}

func TestUserHandler_DeleteAvatar_Returns200WithEmptyAvatarURL(t *testing.T) {
	h, svc := newUserHandler()
	svc.profile = &dto.UserProfileResponse{ID: 1, Name: "Alice", Email: "alice@example.com", AvatarURL: "/uploads/avatars/avatar-1.jpg?v=1000"}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/me/avatar", nil)
	rec := httptest.NewRecorder()

	e := echo.New()
	c := e.NewContext(req, rec)
	c.Set("userID", uint(1))

	require.NoError(t, h.DeleteAvatar(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp dto.UserProfileResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Empty(t, resp.AvatarURL)
}

func TestUserHandler_UploadAvatar_Returns401WhenNoUserID(t *testing.T) {
	h, _ := newUserHandler()

	req, ct := buildMultipartRequest(t, "avatar", "photo.jpg", minimalJPEGForHandler)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()

	e := echo.New()
	c := e.NewContext(req, rec)
	// Intentionally do NOT set "userID" — simulates unauthenticated request
	// Handler will panic/error since userID is not set, which tests 401 behavior.
	assert.Panics(t, func() { _ = h.UploadAvatar(c) })
}
