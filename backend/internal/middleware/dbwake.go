package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/service"
)

// wakingUpRetryAfterSeconds/Millis is the fixed client-facing retry delay
// suggested in the waking_up response (contracts/wake-response.md). A fixed
// value is sufficient for v1 — no backoff schedule is required by the spec.
const (
	wakingUpRetryAfterSeconds = "5"
	wakingUpRetryAfterMillis  = 5000
)

type wakingUpBody struct {
	Status       string `json:"status"`
	RetryAfterMs int    `json:"retry_after_ms"`
}

// DBWake returns an Echo middleware implementing the wake-on-hit contract
// (spec 010, db-idle-autostop; contracts/wake-response.md; see
// docs/decisions/ADR-030-db-idle-autostop.md for the design rationale). When the tracked
// database is not running, it responds immediately with the waking_up
// signal instead of letting the request reach its handler, triggering
// exactly one StartDBInstance call across any concurrent burst of requests
// (FR-006, via tracker.TryClaimStart's dedupe). Once RDS_START_FAILURE_BOUND
// consecutive start attempts have failed, it instead returns a plain 503
// (FR-010) — the client's existing error handling treats that as a genuine,
// non-retryable failure.
func DBWake(tracker *service.DBLifecycleTracker, client domain.DBInstanceClient, log *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if tracker.CurrentState() == domain.DBInstanceRunning {
				return next(c)
			}

			if tracker.TryClaimStart() {
				service.LogDBLifecycleEvent(log, "db_auto_start", "incoming_request", "initiated", nil)
				// StartDBInstance only requests the transition — AWS acknowledges
				// quickly, it does not wait for the instance to become available,
				// so this synchronous call does not violate FR-007's "respond
				// immediately, don't block for minutes" requirement.
				if err := client.StartDBInstance(c.Request().Context()); err != nil {
					giveUp := tracker.RecordStartFailure()
					service.LogDBLifecycleEvent(log, "db_auto_start", "incoming_request", "failed", err)
					if giveUp {
						return echo.NewHTTPError(http.StatusServiceUnavailable,
							"database is currently unavailable, please try again later")
					}
					return respondWakingUp(c)
				}
				// Acceptance by AWS is logged here; "succeeded" (fully available)
				// is logged separately once startDBIdleCheck's poll confirms it
				// (cmd/server/main.go) — MarkStartAccepted only means AWS took the
				// request, not that the instance is usable yet.
				tracker.MarkStartAccepted()
			}

			return respondWakingUp(c)
		}
	}
}

func respondWakingUp(c echo.Context) error {
	c.Response().Header().Set("Retry-After", wakingUpRetryAfterSeconds)
	return c.JSON(http.StatusServiceUnavailable, wakingUpBody{
		Status:       "waking_up",
		RetryAfterMs: wakingUpRetryAfterMillis,
	})
}
