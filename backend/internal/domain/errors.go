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

	// Project role errors (feature 021). ErrSelfRemoval: an admin may not remove
	// themselves from a project (they can change their own role instead).
	// ErrProjectNotEmpty marks a project that still has tasks; see
	// ProjectNotEmptyError for the count.
	ErrSelfRemoval = errors.New("cannot remove yourself from a project")
	// ErrLastProjectAdmin: the change would leave a project with no active admin.
	ErrLastProjectAdmin = errors.New("a project needs at least one active admin")
	ErrProjectNotEmpty  = errors.New("project still has tasks")
	// ErrAlreadyMember: the person is already in the project. ErrProjectHasAdmin:
	// a project cannot be taken over while it still has an active admin. Both are
	// conflicts (409) with their own wording; errors.Is(err, ErrConflict) holds.
	ErrAlreadyMember   = fmt.Errorf("%w: already a member of this project", ErrConflict)
	ErrProjectHasAdmin = fmt.Errorf("%w: this project still has an active admin", ErrConflict)

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

// ForbiddenError is a refusal with its own plain-language reason that is not
// about a role (for example "Only the author can edit a comment"). It satisfies
// errors.Is(err, ErrForbidden).
type ForbiddenError struct{ Reason string }

func (e *ForbiddenError) Error() string { return "forbidden: " + e.Reason }

// Is lets errors.Is(err, ErrForbidden) match.
func (e *ForbiddenError) Is(target error) bool { return target == ErrForbidden }

// AccessDeniedError says a project member lacks a capability. It satisfies
// errors.Is(err, ErrForbidden); handlers answer 403 with Reason().
type AccessDeniedError struct {
	Capability Capability
	Role       string
}

func (e *AccessDeniedError) Error() string { return "forbidden: " + e.Reason() }

// Is lets errors.Is(err, ErrForbidden) match.
func (e *AccessDeniedError) Is(target error) bool { return target == ErrForbidden }

// Reason is a plain-language sentence safe to show to the person, for example
// "Viewers can't edit tasks".
func (e *AccessDeniedError) Reason() string {
	who := map[string]string{RoleViewer: "Viewers", RoleMember: "Members", RoleAdmin: "Admins"}[e.Role]
	if who == "" {
		who = "Your role"
	}
	action, ok := denialAction[e.Capability]
	if !ok {
		action = "do that"
	}
	return who + " can't " + action
}

var denialAction = map[Capability]string{
	CapProjectView:         "view this project",
	CapTaskWrite:           "edit tasks",
	CapTaskDeleteAny:       "delete other people's tasks",
	CapCommentWrite:        "comment",
	CapCommentDeleteAny:    "delete other people's comments",
	CapLabelWrite:          "change labels",
	CapLabelDelete:         "delete labels",
	CapAttachmentWrite:     "attach files",
	CapAttachmentDeleteAny: "delete other people's attachments",
	CapSprintManage:        "manage sprints",
	CapProjectEdit:         "edit the project",
	CapMemberManage:        "add, remove or change people",
	CapProjectDelete:       "delete the project",
}

// ProjectNotEmptyError is returned when a project with tasks is deleted. It
// satisfies errors.Is(err, ErrProjectNotEmpty).
type ProjectNotEmptyError struct{ TaskCount int }

func (e *ProjectNotEmptyError) Error() string { return ErrProjectNotEmpty.Error() }

// Is lets errors.Is(err, ErrProjectNotEmpty) match.
func (e *ProjectNotEmptyError) Is(target error) bool { return target == ErrProjectNotEmpty }
