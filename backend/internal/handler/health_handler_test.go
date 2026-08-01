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

// stubLokiPinger is a test double for handler.LokiPinger.
type stubLokiPinger struct {
	readyErr error
}

func (s *stubLokiPinger) Ready(_ context.Context) error {
	return s.readyErr
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

func TestHealthHandler_Check_ReportsLokiOK_WhenConfiguredAndReachable(t *testing.T) {
	h := handler.NewHealthHandler(&stubDBPinger{}).WithLoki(&stubLokiPinger{})
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
	if body["loki"] != "ok" {
		t.Errorf("expected loki=ok, got %v", body["loki"])
	}
}

func TestHealthHandler_Check_ReportsLokiError_ButStaysOverall200_WhenLokiDown(t *testing.T) {
	// A Loki outage is reported, but must NOT flip the overall status to 503 —
	// the application keeps operating normally without it (FR-012, best-effort
	// logging). This is the key behavioral difference from the DB check.
	lokiErr := errors.New("connection refused")
	h := handler.NewHealthHandler(&stubDBPinger{}).WithLoki(&stubLokiPinger{readyErr: lokiErr})
	e := newHealthEcho(h)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (Loki being down must not degrade overall status), got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status=ok despite Loki being down, got %v", body["status"])
	}
	if body["loki"] != "error" {
		t.Errorf("expected loki=error, got %v", body["loki"])
	}
}

func TestHealthHandler_Check_OmitsLokiField_WhenNotConfigured(t *testing.T) {
	h := handler.NewHealthHandler(&stubDBPinger{}) // WithLoki never called
	e := newHealthEcho(h)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if _, ok := body["loki"]; ok {
		t.Errorf("expected loki field to be absent when WithLoki was never called, got %v", body["loki"])
	}
}
