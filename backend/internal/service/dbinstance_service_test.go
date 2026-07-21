package service_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// ── stubDBInstanceClient ──────────────────────────────────────────────────────

type stubDBInstanceClient struct {
	mu sync.Mutex

	startCalls int
	stopCalls  int

	startErr error
	stopErr  error

	describeCallCount int
	describeState     domain.DBInstanceState
	describeErr       error
}

func (c *stubDBInstanceClient) StartDBInstance(_ context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.startCalls++
	return c.startErr
}

func (c *stubDBInstanceClient) StopDBInstance(_ context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stopCalls++
	return c.stopErr
}

func (c *stubDBInstanceClient) DescribeState(_ context.Context) (domain.DBInstanceState, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.describeCallCount++
	return c.describeState, c.describeErr
}

func (c *stubDBInstanceClient) describeCalls() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.describeCallCount
}

func (c *stubDBInstanceClient) StartCalls() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.startCalls
}

func (c *stubDBInstanceClient) StopCalls() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stopCalls
}

// ── fakeClock ──────────────────────────────────────────────────────────────────

type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeClock(start time.Time) *fakeClock { return &fakeClock{now: start} }

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

const testIdleTimeout = 10 * time.Minute
const testStartFailureBound = 3

func newTestTracker(clock *fakeClock) *service.DBLifecycleTracker {
	return service.NewDBLifecycleTracker(testIdleTimeout, testStartFailureBound, clock.Now)
}

// ── FR-001, FR-003: activity tracking resets the idle countdown ───────────────

func TestDBLifecycleTracker_RecordActivity_ResetsIdleCountdown(t *testing.T) {
	clock := newFakeClock(time.Now())
	tr := newTestTracker(clock)
	tr.RecordActivity()

	clock.Advance(9 * time.Minute)
	tr.RecordActivity() // fresh activity before the 10m window elapses

	clock.Advance(9 * time.Minute) // 9m since the reset — still well under 10m total
	assert.False(t, tr.TryClaimStop(), "must not claim a stop before the idle window elapses since the last activity")
}

// ── FR-002, FR-008: a stop is only claimed when idle, running, and nothing in flight ──

func TestDBLifecycleTracker_TryClaimStop_OnlyWhenIdleRunningAndNotInFlight(t *testing.T) {
	clock := newFakeClock(time.Now())
	tr := newTestTracker(clock)
	tr.RecordActivity()

	clock.Advance(testIdleTimeout - time.Second)
	assert.False(t, tr.TryClaimStop(), "must not stop before the idle timeout has elapsed")

	clock.Advance(2 * time.Second) // now just past the timeout
	assert.True(t, tr.TryClaimStop(), "must claim a stop once idle timeout has elapsed with state running and nothing in flight")
}

func TestDBLifecycleTracker_TryClaimStop_BlockedByInFlightOperation(t *testing.T) {
	clock := newFakeClock(time.Now())
	tr := newTestTracker(clock)
	tr.RecordActivity()
	tr.IncrementInFlight() // a long-running operation starts

	clock.Advance(testIdleTimeout + time.Minute) // long past the idle window

	assert.False(t, tr.TryClaimStop(), "must not stop while an operation is still in flight, no matter how long it has been running (FR-008, research.md Decision 8)")

	tr.DecrementInFlight()
	assert.True(t, tr.TryClaimStop(), "once the in-flight operation completes, a stop can be claimed")
}

func TestDBLifecycleTracker_TryClaimStop_ZeroWhileActivityKeepsArriving(t *testing.T) {
	clock := newFakeClock(time.Now())
	tr := newTestTracker(clock)

	for i := 0; i < 5; i++ {
		tr.RecordActivity()
		clock.Advance(5 * time.Minute) // always well under the 10m window since last activity
		assert.False(t, tr.TryClaimStop())
	}
}

// ── Point-in-time race (spec.md Edge Cases): activity immediately before a stop wins ──

