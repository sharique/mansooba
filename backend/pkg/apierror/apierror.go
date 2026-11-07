package apierror

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/sharique/mansooba/internal/domain"
)

// APIError is the standard error body returned by all endpoints.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	// TaskCount is set only when a project could not be deleted because tasks remain.
	TaskCount *int `json:"task_count,omitempty"`
}

// HTTPErrorHandler maps domain and Echo errors to JSON responses with appropriate status codes.
func HTTPErrorHandler(c *echo.Context, err error) {
	code := http.StatusInternalServerError
	msg := "internal server error"

	var he *echo.HTTPError
	var sc echo.HTTPStatusCoder
	var denied *domain.AccessDeniedError
	var forbidden *domain.ForbiddenError
	var notEmpty *domain.ProjectNotEmptyError
	var taskCount *int
	switch {
	case errors.As(err, &denied):
		// A member whose role lacks the capability: say why, in plain words.
		code, msg = http.StatusForbidden, denied.Reason()
	case errors.As(err, &forbidden):
		code, msg = http.StatusForbidden, forbidden.Reason
	case errors.Is(err, domain.ErrAlreadyMember):
		code, msg = http.StatusConflict, "That person is already a member of this project"
	case errors.Is(err, domain.ErrProjectHasAdmin):
		code, msg = http.StatusConflict, "This project still has an active admin"
	case errors.As(err, &notEmpty):
		n := notEmpty.TaskCount
		taskCount = &n
		code, msg = http.StatusConflict, fmt.Sprintf("Project still has %d %s. Remove them first.", n, plural(n, "task", "tasks"))
	case errors.Is(err, domain.ErrLastProjectAdmin):
		code, msg = http.StatusConflict, "A project needs at least one active admin"
	case errors.Is(err, domain.ErrSelfRemoval):
		code, msg = http.StatusForbidden, "Ask another admin to remove you, or change your own role first"
	case errors.Is(err, domain.ErrNotFound):
		code, msg = http.StatusNotFound, "not found"
	case errors.Is(err, domain.ErrConflict):
		code, msg = http.StatusConflict, "conflict"
	case errors.Is(err, domain.ErrForbidden):
		code, msg = http.StatusForbidden, "forbidden"
	case errors.Is(err, domain.ErrInvalidInput):
		code, msg = http.StatusBadRequest, err.Error()
	case errors.Is(err, domain.ErrUsernameInvalid), errors.Is(err, domain.ErrUsernameReserved), errors.Is(err, domain.ErrUsernameImmutable):
		code, msg = http.StatusBadRequest, err.Error()
	case errors.Is(err, domain.ErrUsernameTaken):
		code, msg = http.StatusConflict, err.Error()
	case errors.Is(err, domain.ErrSprintAlreadyActive):
		code, msg = http.StatusConflict, err.Error()
	case errors.Is(err, domain.ErrSprintNotDeletable):
		code, msg = http.StatusConflict, err.Error()
	case errors.Is(err, domain.ErrSprintNotEditable):
		code, msg = http.StatusConflict, err.Error()
	case errors.Is(err, domain.ErrSprintInvalidTransition):
		code, msg = http.StatusConflict, err.Error()
	case errors.Is(err, domain.ErrSprintNotStarted), errors.Is(err, domain.ErrSprintTooLong):
		code, msg = http.StatusBadRequest, err.Error()
	case errors.Is(err, domain.ErrSprintFieldLocked):
		code, msg = http.StatusConflict, err.Error()
	case errors.As(err, &he):
		code = he.Code
		msg = he.Message
		if msg == "" {
			msg = http.StatusText(code)
		}
	case errors.As(err, &sc):
		// Echo's own sentinels (echo.ErrBadRequest, router 404/405, ...) carry
		// only a status code.
		code = sc.StatusCode()
		msg = http.StatusText(code)
	}

	// A response that has already started cannot be replaced.
	if resp, uerr := echo.UnwrapResponse(c.Response()); uerr == nil && resp.Committed {
		return
	}
	c.JSON(code, APIError{Code: http.StatusText(code), Message: msg, TaskCount: taskCount}) //nolint:errcheck
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
