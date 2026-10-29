package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/handler"
)

type stubBoardService struct {
	getBoardFn func(ctx context.Context, projectKey string, callerID uint) (*dto.BoardResponse, error)
}

func (s *stubBoardService) GetBoard(ctx context.Context, projectKey string, callerID uint) (*dto.BoardResponse, error) {
	return s.getBoardFn(ctx, projectKey, callerID)
}

func newBoardEcho(h *handler.BoardHandler) *echo.Echo {
	e := newEcho()
	api := e.Group("/api/v1", func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			c.Set("userID", uint(1))
			return next(c)
		}
	})
	api.GET("/projects/:key/board", h.GetBoard)
	return e
}

func TestBoardHandler_GetBoard_Returns200(t *testing.T) {
	svc := &stubBoardService{
		getBoardFn: func(_ context.Context, _ string, _ uint) (*dto.BoardResponse, error) {
			return &dto.BoardResponse{
				Columns: []dto.BoardColumn{
					{Status: domain.IssueStatusTodo, Issues: []dto.BoardIssue{{IssueResponse: dto.IssueResponse{ID: 1, Key: "PROJ-1"}}}},
					{Status: domain.IssueStatusInProgress, Issues: []dto.BoardIssue{}},
					{Status: domain.IssueStatusInReview, Issues: []dto.BoardIssue{}},
					{Status: domain.IssueStatusDone, Issues: []dto.BoardIssue{}},
				},
			}, nil
		},
	}
	e := newBoardEcho(handler.NewBoardHandler(svc))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/PROJ/board", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp dto.BoardResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if len(resp.Columns) != 4 {
		t.Errorf("expected 4 columns, got %d", len(resp.Columns))
	}
	if resp.Columns[0].Status != domain.IssueStatusTodo {
		t.Errorf("first column should be todo, got %s", resp.Columns[0].Status)
	}
}

func TestBoardHandler_GetBoard_Returns403_ForNonMember(t *testing.T) {
	svc := &stubBoardService{
		getBoardFn: func(_ context.Context, _ string, _ uint) (*dto.BoardResponse, error) {
			return nil, domain.ErrForbidden
		},
	}
	e := newBoardEcho(handler.NewBoardHandler(svc))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/PROJ/board", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestBoardHandler_GetBoard_Returns404_ForMissingProject(t *testing.T) {
	svc := &stubBoardService{
		getBoardFn: func(_ context.Context, _ string, _ uint) (*dto.BoardResponse, error) {
			return nil, domain.ErrNotFound
		},
	}
	e := newBoardEcho(handler.NewBoardHandler(svc))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/FAKE/board", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestBoardHandler_GetBoard_ReportsLockedOnEveryCard(t *testing.T) {
	svc := &stubBoardService{
		getBoardFn: func(_ context.Context, _ string, _ uint) (*dto.BoardResponse, error) {
			return &dto.BoardResponse{Columns: []dto.BoardColumn{{
				Status: domain.IssueStatusDone,
				Issues: []dto.BoardIssue{
					{IssueResponse: dto.IssueResponse{ID: 1, Key: "PROJ-1"}, Locked: true},
					{IssueResponse: dto.IssueResponse{ID: 2, Key: "PROJ-2"}},
				},
			}}}, nil
		},
	}
	e := newBoardEcho(handler.NewBoardHandler(svc))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects/PROJ/board", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var raw struct {
		Columns []struct {
			Issues []map[string]any `json:"issues"`
		} `json:"columns"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	cards := raw.Columns[0].Issues
	if len(cards) != 2 {
		t.Fatalf("expected 2 cards, got %d", len(cards))
	}
	if cards[0]["locked"] != true {
		t.Errorf("first card: locked = %v, want true", cards[0]["locked"])
	}
	if v, present := cards[1]["locked"]; !present || v != false {
		t.Errorf("second card: locked = %v (present %v), want an explicit false", v, present)
	}
	if cards[0]["key"] != "PROJ-1" {
		t.Errorf("issue fields must stay at the top level of the card, got %v", cards[0])
	}
}
