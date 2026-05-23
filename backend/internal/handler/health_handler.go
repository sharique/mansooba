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

// HealthHandler returns a DB-aware liveness/readiness response.
type HealthHandler struct {
	db DBPinger
}

// NewHealthHandler creates a HealthHandler backed by the given DBPinger.
func NewHealthHandler(db DBPinger) *HealthHandler {
	return &HealthHandler{db: db}
}

// healthResponse is the JSON shape returned by Check.
// DBLatencyMs is a pointer so that the field is present (even at 0ms) in the
// healthy response but absent (nil → omitempty) in the degraded response.
type healthResponse struct {
	Status      string `json:"status"`
	DB          string `json:"db"`
	DBLatencyMs *int64 `json:"db_latency_ms,omitempty"`
	Error       string `json:"error,omitempty"`
}

// Check godoc
// @Summary      Health check
// @Description  Returns overall service status and database connectivity. Returns 200 when healthy, 503 when the database is unreachable.
// @Tags         infra
// @Produce      json
// @Success      200 {object} handler.healthResponse
// @Failure      503 {object} handler.healthResponse
// @Router       /health [get]
func (h *HealthHandler) Check(c echo.Context) error {
	start := time.Now()
	err := h.db.PingContext(c.Request().Context())
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, healthResponse{
			Status: "degraded",
			DB:     "error",
			Error:  err.Error(),
		})
	}

	return c.JSON(http.StatusOK, healthResponse{
		Status:      "ok",
		DB:          "ok",
		DBLatencyMs: &latency,
	})
}
