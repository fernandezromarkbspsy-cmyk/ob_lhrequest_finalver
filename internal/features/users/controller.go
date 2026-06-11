package users

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
	payload, err := c.service.List(ListFilter{
		Page:    intParam(ctx.QueryParam("page"), 1),
		PerPage: intParam(ctx.QueryParam("per_page"), 50),
	})
	if err != nil {
		return toHTTPError(err)
	}
	return ctx.JSON(http.StatusOK, payload)
}

func (c Controller) CreateAPI(ctx echo.Context) error {
	payload := Payload{}
	if err := ctx.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid user payload")
	}
	user, err := c.service.Create(payload)
	if err != nil {
		return toHTTPError(err)
	}
	return ctx.JSON(http.StatusCreated, user)
}

func (c Controller) UpdateAPI(ctx echo.Context) error {
	payload := Payload{}
	if err := ctx.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid user payload")
	}
	user, err := c.service.Update(uintParam(ctx.Param("id")), payload)
	if err != nil {
		return toHTTPError(err)
	}
	return ctx.JSON(http.StatusOK, user)
}

func (c Controller) DisableAPI(ctx echo.Context) error {
	payload := Payload{}
	_ = ctx.Bind(&payload)
	response, err := c.service.Disable(uintParam(ctx.Param("id")), payload)
	if err != nil {
		return toHTTPError(err)
	}
	return ctx.JSON(http.StatusOK, response)
}

func intParam(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
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
