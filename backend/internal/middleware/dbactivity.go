package middleware

import (
	"github.com/labstack/echo/v4"

	"github.com/sharique/mansooba/internal/service"
)

// DBActivity returns an Echo middleware that records every request as
// database activity and tracks it as an in-flight operation for the
// lifetime of the request (spec 010, db-idle-autostop — FR-001, FR-003,
// FR-011, and the in-flight guarantee behind FR-008/research.md Decision 8).
//
// Applied globally (registered only when the auto-stop feature is enabled —
// see cmd/server/main.go), since nearly every route in this application
// touches the database in some way; there is no meaningful subset of routes
// to exclude that would change the idle-detection outcome.
func DBActivity(tracker *service.DBLifecycleTracker) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tracker.RecordActivity()
			tracker.IncrementInFlight()
			defer tracker.DecrementInFlight()
			return next(c)
		}
	}
}
