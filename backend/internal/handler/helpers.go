package handler

import (
	"strconv"

	"github.com/labstack/echo/v5"
)

func parseUintParam(c *echo.Context, name string) (uint, error) {
	v, err := strconv.ParseUint(c.Param(name), 10, 64)
	return uint(v), err
}
