// Package service's DBLifecycleTracker implements the idle auto-stop and
// wake-on-hit decision logic for the demo deployment's database (spec 010,
// db-idle-autostop; see docs/decisions/ADR-030-db-idle-autostop.md). All
// lifecycle-decision state (activity timestamp, in-flight operation count,
// start dedupe, and start-failure counting) lives here, behind one mutex, so
// the handler/middleware layer never has to reason about concurrent state on
// its own.
package service

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/sharique/mansooba/internal/domain"
)

// DBLifecycleTracker tracks whether the database instance should be stopped
// or started, and deduplicates concurrent start attempts. One instance lives
// per running backend process (research.md Decision 1) — there is no
// persisted or shared state across processes.
type DBLifecycleTracker struct {
	mu sync.Mutex

	idleTimeout       time.Duration
	startFailureBound int
	now               func() time.Time

	lastActivity      time.Time
	state             domain.DBInstanceState
	startInFlight     bool
	inFlightCount     int
	startFailureCount int
}

// NewDBLifecycleTracker constructs a tracker starting in the running state
// (the database is assumed available when the backend boots). nowFunc is
// injectable so tests can use a fake clock instead of time.Now.
func NewDBLifecycleTracker(idleTimeout time.Duration, startFailureBound int, nowFunc func() time.Time) *DBLifecycleTracker {
	return &DBLifecycleTracker{
		idleTimeout:       idleTimeout,
		startFailureBound: startFailureBound,
		now:               nowFunc,
		lastActivity:      nowFunc(),
		state:             domain.DBInstanceRunning,
	}
}

// RecordActivity marks the database as used right now (FR-001, FR-011),
// resetting the idle countdown (FR-003).
func (t *DBLifecycleTracker) RecordActivity() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lastActivity = t.now()
}

// IncrementInFlight marks one more request/job as currently using the
// database. Callers MUST pair this with a deferred DecrementInFlight so the
// count can never leak (research.md Decision 8).
func (t *DBLifecycleTracker) IncrementInFlight() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.inFlightCount++
}

// DecrementInFlight reverses IncrementInFlight. It is a no-op (never goes
// negative) if called without a matching increment.
func (t *DBLifecycleTracker) DecrementInFlight() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.inFlightCount > 0 {
		t.inFlightCount--
	}
}

// CurrentState returns the tracker's current view of the database's
// lifecycle state.
func (t *DBLifecycleTracker) CurrentState() domain.DBInstanceState {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.state
}

// TryClaimStop atomically checks whether the database is idle, running, and
// free of in-flight operations, and if so transitions it to stopping and
// returns true. The caller (the idle-check ticker) must then call
// StopDBInstance and report the outcome via MarkStopped or MarkStopFailed.
//
// The idle-time check happens inside this same mutex-protected call, which is
// what closes the point-in-time race in spec.md's Edge Cases: an activity
// update that lands immediately before this call is guaranteed to be visible
// here, since RecordActivity and TryClaimStop share the same lock.
func (t *DBLifecycleTracker) TryClaimStop() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.state != domain.DBInstanceRunning {
		return false
	}
	if t.inFlightCount > 0 {
		return false
	}
	if t.now().Sub(t.lastActivity) < t.idleTimeout {
		return false
	}
	t.state = domain.DBInstanceStopping
	return true
}

// MarkStopped records that a claimed stop succeeded.
func (t *DBLifecycleTracker) MarkStopped() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state = domain.DBInstanceStopped
}

// MarkStopFailed records that a claimed stop failed. State reverts to
// running (the stop never actually happened) so the next idle-check tick
// naturally re-evaluates and retries, per spec.md's "stop call itself fails"
// Edge Case — no request was ever blocked waiting on this, so no
// client-facing handling is needed.
func (t *DBLifecycleTracker) MarkStopFailed() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state = domain.DBInstanceRunning
}

// TryClaimStart atomically checks whether the database is stopped and no
// start attempt is already in flight, and if so claims the exclusive right
// to call StartDBInstance, returning true. Concurrent callers that lose the
// race get false and must not call StartDBInstance themselves — this is what
// satisfies FR-006's dedupe requirement for a burst of simultaneous requests.
func (t *DBLifecycleTracker) TryClaimStart() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.state != domain.DBInstanceStopped || t.startInFlight {
		return false
	}
	t.startInFlight = true
	return true
}

