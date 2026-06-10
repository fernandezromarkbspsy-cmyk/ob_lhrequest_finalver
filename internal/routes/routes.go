package routes

import (
	"golang-dashboard/internal/handlers"
	"net/http"

	"github.com/labstack/echo/v4"
)

func RegisterRoutes(e *echo.Echo) {
	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	e.POST("/api/login", handlers.LoginAPI)
	e.POST("/api/auth/login", handlers.LoginAPI)
	e.POST("/api/auth/send-otp", handlers.SendOTPAPI)
	e.POST("/api/auth/verify-otp", handlers.VerifyOTPAPI)
	e.POST("/api/auth/logout", handlers.LogoutAPI)
	e.GET("/api/auth/me", handlers.MeAPI)
	e.POST("/api/auth/change-password", handlers.ChangePasswordAPI)
	e.GET("/api/stats", handlers.StatsAPI)
	e.GET("/api/events", handlers.EventsAPI)
	e.GET("/api/request-trend", handlers.RequestTrendAPI)
	e.GET("/api/requests", handlers.RequestsAPI)
	e.POST("/api/requests", handlers.CreateRequestAPI)
	e.GET("/api/requests/:id", handlers.RequestAPI)
	e.PUT("/api/requests/:id", handlers.EditRequestAPI)
	e.POST("/api/requests/:id/edit", handlers.EditRequestAPI)
	e.POST("/api/requests/:id/cancel", handlers.CancelRequestAPI)
	e.POST("/api/requests/:id/approve", handlers.ApproveRequestAPI)
	e.POST("/api/requests/bulk-approve", handlers.BulkApproveRequestsAPI)
	e.POST("/api/requests/:id/reject", handlers.RejectRequestAPI)
	e.POST("/api/requests/:id/reject-mm", handlers.RejectRequestAPI)
	e.POST("/api/requests/:id/assign", handlers.AssignRequestAPI)
	e.POST("/api/requests/:id/assign-truck", handlers.ForDockingRequestAPI)
	e.POST("/api/requests/:id/for-docking", handlers.ForDockingRequestAPI)
	e.POST("/api/requests/:id/dock", handlers.DockRequestAPI)
	e.POST("/api/requests/:id/mark-docked", handlers.DockRequestAPI)
	e.POST("/api/requests/:id/confirm", handlers.ConfirmRequestAPI)
	e.GET("/api/requests/:id/events", handlers.RequestEventsAPI)
	e.GET("/api/clusters", handlers.ClustersAPI)
	e.GET("/api/qr", handlers.DriverQRAPI)
	e.GET("/api/notifications", handlers.NotificationsAPI)
	e.PATCH("/api/notifications/:id/read", handlers.ReadNotificationAPI)
	e.PATCH("/api/notifications/:id/sound-played", handlers.NotificationSoundPlayedAPI)
	e.GET("/api/realtime/notifications", handlers.RealtimeNotificationsAPI)
	e.GET("/api/users", handlers.UsersAPI)
	e.POST("/api/users", handlers.CreateUserAPI)
	e.PUT("/api/users/:id", handlers.UpdateUserAPI)
	e.PATCH("/api/users/:id/disable", handlers.DisableUserAPI)
}
