package middleware

import (
	"net/http"

	appauth "golang-dashboard/internal/auth"

	"github.com/labstack/echo/v4"
)

func RequireRole(roles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user, _ := c.Get("auth_user").(*appauth.SessionUser)
			if user == nil {
				return echo.NewHTTPError(http.StatusForbidden, "Insufficient permissions")
			}
			for _, r := range roles {
				if user.Role == r {
					return next(c)
				}
			}
			return echo.NewHTTPError(http.StatusForbidden, "Insufficient permissions")
		}
	}
}
