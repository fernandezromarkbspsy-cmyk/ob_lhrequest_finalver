package routes

import (
	"golang-dashboard/internal/handlers"

	"github.com/labstack/echo/v4"
)

func RegisterRoutes(e *echo.Echo) {
	e.GET("/", handlers.Dashboard)
	e.GET("/dashboard", handlers.Dashboard)
	e.GET("/outbound/lh-request", handlers.LHRequests)
	e.GET("/outbound/lh-requests", handlers.LHRequests)
	e.GET("/midmile/truck-request", handlers.TruckRequests)
	e.GET("/dock/officer", handlers.DockOfficer)
	e.GET("/settings", handlers.Settings)

	e.POST("/api/login", handlers.LoginAPI)
	e.GET("/api/stats", handlers.StatsAPI)
	e.GET("/api/events", handlers.EventsAPI)
	e.GET("/api/request-trend", handlers.RequestTrendAPI)
	e.GET("/api/requests", handlers.RequestsAPI)
	e.POST("/api/requests", handlers.CreateRequestAPI)
	e.POST("/api/requests/:id/edit", handlers.EditRequestAPI)
	e.POST("/api/requests/:id/cancel", handlers.CancelRequestAPI)
	e.POST("/api/requests/:id/approve", handlers.ApproveRequestAPI)
	e.POST("/api/requests/:id/reject", handlers.RejectRequestAPI)
	e.POST("/api/requests/:id/assign", handlers.AssignRequestAPI)
	e.POST("/api/requests/:id/for-docking", handlers.ForDockingRequestAPI)
	e.POST("/api/requests/:id/dock", handlers.DockRequestAPI)
	e.POST("/api/requests/:id/confirm", handlers.ConfirmRequestAPI)
	e.GET("/api/clusters", handlers.ClustersAPI)
	e.GET("/api/qr", handlers.DriverQRAPI)
	e.POST("/api/users", handlers.CreateUserAPI)
}