// MarkStartAccepted records that StartDBInstance was called successfully
// (AWS acknowledged the request, though the instance is not necessarily
// available yet). State moves to starting; a periodic poller is expected to
// call MarkStarted once DescribeState reports it's actually running.
func (t *DBLifecycleTracker) MarkStartAccepted() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state = domain.DBInstanceStarting
	t.startInFlight = false
}

// MarkStarted records that the database has been confirmed running again
// (via DescribeState). Resets the failure counter and dedupe flag for the
// next stop/start cycle.
func (t *DBLifecycleTracker) MarkStarted() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state = domain.DBInstanceRunning
	t.startInFlight = false
	t.startFailureCount = 0
}

// CheckAndStop evaluates whether the database is idle enough to stop and, if
// so, calls client.StopDBInstance and records the outcome. Returns
// attempted=true whenever a stop was claimed and tried (regardless of
// success), so the caller (cmd/server's idle-check ticker) knows whether to
// log anything for this tick. This is the testable unit backing T010 — the
// ticker itself (package main) stays thin wiring, not business logic
// (Constitution II: domain → service → handler, not main).
func (t *DBLifecycleTracker) CheckAndStop(ctx context.Context, client domain.DBInstanceClient) (attempted bool, err error) {
	if !t.TryClaimStop() {
		return false, nil
	}
	if err := client.StopDBInstance(ctx); err != nil {
		t.MarkStopFailed()
		return true, err
	}
	t.MarkStopped()
	return true, nil
}

// CheckStartProgress polls whether a previously-accepted start has actually
// completed, marking the tracker started if so. Returns justStarted=true
// only on the tick where the transition to running is observed. A
// DescribeState error is returned as-is; it does not affect startFailureCount
// (that counter is specifically for StartDBInstance call failures, not
// "still booting" polls — see research.md Decision 6).
func (t *DBLifecycleTracker) CheckStartProgress(ctx context.Context, client domain.DBInstanceClient) (justStarted bool, err error) {
	if t.CurrentState() != domain.DBInstanceStarting {
		return false, nil
	}
	state, err := client.DescribeState(ctx)
	if err != nil {
		return false, err
	}
	if state == domain.DBInstanceRunning {
		t.MarkStarted()
		return true, nil
	}
	return false, nil
}

// LogDBLifecycleEvent emits a structured audit log line for a single
// automatic stop or start transition (FR-009, data-model.md's Stop/Start
// Audit Entry). Used by both cmd/server's idle-check ticker (the stop path)
// and internal/middleware/dbwake.go (the start path) so the two call sites
// can't drift out of sync in field naming — extracted here rather than
// duplicated, per tasks.md T023.
//
// event is "db_auto_stop" or "db_auto_start"; trigger is "idle_timeout" or
// "incoming_request"; outcome is "initiated", "succeeded", or "failed". err
// is only expected (and logged) when outcome is "failed".
func LogDBLifecycleEvent(log *zap.Logger, event, trigger, outcome string, err error) {
	fields := []zap.Field{
		zap.String("event", event),
		zap.String("trigger", trigger),
		zap.String("outcome", outcome),
	}
	if err != nil {
		fields = append(fields, zap.Error(err))
	}
	if outcome == "failed" {
		log.Warn(event, fields...)
		return
	}
	log.Info(event, fields...)
}

// RecordStartFailure records that a StartDBInstance call itself failed (not
// a "still booting" delay — an outright failure, e.g. permissions or quota).
// It returns true once RDS_START_FAILURE_BOUND consecutive failures have
// been reached (FR-010, research.md Decision 6), in which case it also
// resets the failure counter and clears the in-flight claim so a later
// request can try a fresh cycle (data-model.md) — state is left as stopped
// throughout this whole failure path, since no start was ever accepted.
func (t *DBLifecycleTracker) RecordStartFailure() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.startInFlight = false
	t.startFailureCount++
	giveUp := t.startFailureCount >= t.startFailureBound
	if giveUp {
		t.startFailureCount = 0
	}
	return giveUp
}
