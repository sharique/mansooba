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
