package domain

import "errors"

// Sentinel errors returned by repository implementations.
// Service and handler layers check against these with errors.Is.
var (
	ErrNotFound  = errors.New("not found")
	ErrConflict  = errors.New("conflict")
	ErrForbidden = errors.New("forbidden")
)
