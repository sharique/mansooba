package domain

import (
	"errors"
	"fmt"
)

// Sentinel errors returned by repository implementations.
// Service and handler layers check against these with errors.Is.
var (
	ErrNotFound  = errors.New("not found")
	ErrConflict  = errors.New("conflict")
	ErrForbidden = errors.New("forbidden")

	// ErrInvalidInput marks a request the caller got wrong (a value outside the
	// allowed set, for example). Services return it through NewInputError;
	// handlers map it to HTTP 400 with the message.
	ErrInvalidInput = errors.New("invalid input")

	// Sprint lifecycle errors. Handlers map these to HTTP 409 Conflict.
	ErrSprintAlreadyActive     = errors.New("project already has an active sprint")
	ErrSprintNotDeletable      = errors.New("only planning sprints can be deleted")
	ErrSprintNotEditable       = errors.New("completed sprints cannot be modified")
	ErrSprintInvalidTransition = errors.New("invalid sprint state transition")
	ErrSprintNotStarted        = errors.New("sprint has not been started yet")
	// ErrSprintTooLong maps to 400; ErrSprintFieldLocked to 409.
	ErrSprintTooLong     = errors.New("a sprint cannot be longer than 28 days (4 weeks)")
	ErrSprintFieldLocked = errors.New("an active sprint can only have its end date changed")

	// Auth / revocation errors (feature 007).
	ErrTokenRevoked               = errors.New("token revoked")
	ErrRevocationStoreUnavailable = errors.New("revocation store unavailable")
	ErrAccountDisabled            = errors.New("account disabled")

	// Admin user management errors (feature 007).
	ErrLastAdmin = errors.New("last admin")

	// Attachment errors (feature 009). ErrAttachmentStorageUnavailable wraps
	// S3/LocalStack failures on download-presign or delete — handlers map
	// it to HTTP 502, distinct from a per-file rejection reason on upload.
	ErrAttachmentCapReached         = errors.New("attachment cap reached")
	ErrAttachmentStorageUnavailable = errors.New("attachment storage unavailable")

	// Avatar errors (feature 014). ErrAvatarStorageUnavailable wraps S3
	// failures on upload/delete — handlers map it to HTTP 502 with a generic
	// message, distinct from a client-side rejection of the uploaded file.
	ErrAvatarStorageUnavailable = errors.New("avatar storage unavailable")
	// ErrAvatarRejected marks an upload refused because of the file itself
	// (too large, disallowed type, content mismatch). Its message is safe to
	// show to the client, unlike a storage failure.
	ErrAvatarRejected = errors.New("avatar rejected")

	// Password-change errors (feature 012).
	ErrCurrentPasswordMismatch  = errors.New("current password is incorrect")
	ErrNewPasswordSameAsCurrent = errors.New("new password must differ from current password")
)

// InputError is an ErrInvalidInput with a message that is safe to show the
// caller.
type InputError struct{ Msg string }

func (e *InputError) Error() string { return e.Msg }

// Is makes errors.Is(err, ErrInvalidInput) true for every InputError.
func (e *InputError) Is(target error) bool { return target == ErrInvalidInput }

// NewInputError returns an ErrInvalidInput with the formatted message.
func NewInputError(format string, a ...any) error {
	return &InputError{Msg: fmt.Sprintf(format, a...)}
}
