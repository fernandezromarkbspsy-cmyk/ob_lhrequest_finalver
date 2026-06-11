package clusters

import (
	"net/http"

	"golang-dashboard/internal/database"

	"github.com/labstack/echo/v4"
)

type Controller struct {
	service Service
}

func NewController() Controller {
	return Controller{service: NewService(NewRepository(database.DB))}
}

func (c Controller) ListAPI(ctx echo.Context) error {
	options, err := c.service.List()
	if err != nil {
		if appErr, ok := err.(AppError); ok {
			return echo.NewHTTPError(appErr.Code, appErr.Message)
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Internal server error")
	}
	return ctx.JSON(http.StatusOK, options)
}
