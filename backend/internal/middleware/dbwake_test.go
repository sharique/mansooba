package middleware_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/middleware"
	"github.com/sharique/mansooba/internal/service"
)

// ── stubClient ─────────────────────────────────────────────────────────────────

type stubClient struct {
	mu         sync.Mutex
	startCalls int
	startErr   error
}

func (c *stubClient) StartDBInstance(_ context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.startCalls++
	return c.startErr
}
func (c *stubClient) StopDBInstance(_ context.Context) error { return nil }
func (c *stubClient) DescribeState(_ context.Context) (domain.DBInstanceState, error) {
	return domain.DBInstanceRunning, nil
}
func (c *stubClient) StartCalls() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.startCalls
}

// newStoppedTracker returns a tracker already in the stopped state, reached
// via its own normal claim/stop transitions (a zero idle timeout means the
// very first TryClaimStop call succeeds immediately) — this package tests
// the middleware's behavior given a state, not the idle-timing decision
// itself (that's internal/service's job, covered by dbinstance_service_test.go).
func newStoppedTracker() *service.DBLifecycleTracker {
	tr := service.NewDBLifecycleTracker(0, 3, time.Now)
	tr.RecordActivity()
	if !tr.TryClaimStop() {
		panic("test setup: expected TryClaimStop to succeed with a zero idle timeout")
	}
	tr.MarkStopped()
	return tr
}

func doRequest(t *testing.T, mw echo.MiddlewareFunc) (*httptest.ResponseRecorder, bool) {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handlerCalled := false
	next := func(c echo.Context) error {
		handlerCalled = true
		return c.NoContent(http.StatusOK)
	}
	if err := mw(next)(c); err != nil {
		e.HTTPErrorHandler(err, c)
	}
	return rec, handlerCalled
}

// ── Passes through unmodified when running ─────────────────────────────────────

func TestDBWake_PassesThroughWhenRunning(t *testing.T) {
	tr := service.NewDBLifecycleTracker(10*time.Minute, 3, time.Now) // fresh tracker starts running
	client := &stubClient{}

	rec, called := doRequest(t, middleware.DBWake(tr, client, zap.NewNop()))

	assert.True(t, called, "the wrapped handler must run when the database is running")
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 0, client.StartCalls())
}

// ── Returns the waking_up contract when stopped, and triggers exactly one start ──

func TestDBWake_ReturnsWakingUpAndTriggersStart_WhenStopped(t *testing.T) {
	tr := newStoppedTracker()
	client := &stubClient{}

	rec, called := doRequest(t, middleware.DBWake(tr, client, zap.NewNop()))

	assert.False(t, called, "the wrapped handler must not run while the database is stopped")
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Equal(t, "5", rec.Header().Get("Retry-After"))

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "waking_up", body["status"])
	assert.NotEmpty(t, body["retry_after_ms"])

	assert.Equal(t, 1, client.StartCalls(), "the triggering request must cause exactly one StartDBInstance call")
	assert.Equal(t, domain.DBInstanceStarting, tr.CurrentState())
}

func TestDBWake_DedupesConcurrentStartAttempts(t *testing.T) {
	tr := newStoppedTracker()
	client := &stubClient{}

	const requests = 10
	var wg sync.WaitGroup
	wg.Add(requests)
	for i := 0; i < requests; i++ {
		go func() {
			defer wg.Done()
			doRequest(t, middleware.DBWake(tr, client, zap.NewNop()))
		}()
	}
	wg.Wait()

	assert.Equal(t, 1, client.StartCalls(), "a burst of concurrent requests must trigger only one StartDBInstance call (FR-006)")
}

// ── Give-up: plain 503 without the waking_up body once the bound is reached ────

func TestDBWake_GivesUpAfterRepeatedStartFailures(t *testing.T) {
	tr := newStoppedTracker()
	client := &stubClient{startErr: assert.AnError}

	rec1, _ := doRequest(t, middleware.DBWake(tr, client, zap.NewNop()))
	assertWakingUp(t, rec1)
	require.Equal(t, domain.DBInstanceStopped, tr.CurrentState(), "a failed attempt leaves state stopped so the next request can retry")

	rec2, _ := doRequest(t, middleware.DBWake(tr, client, zap.NewNop()))
	assertWakingUp(t, rec2)

	rec3, called := doRequest(t, middleware.DBWake(tr, client, zap.NewNop()))

	assert.False(t, called)
	assert.Equal(t, http.StatusServiceUnavailable, rec3.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec3.Body.Bytes(), &body))
	_, hasWakingUpStatus := body["status"]
	assert.False(t, hasWakingUpStatus, "a request that exhausts the retry bound must get a plain 503 without the waking_up body")
	assert.Equal(t, 3, client.StartCalls())
}

func assertWakingUp(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "waking_up", body["status"])
}
