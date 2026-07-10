package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/handler"
	"github.com/sharique/mansooba/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── stubAttachmentService ───────────────────────────────────────────────────

type stubAttachmentService struct {
	uploadFn   func(issueID, callerID uint, files []dto.AttachmentUploadFile) (*dto.AttachmentUploadResult, error)
	listFn     func(issueID, callerID uint) ([]*dto.AttachmentResponse, error)
	downloadFn func(issueID, attachmentID, callerID uint) (string, string, error)
	deleteFn   func(issueID, attachmentID, callerID uint) error
}

func (s *stubAttachmentService) Upload(_ context.Context, issueID, callerID uint, files []dto.AttachmentUploadFile) (*dto.AttachmentUploadResult, error) {
	return s.uploadFn(issueID, callerID, files)
}

func (s *stubAttachmentService) List(_ context.Context, issueID, callerID uint) ([]*dto.AttachmentResponse, error) {
	return s.listFn(issueID, callerID)
}

func (s *stubAttachmentService) GenerateDownloadURL(_ context.Context, issueID, attachmentID, callerID uint) (string, string, error) {
	return s.downloadFn(issueID, attachmentID, callerID)
}

func (s *stubAttachmentService) Delete(_ context.Context, issueID, attachmentID, callerID uint) error {
	return s.deleteFn(issueID, attachmentID, callerID)
}

var _ service.AttachmentService = (*stubAttachmentService)(nil)

// ── helpers ───────────────────────────────────────────────────────────────────

func buildAttachmentMultipartRequest(t *testing.T, files map[string][]byte) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for filename, data := range files {
		fw, err := w.CreateFormFile("files", filename)
		require.NoError(t, err)
		_, err = fw.Write(data)
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	req := httptest.NewRequest(http.MethodPost, "/api/v1/issues/1/attachments", &buf)
	req.Header.Set(echo.HeaderContentType, w.FormDataContentType())
	return req
}

func newAttachmentEchoCtx(method, path string, issueID, aid string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userID", uint(42))
	if aid != "" {
		c.SetParamNames("id", "aid")
		c.SetParamValues(issueID, aid)
	} else {
		c.SetParamNames("id")
		c.SetParamValues(issueID)
	}
	return c, rec
}

// ── Upload ────────────────────────────────────────────────────────────────────

func TestAttachmentHandler_Upload_Returns200(t *testing.T) {
	svc := &stubAttachmentService{
		uploadFn: func(issueID, callerID uint, files []dto.AttachmentUploadFile) (*dto.AttachmentUploadResult, error) {
			return &dto.AttachmentUploadResult{
				Uploaded: []dto.AttachmentResponse{{ID: 1, IssueID: issueID, Filename: files[0].Filename}},
				Rejected: []dto.AttachmentRejection{},
			}, nil
		},
	}
	h := handler.NewAttachmentHandler(svc)

	req := buildAttachmentMultipartRequest(t, map[string][]byte{"a.png": []byte("data")})
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)
	c.Set("userID", uint(42))
	c.SetParamNames("id")
	c.SetParamValues("1")

	require.NoError(t, h.Upload(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	var result dto.AttachmentUploadResult
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	assert.Len(t, result.Uploaded, 1)
	assert.Equal(t, "a.png", result.Uploaded[0].Filename)
}

func TestAttachmentHandler_Upload_ViewerForbidden(t *testing.T) {
	svc := &stubAttachmentService{
		uploadFn: func(issueID, callerID uint, files []dto.AttachmentUploadFile) (*dto.AttachmentUploadResult, error) {
			return nil, domain.ErrForbidden
		},
	}
	h := handler.NewAttachmentHandler(svc)

	req := buildAttachmentMultipartRequest(t, map[string][]byte{"a.png": []byte("data")})
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)
	c.Set("userID", uint(7))
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := h.Upload(c)
	require.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusForbidden, he.Code)
}

func TestAttachmentHandler_Upload_IssueNotFound(t *testing.T) {
	svc := &stubAttachmentService{
		uploadFn: func(issueID, callerID uint, files []dto.AttachmentUploadFile) (*dto.AttachmentUploadResult, error) {
			return nil, domain.ErrNotFound
		},
	}
	h := handler.NewAttachmentHandler(svc)
	req := buildAttachmentMultipartRequest(t, map[string][]byte{"a.png": []byte("data")})
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)
	c.Set("userID", uint(42))
	c.SetParamNames("id")
	c.SetParamValues("999")

	err := h.Upload(c)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusNotFound, he.Code)
}

func TestAttachmentHandler_Upload_CapReachedReturns409(t *testing.T) {
	svc := &stubAttachmentService{
		uploadFn: func(issueID, callerID uint, files []dto.AttachmentUploadFile) (*dto.AttachmentUploadResult, error) {
			return nil, domain.ErrAttachmentCapReached
		},
	}
	h := handler.NewAttachmentHandler(svc)
	req := buildAttachmentMultipartRequest(t, map[string][]byte{"a.png": []byte("data")})
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)
	c.Set("userID", uint(42))
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := h.Upload(c)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusConflict, he.Code)
}

