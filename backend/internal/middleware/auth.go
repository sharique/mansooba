package middleware

import (
	"errors"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// JWTAuth returns an Echo middleware that validates Bearer tokens.
// On success it stores the user ID (uint) under the "userID" context key.
func JWTAuth(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := c.Request().Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				return echo.ErrUnauthorized
			}

			tokenStr := strings.TrimPrefix(header, "Bearer ")
			claims := &jwt.RegisteredClaims{}
			_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, errors.New("unexpected signing method")
				}
				return []byte(secret), nil
			})
			if err != nil {
				return echo.ErrUnauthorized
			}

			var userID uint
			if _, err := fmt.Sscanf(claims.Subject, "%d", &userID); err != nil {
				return echo.ErrUnauthorized
			}

			c.Set("userID", userID)
			return next(c)
		}
	}
}
