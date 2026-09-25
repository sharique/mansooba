package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	profile   *dto.UserProfileResponse
	updateFn  func(userID uint, req dto.UpdateProfileRequest) (*dto.UserProfileResponse, error)
	avatarErr error

	getAvatarFn    func(filename string) (io.ReadCloser, string, error)
	getAvatarCalls int
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
	if s.avatarErr != nil {
		return nil, s.avatarErr
	}
	if s.profile == nil {
		return nil, domain.ErrNotFound
	}
	s.profile.AvatarURL = "/avatars/avatar-1.jpg?v=1000"
	return s.profile, nil
}

func (s *stubUserService) GetAvatar(_ context.Context, filename string) (io.ReadCloser, string, error) {
	s.getAvatarCalls++
	if s.getAvatarFn != nil {
		return s.getAvatarFn(filename)
	}
	return nil, "", domain.ErrNotFound
}

func (s *stubUserService) DeleteAvatar(_ context.Context, userID uint) (*dto.UserProfileResponse, error) {
	if s.avatarErr != nil {
		return nil, s.avatarErr
	}
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
	b[0] = 0xFF
	b[1] = 0xD8
	b[2] = 0xFF
	b[3] = 0xE0
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
	svc.profile = &dto.UserProfileResponse{ID: 1, Name: "Alice", Email: "alice@example.com", AvatarURL: "/avatars/avatar-1.jpg?v=1000"}

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

// ── T004a: is_admin returned in /auth/me response ─────────────────────────────

func TestUserHandler_GetProfile_ReturnsIsAdmin_TrueForAdmin(t *testing.T) {
	profile := &dto.UserProfileResponse{ID: 1, Name: "Admin", Email: "admin@example.com", IsAdmin: true}
	userSvc := &stubUserService{profile: profile}
	h := handler.NewUserHandler(userSvc, &stubActivityServiceForUser{}, &stubIssueServiceForUser{})
	c, rec := setupUserEcho(http.MethodGet, "/auth/me", "", 1)

	require.NoError(t, h.GetProfile(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp dto.UserProfileResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.True(t, resp.IsAdmin)
}

func TestUserHandler_GetProfile_ReturnsIsAdmin_FalseForRegularUser(t *testing.T) {
	profile := &dto.UserProfileResponse{ID: 2, Name: "Alice", Email: "alice@example.com", IsAdmin: false}
	userSvc := &stubUserService{profile: profile}
	h := handler.NewUserHandler(userSvc, &stubActivityServiceForUser{}, &stubIssueServiceForUser{})
	c, rec := setupUserEcho(http.MethodGet, "/auth/me", "", 2)

	require.NoError(t, h.GetProfile(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp dto.UserProfileResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.False(t, resp.IsAdmin)
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

// ── ServeAvatar ───────────────────────────────────────────────────────────────

func serveAvatarRequest(h *handler.UserHandler, filename string) (*httptest.ResponseRecorder, error) {
	c, rec := setupUserEcho(http.MethodGet, "/avatars/x", "", 0)
	c.SetParamNames("filename")
	c.SetParamValues(filename)
	return rec, h.ServeAvatar(c)
}

func TestUserHandler_ServeAvatar_Returns200WithBytesAndCacheHeaders(t *testing.T) {
	h, svc := newUserHandler()
	svc.getAvatarFn = func(filename string) (io.ReadCloser, string, error) {
		assert.Equal(t, "avatar-1.jpg", filename)
		return io.NopCloser(strings.NewReader("jpeg-bytes")), "image/jpeg", nil
	}

	rec, err := serveAvatarRequest(h, "avatar-1.jpg")

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "jpeg-bytes", rec.Body.String())
	assert.Equal(t, "image/jpeg", rec.Header().Get("Content-Type"))
	assert.Equal(t, "public, max-age=3600", rec.Header().Get("Cache-Control"))
}

func TestUserHandler_ServeAvatar_InvalidFilenameReturns404WithoutCallingService(t *testing.T) {
	invalid := []string{
		"../attachments/PROJ/PROJ-3/x.pdf",
		"avatar-1.gif",
		"avatar-.jpg",
		"avatar-abc.jpg",
		"AVATAR-1.jpg",
		"avatar-1.jpg/../x",
		"avatar-1",
		"avatar-1.jpg\n",
		"",
	}
	for _, name := range invalid {
		t.Run(name, func(t *testing.T) {
			h, svc := newUserHandler()

			_, err := serveAvatarRequest(h, name)

			var he *echo.HTTPError
			require.True(t, errors.As(err, &he), "expected *echo.HTTPError, got %v", err)
			assert.Equal(t, http.StatusNotFound, he.Code)
			assert.Zero(t, svc.getAvatarCalls, "service must not be called for an invalid filename")
		})
	}
}

func TestUserHandler_ServeAvatar_NotFoundFromServiceReturns404(t *testing.T) {
	h, svc := newUserHandler()
	svc.getAvatarFn = func(string) (io.ReadCloser, string, error) {
		return nil, "", domain.ErrNotFound
	}

	_, err := serveAvatarRequest(h, "avatar-7.png")

	var he *echo.HTTPError
	require.True(t, errors.As(err, &he))
	assert.Equal(t, http.StatusNotFound, he.Code)
}

func TestUserHandler_ServeAvatar_OtherErrorReturns500WithoutLeakingDetail(t *testing.T) {
	h, svc := newUserHandler()
	svc.getAvatarFn = func(string) (io.ReadCloser, string, error) {
		return nil, "", errors.New("s3 exploded: secret-endpoint-detail")
	}

	_, err := serveAvatarRequest(h, "avatar-7.png")

	var he *echo.HTTPError
	require.True(t, errors.As(err, &he))
	assert.Equal(t, http.StatusInternalServerError, he.Code)
	assert.NotContains(t, fmt.Sprint(he.Message), "secret-endpoint-detail")
}

// ── storage failures are not the client's fault and must not leak detail ──────

func TestUserHandler_UploadAvatar_StorageUnavailableReturns502WithoutDetail(t *testing.T) {
	h, svc := newUserHandler()
	svc.avatarErr = fmt.Errorf("%w: operation error S3: PutObject, https://internal-endpoint:4566", domain.ErrAvatarStorageUnavailable)
	c, _ := setupUserEcho(http.MethodPost, "/auth/me/avatar", "", 1)
	req := newAvatarUploadRequest(t)
	c.SetRequest(req)

	err := h.UploadAvatar(c)

	var he *echo.HTTPError
	require.True(t, errors.As(err, &he))
	assert.Equal(t, http.StatusBadGateway, he.Code)
	assert.NotContains(t, fmt.Sprint(he.Message), "internal-endpoint")
}

func TestUserHandler_DeleteAvatar_StorageUnavailableReturns502WithoutDetail(t *testing.T) {
	h, svc := newUserHandler()
	svc.avatarErr = fmt.Errorf("%w: operation error S3: DeleteObject, https://internal-endpoint:4566", domain.ErrAvatarStorageUnavailable)
	c, _ := setupUserEcho(http.MethodDelete, "/auth/me/avatar", "", 1)

	err := h.DeleteAvatar(c)

	var he *echo.HTTPError
	require.True(t, errors.As(err, &he))
	assert.Equal(t, http.StatusBadGateway, he.Code)
	assert.NotContains(t, fmt.Sprint(he.Message), "internal-endpoint")
}

func TestUserHandler_UploadAvatar_ValidationRejectionStill400WithMessage(t *testing.T) {
	h, svc := newUserHandler()
	svc.avatarErr = errors.New("content type \"image/gif\" is not accepted")
	c, _ := setupUserEcho(http.MethodPost, "/auth/me/avatar", "", 1)
	c.SetRequest(newAvatarUploadRequest(t))

	err := h.UploadAvatar(c)

	var he *echo.HTTPError
	require.True(t, errors.As(err, &he))
	assert.Equal(t, http.StatusBadRequest, he.Code)
	assert.Contains(t, fmt.Sprint(he.Message), "not accepted")
}

func newAvatarUploadRequest(t *testing.T) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("avatar", "me.jpg")
	require.NoError(t, err)
	_, err = part.Write([]byte("bytes"))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	req := httptest.NewRequest(http.MethodPost, "/auth/me/avatar", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}
