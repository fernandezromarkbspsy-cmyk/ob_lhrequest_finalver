package requests

import (
	"net/http"
	"strconv"
	"time"

	"golang-dashboard/internal/database"

	"github.com/labstack/echo/v4"
)

type Controller struct {
	service Service
}

func NewController() Controller {
	repo := NewRepository(database.DB)
	return Controller{service: NewService(repo)}
}

func (c Controller) StatsAPI(ctx echo.Context) error {
	stats, err := c.service.Stats()
	if err != nil {
		return toHTTPError(err)
	}
	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"total_today":      stats.TotalToday,
		"pending_ops":      stats.PendingOps,
		"pending_mm":       stats.PendingMM,
		"pending":          stats.PendingOps,
		"approved":         stats.PendingMM,
		"for_docking":      stats.ForDocking,
		"confirmed_trucks": stats.ConfirmedTrucks,
		"rejected":         stats.Rejected,
	})
}

func (c Controller) TrendAPI(ctx echo.Context) error {
	start, end, points, err := c.service.Trend(time.Now())
	if err != nil {
		return toHTTPError(err)
	}
	return ctx.JSON(http.StatusOK, map[string]interface{}{
		"start":        start.Format(time.RFC3339),
		"end":          end.Format(time.RFC3339),
		"period_label": formatTrendPeriod(start, end),
		"points":       points,
	})
}

func (c Controller) ListAPI(ctx echo.Context) error {
	result, err := c.service.List(ListFilter{
		Queue:    ctx.QueryParam("queue"),
		Status:   ctx.QueryParam("status"),
		Search:   ctx.QueryParam("search"),
		DateFrom: ctx.QueryParam("date_from"),
		DateTo:   ctx.QueryParam("date_to"),
		Page:     intParam(ctx.QueryParam("page"), 1),
		PerPage:  intParam(ctx.QueryParam("per_page"), 20),
	})
	if err != nil {
		return toHTTPError(err)
	}
	return ctx.JSON(http.StatusOK, result)
}

func (c Controller) GetAPI(ctx echo.Context) error {
	row, err := c.service.Get(uintParam(ctx.Param("id")))
	if err != nil {
		return toHTTPError(err)
	}
	return ctx.JSON(http.StatusOK, row)
}

func (c Controller) EventsAPI(ctx echo.Context) error {
	response, err := c.service.Events(uintParam(ctx.Param("id")))
	if err != nil {
		return toHTTPError(err)
	}
	return ctx.JSON(http.StatusOK, response)
}

func (c Controller) CreateAPI(ctx echo.Context) error {
	payload := Payload{}
	if err := ctx.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request payload")
	}
	row, err := c.service.Create(payload)
	if err != nil {
		return toHTTPError(err)
	}
	return ctx.JSON(http.StatusCreated, row)
}

func (c Controller) EditAPI(ctx echo.Context) error {
	return c.update(ctx, "edit")
}

func (c Controller) CancelAPI(ctx echo.Context) error {
	return c.update(ctx, "cancel")
}

func (c Controller) ApproveAPI(ctx echo.Context) error {
	return c.update(ctx, "approve")
}

func (c Controller) RejectAPI(ctx echo.Context) error {
	return c.update(ctx, "reject")
}

func (c Controller) AssignAPI(ctx echo.Context) error {
	return c.update(ctx, "assign")
}

func (c Controller) ForDockingAPI(ctx echo.Context) error {
	return c.update(ctx, "for-docking")
}

func (c Controller) DockAPI(ctx echo.Context) error {
	return c.update(ctx, "dock")
}

func (c Controller) ConfirmAPI(ctx echo.Context) error {
	return c.update(ctx, "confirm")
}

func (c Controller) BulkApproveAPI(ctx echo.Context) error {
	payload := BulkApprovePayload{}
	if err := ctx.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid bulk approve payload")
	}
	result, err := c.service.BulkApprove(payload)
	if err != nil {
		return toHTTPError(err)
	}
	return ctx.JSON(http.StatusOK, result)
}

func (c Controller) update(ctx echo.Context, action string) error {
	payload := Payload{}
	if err := ctx.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request payload")
	}
	row, err := c.service.Update(uintParam(ctx.Param("id")), action, payload)
	if err != nil {
		return toHTTPError(err)
	}
	return ctx.JSON(http.StatusOK, row)
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