func TestAttachmentHandler_Upload_NoFilesReturns400(t *testing.T) {
	svc := &stubAttachmentService{}
	h := handler.NewAttachmentHandler(svc)
	req := buildAttachmentMultipartRequest(t, map[string][]byte{})
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)
	c.Set("userID", uint(42))
	c.SetParamNames("id")
	c.SetParamValues("1")

	err := h.Upload(c)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, he.Code)
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestAttachmentHandler_List_Returns200(t *testing.T) {
	svc := &stubAttachmentService{
		listFn: func(issueID, callerID uint) ([]*dto.AttachmentResponse, error) {
			return []*dto.AttachmentResponse{{ID: 1, IssueID: issueID, Filename: "a.png"}}, nil
		},
	}
	h := handler.NewAttachmentHandler(svc)
	c, rec := newAttachmentEchoCtx(http.MethodGet, "/api/v1/issues/1/attachments", "1", "")

	require.NoError(t, h.List(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp dto.AttachmentListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Len(t, resp.Attachments, 1)
}

func TestAttachmentHandler_List_NonMemberForbidden(t *testing.T) {
	svc := &stubAttachmentService{
		listFn: func(issueID, callerID uint) ([]*dto.AttachmentResponse, error) {
			return nil, domain.ErrForbidden
		},
	}
	h := handler.NewAttachmentHandler(svc)
	c, _ := newAttachmentEchoCtx(http.MethodGet, "/api/v1/issues/1/attachments", "1", "")

	err := h.List(c)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusForbidden, he.Code)
}

// ── Download ──────────────────────────────────────────────────────────────────

func TestAttachmentHandler_Download_Returns200WithPresignedURL(t *testing.T) {
	svc := &stubAttachmentService{
		downloadFn: func(issueID, attachmentID, callerID uint) (string, string, error) {
			return "https://s3.example.com/issues/1/x.png?X-Amz-Signature=abc", "x.png", nil
		},
	}
	h := handler.NewAttachmentHandler(svc)
	c, rec := newAttachmentEchoCtx(http.MethodGet, "/api/v1/issues/1/attachments/5/download", "1", "5")

	require.NoError(t, h.Download(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp dto.AttachmentDownloadResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Contains(t, resp.URL, "X-Amz-Signature")
	assert.Equal(t, "x.png", resp.Filename)
}

func TestAttachmentHandler_Download_NonMemberForbidden_NoURLReturned(t *testing.T) {
	svc := &stubAttachmentService{
		downloadFn: func(issueID, attachmentID, callerID uint) (string, string, error) {
			return "", "", domain.ErrForbidden
		},
	}
	h := handler.NewAttachmentHandler(svc)
	c, rec := newAttachmentEchoCtx(http.MethodGet, "/api/v1/issues/1/attachments/5/download", "1", "5")

	err := h.Download(c)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusForbidden, he.Code)
	assert.Empty(t, rec.Body.String(), "no presigned URL must ever be returned for a denied caller")
}

func TestAttachmentHandler_Download_StorageUnavailableReturns502(t *testing.T) {
	svc := &stubAttachmentService{
		downloadFn: func(issueID, attachmentID, callerID uint) (string, string, error) {
			return "", "", errors.Join(domain.ErrAttachmentStorageUnavailable, errors.New("connection refused"))
		},
	}
	h := handler.NewAttachmentHandler(svc)
	c, _ := newAttachmentEchoCtx(http.MethodGet, "/api/v1/issues/1/attachments/5/download", "1", "5")

	err := h.Download(c)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadGateway, he.Code)
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestAttachmentHandler_Delete_Returns200(t *testing.T) {
	svc := &stubAttachmentService{
		deleteFn: func(issueID, attachmentID, callerID uint) error { return nil },
	}
	h := handler.NewAttachmentHandler(svc)
	c, rec := newAttachmentEchoCtx(http.MethodDelete, "/api/v1/issues/1/attachments/5", "1", "5")

	require.NoError(t, h.Delete(c))
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestAttachmentHandler_Delete_RegularMemberForbidden(t *testing.T) {
	svc := &stubAttachmentService{
		deleteFn: func(issueID, attachmentID, callerID uint) error { return domain.ErrForbidden },
	}
	h := handler.NewAttachmentHandler(svc)
	c, _ := newAttachmentEchoCtx(http.MethodDelete, "/api/v1/issues/1/attachments/5", "1", "5")

	err := h.Delete(c)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusForbidden, he.Code)
}

func TestAttachmentHandler_Delete_NotFound(t *testing.T) {
	svc := &stubAttachmentService{
		deleteFn: func(issueID, attachmentID, callerID uint) error { return domain.ErrNotFound },
	}
	h := handler.NewAttachmentHandler(svc)
	c, _ := newAttachmentEchoCtx(http.MethodDelete, "/api/v1/issues/1/attachments/999", "1", "999")

	err := h.Delete(c)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusNotFound, he.Code)
}

func TestAttachmentHandler_Delete_StorageUnavailableReturns502(t *testing.T) {
	svc := &stubAttachmentService{
		deleteFn: func(issueID, attachmentID, callerID uint) error {
			return errors.Join(domain.ErrAttachmentStorageUnavailable, errors.New("connection refused"))
		},
	}
	h := handler.NewAttachmentHandler(svc)
	c, _ := newAttachmentEchoCtx(http.MethodDelete, "/api/v1/issues/1/attachments/5", "1", "5")

	err := h.Delete(c)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadGateway, he.Code)
}
