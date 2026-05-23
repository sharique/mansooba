package apierror

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sharique/mansooba/internal/domain"
)

// APIError is the standard error body returned by all endpoints.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// HTTPErrorHandler maps domain and Echo errors to JSON responses with appropriate status codes.
func HTTPErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	msg := "internal server error"

	var he *echo.HTTPError
	switch {
	case errors.Is(err, domain.ErrNotFound):
		code, msg = http.StatusNotFound, "not found"
	case errors.Is(err, domain.ErrConflict):
		code, msg = http.StatusConflict, "conflict"
	case errors.Is(err, domain.ErrForbidden):
		code, msg = http.StatusForbidden, "forbidden"
	case errors.Is(err, domain.ErrSprintAlreadyActive):
		code, msg = http.StatusConflict, err.Error()
	case errors.Is(err, domain.ErrSprintNotDeletable):
		code, msg = http.StatusConflict, err.Error()
	case errors.Is(err, domain.ErrSprintNotEditable):
		code, msg = http.StatusConflict, err.Error()
	case errors.Is(err, domain.ErrSprintInvalidTransition):
		code, msg = http.StatusConflict, err.Error()
	case errors.Is(err, domain.ErrSprintNotStarted):
		code, msg = http.StatusBadRequest, err.Error()
	case errors.As(err, &he):
		code = he.Code
		if s, ok := he.Message.(string); ok {
			msg = s
		}
	}

	if !c.Response().Committed {
		c.JSON(code, APIError{Code: http.StatusText(code), Message: msg}) //nolint:errcheck
	}
}
