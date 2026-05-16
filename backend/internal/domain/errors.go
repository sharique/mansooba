package domain

import "errors"

// Sentinel errors returned by repository implementations.
// Service and handler layers check against these with errors.Is.
var (
	ErrNotFound  = errors.New("not found")
	ErrConflict  = errors.New("conflict")
	ErrForbidden = errors.New("forbidden")

	// Sprint lifecycle errors. Handlers map these to HTTP 409 Conflict.
	ErrSprintAlreadyActive     = errors.New("project already has an active sprint")
	ErrSprintNotDeletable      = errors.New("only planning sprints can be deleted")
	ErrSprintNotEditable       = errors.New("completed sprints cannot be modified")
	ErrSprintInvalidTransition = errors.New("invalid sprint state transition")
	ErrSprintNotStarted        = errors.New("sprint has not been started yet")
)
