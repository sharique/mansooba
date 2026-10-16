package handler_test

import "github.com/labstack/echo/v5"

// setParams sets path parameters on a context built by hand. It replaces
// Echo v4's SetParamNames/SetParamValues pair, which Echo v5 removed in favour
// of echo.PathValues (feature 018).
func setParams(c *echo.Context, names []string, values ...string) {
	pv := make(echo.PathValues, 0, len(names))
	for i, n := range names {
		pv = append(pv, echo.PathValue{Name: n, Value: values[i]})
	}
	c.SetPathValues(pv)
}
