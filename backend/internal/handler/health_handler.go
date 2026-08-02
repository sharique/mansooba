package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// DBPinger is the subset of *sql.DB required by HealthHandler.
// Accepting an interface keeps the handler testable without a real database.
type DBPinger interface {
	PingContext(ctx context.Context) error
}

// LokiPinger is the subset of lokiclient.Client required by HealthHandler
// (011-system-logs, ADR-031). Loki is this feature's first new long-running
// service (Constitution V requires a health check for every one); nil is a
// valid value (e.g. in tests, or if System Logs is ever made optional) and
// simply omits the loki field from the response rather than checking it.
type LokiPinger interface {
	Ready(ctx context.Context) error
}

// HealthHandler returns a DB-aware (and, since 011-system-logs, Loki-aware)
// liveness/readiness response.
type HealthHandler struct {
	db   DBPinger
	loki LokiPinger
}

// NewHealthHandler creates a HealthHandler backed by the given DBPinger. Loki
// is not checked unless WithLoki is also called.
func NewHealthHandler(db DBPinger) *HealthHandler {
	return &HealthHandler{db: db}
}

// WithLoki adds a Loki reachability check to the health response
// (011-system-logs, Constitution V). Returns the same handler for chaining.
func (h *HealthHandler) WithLoki(loki LokiPinger) *HealthHandler {
	h.loki = loki
	return h
}

// healthResponse is the JSON shape returned by Check.
// DBLatencyMs is a pointer so that the field is present (even at 0ms) in the
// healthy response but absent (nil → omitempty) in the degraded response.
// Loki is omitted entirely when no LokiPinger is configured (WithLoki was
// never called) — distinct from "ok"/"error", which mean the check ran.
type healthResponse struct {
	Status      string `json:"status"`
	DB          string `json:"db"`
	DBLatencyMs *int64 `json:"db_latency_ms,omitempty"`
	Loki        string `json:"loki,omitempty"`
	Error       string `json:"error,omitempty"`
}

// Check godoc
// @Summary      Health check
// @Description  Returns overall service status, database connectivity, and (011-system-logs) Loki reachability. Returns 200 when healthy, 503 when the database is unreachable. A Loki outage is reported (loki=error) but does NOT flip the overall status to 503 — the application continues operating normally without it (FR-012, best-effort logging).
// @Tags         infra
// @Produce      json
// @Success      200 {object} handler.healthResponse
// @Failure      503 {object} handler.healthResponse
// @Router       /health [get]
func (h *HealthHandler) Check(c echo.Context) error {
	start := time.Now()
	dbErr := h.db.PingContext(c.Request().Context())
	latency := time.Since(start).Milliseconds()

	resp := healthResponse{Status: "ok", DB: "ok", DBLatencyMs: &latency}

	if h.loki != nil {
		if err := h.loki.Ready(c.Request().Context()); err != nil {
			resp.Loki = "error"
		} else {
			resp.Loki = "ok"
		}
	}

	if dbErr != nil {
		resp.Status = "degraded"
		resp.DB = "error"
		resp.DBLatencyMs = nil
		resp.Error = dbErr.Error()
		return c.JSON(http.StatusServiceUnavailable, resp)
	}

	return c.JSON(http.StatusOK, resp)
}
