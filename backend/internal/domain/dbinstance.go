package domain

import "context"

// DBInstanceState mirrors the cloud database instance's lifecycle state, as
// tracked by DBLifecycleTracker (internal/service) for the idle
// auto-stop/wake-on-hit feature (spec 010, db-idle-autostop).
type DBInstanceState int

const (
	DBInstanceRunning DBInstanceState = iota
	DBInstanceStopping
	DBInstanceStopped
	DBInstanceStarting
)

func (s DBInstanceState) String() string {
	switch s {
	case DBInstanceRunning:
		return "running"
	case DBInstanceStopping:
		return "stopping"
	case DBInstanceStopped:
		return "stopped"
	case DBInstanceStarting:
		return "starting"
	default:
		return "unknown"
	}
}

// DBInstanceClient is the control-plane contract for starting, stopping, and
// describing the cloud database instance. Implementations live in
// internal/pkg/rdsclient; tests use a mock so the lifecycle-decision logic in
// internal/service never needs real AWS credentials or a live instance.
type DBInstanceClient interface {
	// StartDBInstance requests the instance start. It does not block until the
	// instance is fully available — callers poll DescribeState for that.
	StartDBInstance(ctx context.Context) error
	// StopDBInstance requests the instance stop. Like StartDBInstance, this is
	// a request, not a wait for completion.
	StopDBInstance(ctx context.Context) error
	// DescribeState returns the instance's current lifecycle state.
	DescribeState(ctx context.Context) (DBInstanceState, error)
}
