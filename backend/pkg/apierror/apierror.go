package apierror

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/sharique/mansooba/internal/domain"
)

// APIError is the standard error body returned by all endpoints.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// HTTPErrorHandler maps domain and Echo errors to JSON responses with appropriate status codes.
func HTTPErrorHandler(c *echo.Context, err error) {
	code := http.StatusInternalServerError
	msg := "internal server error"

	var he *echo.HTTPError
	var sc echo.HTTPStatusCoder
	switch {
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
	c.JSON(code, APIError{Code: http.StatusText(code), Message: msg}) //nolint:errcheck
}
