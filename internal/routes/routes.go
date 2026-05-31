package routes

import (
	"context"
	"net/http"
	"time"

	appauth "golang-dashboard/internal/auth"
	"golang-dashboard/internal/cache"
	"golang-dashboard/internal/database"
	"golang-dashboard/internal/handlers"
	appmw "golang-dashboard/internal/middleware"
	"golang-dashboard/internal/observability"

	"github.com/labstack/echo/v4"
)

func RegisterRoutes(e *echo.Echo) {
	loginLimiter := appmw.NewIPRateLimiter(10, time.Minute)

	e.GET("/metrics", observability.Handler)

	e.GET("/health", func(c echo.Context) error {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		type componentStatus struct {
			Status string `json:"status"`
		}
		type healthResponse struct {
			Database componentStatus `json:"database"`
			Redis    componentStatus `json:"redis"`
		}

		resp := healthResponse{
			Database: componentStatus{Status: "ok"},
			Redis:    componentStatus{Status: "ok"},
		}

		dbOk := true
		if database.DB == nil {
			resp.Database.Status = "unavailable"
			dbOk = false
		} else if err := database.DB.WithContext(ctx).Exec("SELECT 1").Error; err != nil {
			resp.Database.Status = "error"
			dbOk = false
		}

		if cache.Client == nil {
			resp.Redis.Status = "unavailable"
		} else if err := cache.Client.Ping(ctx).Err(); err != nil {
			resp.Redis.Status = "degraded"
		}

		if !dbOk {
			return c.JSON(http.StatusServiceUnavailable, resp)
		}
		return c.JSON(http.StatusOK, resp)
	})

	e.POST("/api/login", handlers.LoginAPI, loginLimiter)
	e.POST("/api/logout", handlers.LogoutAPI)

	api := e.Group("/api", appauth.RequireAuth())
	api.GET("/stats", handlers.StatsAPI)
	api.GET("/events", handlers.EventsAPI)
	api.GET("/request-trend", handlers.RequestTrendAPI)
	api.GET("/requests", handlers.RequestsAPI)
	api.GET("/clusters", handlers.ClustersAPI)
	api.GET("/qr", handlers.DriverQRAPI)

	opsGroup := api.Group("", appmw.RequireRole("fte_ops", "ops_pic", "admin"))
	opsGroup.POST("/requests", handlers.CreateRequestAPI)
	opsGroup.POST("/requests/:id/edit", handlers.EditRequestAPI)
	opsGroup.POST("/requests/:id/cancel", handlers.CancelRequestAPI)
	opsGroup.POST("/requests/:id/approve", handlers.ApproveRequestAPI)
	opsGroup.POST("/requests/:id/reject", handlers.RejectRequestAPI)

	mmGroup := api.Group("", appmw.RequireRole("fte_mm", "admin"))
	mmGroup.POST("/requests/:id/assign", handlers.AssignRequestAPI)
	mmGroup.POST("/requests/:id/for-docking", handlers.ForDockingRequestAPI)

	dockGroup := api.Group("", appmw.RequireRole("dock_officer", "doc_officer", "admin"))
	dockGroup.POST("/requests/:id/dock", handlers.DockRequestAPI)
	dockGroup.POST("/requests/:id/confirm", handlers.ConfirmRequestAPI)

	usersGroup := api.Group("", appmw.RequireRole("fte_ops", "fte_mm", "admin"))
	usersGroup.POST("/users", handlers.CreateUserAPI)
}
