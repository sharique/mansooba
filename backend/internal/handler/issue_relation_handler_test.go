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

// ── stubIssueRelationService ──────────────────────────────────────────────────

type stubIssueRelationService struct {
	listFn   func(issueID uint) ([]*dto.RelationResponse, error)
	createFn func(issueID, userID uint, req dto.CreateRelationRequest) (*dto.RelationResponse, error)
	deleteFn func(relationID, userID uint) error
}

func (s *stubIssueRelationService) List(_ context.Context, issueID uint) ([]*dto.RelationResponse, error) {
	return s.listFn(issueID)
}

func (s *stubIssueRelationService) Create(_ context.Context, issueID, userID uint, req dto.CreateRelationRequest) (*dto.RelationResponse, error) {
	return s.createFn(issueID, userID, req)
}

func (s *stubIssueRelationService) Delete(_ context.Context, relationID, userID uint) error {
	return s.deleteFn(relationID, userID)
}

var _ service.IssueRelationService = (*stubIssueRelationService)(nil)

// ── helpers ───────────────────────────────────────────────────────────────────

func newRelationEcho(h *handler.IssueRelationHandler) *echo.Echo {
	e := newEcho()
	api := e.Group("/api/v1", func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("userID", uint(1))
			return next(c)
		}
	})
	issueRels := api.Group("/issues/:id/relations")
	issueRels.GET("", h.List)
	issueRels.POST("", h.Create)
	issueRels.DELETE("/:rid", h.Delete)
	return e
}

func relatedIssue() dto.RelatedIssueInfo {
	return dto.RelatedIssueInfo{ID: 42, Key: "PROJ-7", Title: "Fix login", Status: "in_progress"}
}

// ── T023: GET /api/v1/issues/:id/relations ────────────────────────────────────

func TestIssueRelationHandler_List_ReturnsEmptyArray(t *testing.T) {
	svc := &stubIssueRelationService{
		listFn: func(_ uint) ([]*dto.RelationResponse, error) { return []*dto.RelationResponse{}, nil },
	}
	h := handler.NewIssueRelationHandler(svc)
	e := newRelationEcho(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/issues/1/relations", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp []*dto.RelationResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Empty(t, resp)
}

func TestIssueRelationHandler_List_ReturnsAllRelations(t *testing.T) {
	relations := []*dto.RelationResponse{
		{ID: 1, RelationType: "blocks", RelatedIssue: relatedIssue()},
		{ID: 3, RelationType: "relates_to", RelatedIssue: relatedIssue()},
	}
	svc := &stubIssueRelationService{
		listFn: func(_ uint) ([]*dto.RelationResponse, error) { return relations, nil },
	}
	h := handler.NewIssueRelationHandler(svc)
	e := newRelationEcho(h)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/issues/1/relations", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp []*dto.RelationResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Len(t, resp, 2)
	assert.Equal(t, "blocks", resp[0].RelationType)
}

// ── T023: POST /api/v1/issues/:id/relations ───────────────────────────────────

func TestIssueRelationHandler_Create_Blocks_Returns201(t *testing.T) {
	svc := &stubIssueRelationService{
		createFn: func(_, _ uint, req dto.CreateRelationRequest) (*dto.RelationResponse, error) {
			return &dto.RelationResponse{ID: 5, RelationType: req.RelationType, RelatedIssue: relatedIssue()}, nil
		},
	}
	h := handler.NewIssueRelationHandler(svc)
	e := newRelationEcho(h)

	body := `{"target_issue_id":42,"relation_type":"blocks"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/issues/1/relations", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	var resp dto.RelationResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "blocks", resp.RelationType)
}

func TestIssueRelationHandler_Create_RelatesTo_Returns201(t *testing.T) {
	svc := &stubIssueRelationService{
		createFn: func(_, _ uint, req dto.CreateRelationRequest) (*dto.RelationResponse, error) {
			return &dto.RelationResponse{ID: 6, RelationType: req.RelationType, RelatedIssue: relatedIssue()}, nil
		},
	}
	h := handler.NewIssueRelationHandler(svc)
	e := newRelationEcho(h)

	body := `{"target_issue_id":42,"relation_type":"relates_to"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/issues/1/relations", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestIssueRelationHandler_Create_SelfLink_Returns400(t *testing.T) {
	svc := &stubIssueRelationService{
		createFn: func(_, _ uint, _ dto.CreateRelationRequest) (*dto.RelationResponse, error) {
			return nil, service.ErrSelfRelation
		},
	}
	h := handler.NewIssueRelationHandler(svc)
	e := newRelationEcho(h)

	body := `{"target_issue_id":1,"relation_type":"blocks"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/issues/1/relations", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestIssueRelationHandler_Create_CircularBlocks_Returns400(t *testing.T) {
	svc := &stubIssueRelationService{
		createFn: func(_, _ uint, _ dto.CreateRelationRequest) (*dto.RelationResponse, error) {
			return nil, service.ErrCircularRelation
		},
	}
	h := handler.NewIssueRelationHandler(svc)
	e := newRelationEcho(h)

	body := `{"target_issue_id":42,"relation_type":"blocks"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/issues/1/relations", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestIssueRelationHandler_Create_CrossProject_Returns400(t *testing.T) {
	svc := &stubIssueRelationService{
		createFn: func(_, _ uint, _ dto.CreateRelationRequest) (*dto.RelationResponse, error) {
			return nil, service.ErrCrossProjectRelation
		},
	}
	h := handler.NewIssueRelationHandler(svc)
	e := newRelationEcho(h)

	body := `{"target_issue_id":99,"relation_type":"relates_to"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/issues/1/relations", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestIssueRelationHandler_Create_DuplicateRelation_Returns400(t *testing.T) {
	svc := &stubIssueRelationService{
		createFn: func(_, _ uint, _ dto.CreateRelationRequest) (*dto.RelationResponse, error) {
			return nil, service.ErrDuplicateRelation
		},
	}
	h := handler.NewIssueRelationHandler(svc)
	e := newRelationEcho(h)

	body := `{"target_issue_id":42,"relation_type":"blocks"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/issues/1/relations", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestIssueRelationHandler_Create_InvalidType_Returns400(t *testing.T) {
	svc := &stubIssueRelationService{
		createFn: func(_, _ uint, _ dto.CreateRelationRequest) (*dto.RelationResponse, error) {
			return nil, service.ErrInvalidRelationType
		},
	}
	h := handler.NewIssueRelationHandler(svc)
	e := newRelationEcho(h)

	body := `{"target_issue_id":42,"relation_type":"is_blocked_by"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/issues/1/relations", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ── T023: DELETE /api/v1/issues/:id/relations/:rid ───────────────────────────

func TestIssueRelationHandler_Delete_Returns204(t *testing.T) {
	svc := &stubIssueRelationService{
		deleteFn: func(_, _ uint) error { return nil },
	}
	h := handler.NewIssueRelationHandler(svc)
	e := newRelationEcho(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/issues/1/relations/5", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestIssueRelationHandler_Delete_NotFound_Returns404(t *testing.T) {
	svc := &stubIssueRelationService{
		deleteFn: func(_, _ uint) error { return service.ErrRelationNotFound },
	}
	h := handler.NewIssueRelationHandler(svc)
	e := newRelationEcho(h)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/issues/1/relations/99", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}