func TestDBLifecycleTracker_ActivityImmediatelyBeforeStop_WinsTheRace(t *testing.T) {
	clock := newFakeClock(time.Now())
	tr := newTestTracker(clock)
	tr.RecordActivity()
	clock.Advance(testIdleTimeout + time.Second) // idle window has elapsed

	// Activity arrives at the last possible moment, immediately before the stop decision.
	tr.RecordActivity()

	assert.False(t, tr.TryClaimStop(), "activity recorded immediately before the stop check must win the race and prevent the stop")
}

// ── FR-006: TryClaimStart dedupes concurrent callers ───────────────────────────

func TestDBLifecycleTracker_TryClaimStart_OnlySucceedsOnceConcurrently(t *testing.T) {
	clock := newFakeClock(time.Now())
	tr := newTestTracker(clock)
	// Force into a stopped state via the normal path.
	tr.RecordActivity()
	clock.Advance(testIdleTimeout + time.Second)
	require.True(t, tr.TryClaimStop())
	tr.MarkStopped()

	const callers = 20
	var claimed int32
	var wg sync.WaitGroup
	wg.Add(callers)
	for i := 0; i < callers; i++ {
		go func() {
			defer wg.Done()
			if tr.TryClaimStart() {
				atomic.AddInt32(&claimed, 1)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), claimed, "exactly one concurrent caller should successfully claim the start (FR-006 dedupe)")
}

func TestDBLifecycleTracker_TryClaimStart_FailsWhenNotStopped(t *testing.T) {
	clock := newFakeClock(time.Now())
	tr := newTestTracker(clock)
	// Fresh tracker starts as running.
	assert.False(t, tr.TryClaimStart(), "must not claim a start when the database is already running")
}

// ── inFlightCount invariants ────────────────────────────────────────────────────

func TestDBLifecycleTracker_InFlightCount_NeverNegative(t *testing.T) {
	clock := newFakeClock(time.Now())
	tr := newTestTracker(clock)

	tr.IncrementInFlight()
	tr.DecrementInFlight()
	tr.DecrementInFlight() // an extra decrement must not drive the count negative

	tr.RecordActivity()
	clock.Advance(testIdleTimeout + time.Second)
	assert.True(t, tr.TryClaimStop(), "inFlightCount must have settled at 0, not gone negative, so a stop can still be claimed")
}

// ── FR-010, research.md Decision 6: startFailureCount and the give-up bound ────

func TestDBLifecycleTracker_RecordStartFailure_GivesUpAtBoundThenResets(t *testing.T) {
	clock := newFakeClock(time.Now())
	tr := newTestTracker(clock)
	tr.RecordActivity()
	clock.Advance(testIdleTimeout + time.Second)
	require.True(t, tr.TryClaimStop())
	tr.MarkStopped()

	require.True(t, tr.TryClaimStart())
	assert.False(t, tr.RecordStartFailure(), "1st failure must not give up yet (bound is 3)")

	require.True(t, tr.TryClaimStart(), "a fresh attempt can be claimed after a non-final failure")
	assert.False(t, tr.RecordStartFailure(), "2nd failure must not give up yet")

	require.True(t, tr.TryClaimStart())
	assert.True(t, tr.RecordStartFailure(), "3rd consecutive failure must reach RDS_START_FAILURE_BOUND and give up")

	// data-model.md: startFailureCount resets once a later request tries again (a fresh cycle).
	assert.True(t, tr.TryClaimStart(), "a later request must be able to try again after give-up reset the cycle")
}

func TestDBLifecycleTracker_MarkStarted_ResetsFailureCountAndState(t *testing.T) {
	clock := newFakeClock(time.Now())
	tr := newTestTracker(clock)
	tr.RecordActivity()
	clock.Advance(testIdleTimeout + time.Second)
	require.True(t, tr.TryClaimStop())
	tr.MarkStopped()

	require.True(t, tr.TryClaimStart())
	require.False(t, tr.RecordStartFailure())
	tr.MarkStartAccepted()

	tr.MarkStarted()
	assert.Equal(t, domain.DBInstanceRunning, tr.CurrentState())

	// A subsequent idle period should be stoppable again — confirms failure count/state didn't get stuck.
	clock.Advance(testIdleTimeout + time.Second)
	assert.True(t, tr.TryClaimStop())
}

// ── T010: the idle-check loop (CheckAndStop) calls StopDBInstance exactly once ──

func TestDBLifecycleTracker_CheckAndStop_CallsStopExactlyOnceWhenIdle(t *testing.T) {
	clock := newFakeClock(time.Now())
	tr := newTestTracker(clock)
	client := &stubDBInstanceClient{}
	tr.RecordActivity()

	clock.Advance(testIdleTimeout + time.Second)
	attempted, err := tr.CheckAndStop(context.Background(), client)
	require.NoError(t, err)
	assert.True(t, attempted)
	assert.Equal(t, 1, client.StopCalls(), "StopDBInstance must be called exactly once for this idle tick")
	assert.Equal(t, domain.DBInstanceStopped, tr.CurrentState())

	// A second tick with the database already stopped must not call Stop again.
	attempted, err = tr.CheckAndStop(context.Background(), client)
	require.NoError(t, err)
	assert.False(t, attempted)
	assert.Equal(t, 1, client.StopCalls(), "a subsequent tick must not call StopDBInstance again")
}

func TestDBLifecycleTracker_CheckAndStop_ZeroCallsWhileActivityOrInFlight(t *testing.T) {
	clock := newFakeClock(time.Now())
	tr := newTestTracker(clock)
	client := &stubDBInstanceClient{}

	tr.RecordActivity()
	clock.Advance(5 * time.Minute) // well under the idle window
	attempted, err := tr.CheckAndStop(context.Background(), client)
	require.NoError(t, err)
	assert.False(t, attempted)
	assert.Equal(t, 0, client.StopCalls())

	tr.IncrementInFlight()
	clock.Advance(testIdleTimeout + time.Minute) // idle window elapsed, but an operation is in flight
	attempted, err = tr.CheckAndStop(context.Background(), client)
	require.NoError(t, err)
	assert.False(t, attempted)
	assert.Equal(t, 0, client.StopCalls(), "must not call StopDBInstance while a request remains in flight")
}

func TestDBLifecycleTracker_CheckAndStop_FailedCallRevertsToRunning(t *testing.T) {
	clock := newFakeClock(time.Now())
	tr := newTestTracker(clock)
	client := &stubDBInstanceClient{stopErr: assert.AnError}
	tr.RecordActivity()
	clock.Advance(testIdleTimeout + time.Second)

	attempted, err := tr.CheckAndStop(context.Background(), client)
	assert.True(t, attempted)
	assert.Error(t, err)
	assert.Equal(t, domain.DBInstanceRunning, tr.CurrentState(), "a failed stop call must leave state running so the next tick retries")

	// Next tick retries and succeeds once the transient error clears.
	client.stopErr = nil
	attempted, err = tr.CheckAndStop(context.Background(), client)
	require.NoError(t, err)
	assert.True(t, attempted)
	assert.Equal(t, domain.DBInstanceStopped, tr.CurrentState())
}

// ── CheckStartProgress: polling for a pending start to complete ───────────────

func TestDBLifecycleTracker_CheckStartProgress_MarksStartedWhenAvailable(t *testing.T) {
	clock := newFakeClock(time.Now())
	tr := newTestTracker(clock)
	client := &stubDBInstanceClient{}
	tr.RecordActivity()
	clock.Advance(testIdleTimeout + time.Second)
	require.True(t, tr.TryClaimStop())
	tr.MarkStopped()
	require.True(t, tr.TryClaimStart())
	tr.MarkStartAccepted()

	client.describeState = domain.DBInstanceStarting
	justStarted, err := tr.CheckStartProgress(context.Background(), client)
	require.NoError(t, err)
	assert.False(t, justStarted, "must not report started while AWS still reports starting")

	client.describeState = domain.DBInstanceRunning
	justStarted, err = tr.CheckStartProgress(context.Background(), client)
	require.NoError(t, err)
	assert.True(t, justStarted)
	assert.Equal(t, domain.DBInstanceRunning, tr.CurrentState())
}

func TestDBLifecycleTracker_CheckStartProgress_NoOpWhenNotStarting(t *testing.T) {
	clock := newFakeClock(time.Now())
	tr := newTestTracker(clock) // fresh tracker: running
	client := &stubDBInstanceClient{}

	justStarted, err := tr.CheckStartProgress(context.Background(), client)
	require.NoError(t, err)
	assert.False(t, justStarted)
	assert.Equal(t, 0, client.describeCalls(), "must not call DescribeState when there's no pending start to poll")
}

// ── T022, US3: stop and start audit log lines share the same field schema ──────

func TestLogDBLifecycleEvent_StopAndStartShareTheSameFieldSchema(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	log := zap.New(core)

	service.LogDBLifecycleEvent(log, "db_auto_stop", "idle_timeout", "succeeded", nil)
	service.LogDBLifecycleEvent(log, "db_auto_start", "incoming_request", "succeeded", nil)

	entries := logs.All()
	require.Len(t, entries, 2)

	fieldKeys := func(e observer.LoggedEntry) []string {
		keys := make([]string, 0, len(e.Context))
		for _, f := range e.Context {
			keys = append(keys, f.Key)
		}
		return keys
	}

	stopKeys := fieldKeys(entries[0])
	startKeys := fieldKeys(entries[1])
	assert.ElementsMatch(t, []string{"event", "trigger", "outcome"}, stopKeys, "the stop path's log fields must match the documented Stop/Start Audit Entry shape (data-model.md)")
	assert.ElementsMatch(t, stopKeys, startKeys, "the stop and start paths must emit structurally identical field sets")

	assert.Equal(t, "db_auto_stop", entries[0].ContextMap()["event"])
	assert.Equal(t, "idle_timeout", entries[0].ContextMap()["trigger"])
	assert.Equal(t, "succeeded", entries[0].ContextMap()["outcome"])
	assert.Equal(t, "db_auto_start", entries[1].ContextMap()["event"])
	assert.Equal(t, "incoming_request", entries[1].ContextMap()["trigger"])
	assert.Equal(t, "succeeded", entries[1].ContextMap()["outcome"])
}

func TestLogDBLifecycleEvent_FailedOutcomeIncludesErrorFieldAndLogsAtWarn(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	log := zap.New(core)

	service.LogDBLifecycleEvent(log, "db_auto_stop", "idle_timeout", "failed", assert.AnError)
	service.LogDBLifecycleEvent(log, "db_auto_start", "incoming_request", "failed", assert.AnError)

	entries := logs.All()
	require.Len(t, entries, 2)
	for _, e := range entries {
		assert.Equal(t, zapcore.WarnLevel, e.Level, "a failed outcome must log at Warn, not Info")
		assert.Contains(t, e.ContextMap(), "error", "a failed outcome must include the error field")
		assert.Equal(t, "failed", e.ContextMap()["outcome"])
	}
}

// ── Stop-failure handling (spec.md Edge Cases): a failed stop leaves state running ──

func TestDBLifecycleTracker_MarkStopFailed_RevertsToRunning(t *testing.T) {
	clock := newFakeClock(time.Now())
	tr := newTestTracker(clock)
	tr.RecordActivity()
	clock.Advance(testIdleTimeout + time.Second)
	require.True(t, tr.TryClaimStop())

	tr.MarkStopFailed()
	assert.Equal(t, domain.DBInstanceRunning, tr.CurrentState(), "a failed stop call must leave state as running, not stuck mid-transition")

	// The next tick naturally retries since state is running and idle time has elapsed again.
	assert.True(t, tr.TryClaimStop())
}
