package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/sharique/mansooba/internal/handler"
)

// stubDBPinger is a test double for handler.DBPinger.
type stubDBPinger struct {
	pingErr error
}

func (s *stubDBPinger) PingContext(_ context.Context) error {
	return s.pingErr
}

func newHealthEcho(h *handler.HealthHandler) *echo.Echo {
	e := newEcho()
	e.GET("/health", h.Check)
	return e
}

func TestHealthHandler_Check_Returns200_WhenDBHealthy(t *testing.T) {
	h := handler.NewHealthHandler(&stubDBPinger{pingErr: nil})
	e := newHealthEcho(h)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", body["status"])
	}
	if body["db"] != "ok" {
		t.Errorf("expected db=ok, got %v", body["db"])
	}
	if _, ok := body["db_latency_ms"]; !ok {
		t.Error("expected db_latency_ms to be present in healthy response")
	}
	if _, ok := body["error"]; ok {
		t.Error("error field should not appear in healthy response")
	}
}

func TestHealthHandler_Check_Returns503_WhenDBUnreachable(t *testing.T) {
	dbErr := errors.New("dial tcp: connection refused")
	h := handler.NewHealthHandler(&stubDBPinger{pingErr: dbErr})
	e := newHealthEcho(h)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", rec.Code, rec.Body.String())
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if body["status"] != "degraded" {
		t.Errorf("expected status=degraded, got %v", body["status"])
	}
	if body["db"] != "error" {
		t.Errorf("expected db=error, got %v", body["db"])
	}
	if body["error"] != dbErr.Error() {
		t.Errorf("expected error=%q, got %v", dbErr.Error(), body["error"])
	}
	if _, ok := body["db_latency_ms"]; ok {
		t.Error("db_latency_ms should not appear in degraded response")
	}
}
