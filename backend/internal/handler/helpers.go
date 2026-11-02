package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/service"
)

func parseUintParam(c *echo.Context, name string) (uint, error) {
	v, err := strconv.ParseUint(c.Param(name), 10, 64)
	return uint(v), err
}

// validateRequest checks req against its `validate` tags. A failure becomes a
// 400 whose message names the offending field in the form clients send it
// (for example "title is required"). Anything that is not a validation failure
// (such as a missing validator) is returned unchanged and ends up as a 500.
func validateRequest(c *echo.Context, req any) error {
	err := c.Validate(req)
	if err == nil {
		return nil
	}
	var ve validator.ValidationErrors
	if errors.As(err, &ve) && len(ve) > 0 {
		return echo.NewHTTPError(http.StatusBadRequest, validationMessage(ve[0]))
	}
	return err
}

func validationMessage(fe validator.FieldError) string {
	field := snakeCase(fe.Field())
	switch fe.Tag() {
	case "required":
		return field + " is required"
	case "email":
		return field + " must be a valid email address"
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, strings.Join(strings.Fields(fe.Param()), ", "))
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", field, fe.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", field, fe.Param())
	case "password_complexity":
		return field + " must be at least 8 characters and include an upper-case letter, a lower-case letter and a digit"
	}
	return field + " is invalid"
}

// snakeCase turns a Go field name into the JSON name clients use
// (FullName -> full_name, StoryPoints -> story_points).
func snakeCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			b.WriteByte('_')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// forbiddenError turns a refusal into a 403. When the refusal carries a reason
// ("Viewers can't edit tasks") the person gets it; any other refusal keeps the
// plain word, never internal detail.
func forbiddenError(err error) error {
	var denied *domain.AccessDeniedError
	if errors.As(err, &denied) {
		return echo.NewHTTPError(http.StatusForbidden, denied.Reason())
	}
	return echo.NewHTTPError(http.StatusForbidden, "forbidden")
}

// requireSystem asks the user service whether the caller holds a system
// capability and turns a refusal into a 403. Handlers never read the
// administrator flag themselves.
func requireSystem(c *echo.Context, users service.UserService, callerID uint, cap domain.SystemCapability) error {
	if err := users.RequireSystem(c.Request().Context(), callerID, cap); err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			return forbiddenError(err)
		}
		return err
	}
	return nil
}
