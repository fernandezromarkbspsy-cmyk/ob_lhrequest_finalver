package notifications

import (
	"net/http"
	"strconv"

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
	response, err := c.service.List(ctx.QueryParam("role"))
	if err != nil {
		return toHTTPError(err)
	}
	return ctx.JSON(http.StatusOK, response)
}

func (c Controller) ReadAPI(ctx echo.Context) error {
	response, err := c.service.Read(uintParam(ctx.Param("id")))
	if err != nil {
		return toHTTPError(err)
	}
	return ctx.JSON(http.StatusOK, response)
}

func (c Controller) SoundPlayedAPI(ctx echo.Context) error {
	response, err := c.service.SoundPlayed(uintParam(ctx.Param("id")))
	if err != nil {
		return toHTTPError(err)
	}
	return ctx.JSON(http.StatusOK, response)
}

func uintParam(value string) uint {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 0
	}
	return uint(parsed)
}

func toHTTPError(err error) error {
	if appErr, ok := err.(AppError); ok {
		return echo.NewHTTPError(appErr.Code, appErr.Message)
	}
	return echo.NewHTTPError(http.StatusInternalServerError, "Internal server error")
}
