package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/sharique/jira-go/internal/domain"
	"github.com/sharique/jira-go/internal/dto"
	"github.com/sharique/jira-go/internal/handler"
	"github.com/sharique/jira-go/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── stubCommentService ────────────────────────────────────────────────────────

type stubCommentService struct {
	comments []*dto.CommentResponse
	createFn func(issueID, callerID uint, req dto.CreateCommentRequest) (*dto.CommentResponse, error)
}

func (s *stubCommentService) Create(ctx context.Context, issueID, callerID uint, req dto.CreateCommentRequest) (*dto.CommentResponse, error) {
	if s.createFn != nil {
		return s.createFn(issueID, callerID, req)
	}
	resp := &dto.CommentResponse{ID: 1, IssueID: issueID, AuthorID: callerID, Body: req.Body}
	s.comments = append(s.comments, resp)
	return resp, nil
}

func (s *stubCommentService) List(ctx context.Context, issueID, callerID uint) ([]*dto.CommentResponse, error) {
	return s.comments, nil
}

func (s *stubCommentService) Update(ctx context.Context, commentID, callerID uint, req dto.UpdateCommentRequest) (*dto.CommentResponse, error) {
	return &dto.CommentResponse{ID: commentID, Body: req.Body}, nil
}

func (s *stubCommentService) Delete(ctx context.Context, commentID, callerID uint) error {
	return nil
}

// Ensure interface is satisfied at compile time.
var _ service.CommentService = (*stubCommentService)(nil)

// ── stubActivityService ───────────────────────────────────────────────────────

type stubActivitySvcHandler struct{}

func (s *stubActivitySvcHandler) Record(ctx context.Context, e *domain.ActivityEvent) error {
	return nil
}
func (s *stubActivitySvcHandler) ListByIssue(ctx context.Context, issueID uint) ([]*domain.ActivityEvent, error) {
	return []*domain.ActivityEvent{
		{ID: 1, IssueID: issueID, Kind: domain.ActivityStatusChanged, OldValue: "todo", NewValue: "in_progress"},
	}, nil
}

var _ service.ActivityService = (*stubActivitySvcHandler)(nil)

// ── helpers ───────────────────────────────────────────────────────────────────

func newCommentEchoCtx(method, path, body string, issueID string) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userID", uint(42))
	c.SetParamNames("id")
	c.SetParamValues(issueID)
	return c, rec
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestCommentHandler_Create_Returns201(t *testing.T) {
	svc := &stubCommentService{}
	h := handler.NewCommentHandler(svc)
	c, rec := newCommentEchoCtx(http.MethodPost, "/issues/1/comments", `{"body":"hello"}`, "1")

	require.NoError(t, h.Create(c))
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp dto.CommentResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "hello", resp.Body)
}

func TestCommentHandler_List_Returns200(t *testing.T) {
	svc := &stubCommentService{comments: []*dto.CommentResponse{{ID: 1, Body: "hi"}}}
	h := handler.NewCommentHandler(svc)
	c, rec := newCommentEchoCtx(http.MethodGet, "/issues/1/comments", "", "1")

	require.NoError(t, h.List(c))
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCommentHandler_Create_ForbiddenPropagates(t *testing.T) {
	svc := &stubCommentService{
		createFn: func(_, _ uint, _ dto.CreateCommentRequest) (*dto.CommentResponse, error) {
			return nil, domain.ErrForbidden
		},
	}
	h := handler.NewCommentHandler(svc)
	c, rec := newCommentEchoCtx(http.MethodPost, "/issues/1/comments", `{"body":"x"}`, "1")

	err := h.Create(c)
	var httpErr *echo.HTTPError
	require.ErrorAs(t, err, &httpErr)
	assert.Equal(t, http.StatusForbidden, httpErr.Code)
	_ = rec
}

func TestActivityHandler_List_Returns200(t *testing.T) {
	actSvc := &stubActivitySvcHandler{}
	h := handler.NewActivityHandler(actSvc)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/issues/1/activity", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userID", uint(42))
	c.SetParamNames("id")
	c.SetParamValues("1")

	require.NoError(t, h.ListByIssue(c))
	assert.Equal(t, http.StatusOK, rec.Code)
}
