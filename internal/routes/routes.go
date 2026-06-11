package routes

import (
	"encoding/json"
	featureAuth "golang-dashboard/internal/features/auth"
	featureClusters "golang-dashboard/internal/features/clusters"
	featureNotifications "golang-dashboard/internal/features/notifications"
	featureQR "golang-dashboard/internal/features/qr"
	featureRealtime "golang-dashboard/internal/features/realtime"
	featureRequests "golang-dashboard/internal/features/requests"
	featureUsers "golang-dashboard/internal/features/users"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/labstack/echo/v4"
)

func RegisterRoutes(r chi.Router) {
	requestsController := featureRequests.NewController()
	usersController := featureUsers.NewController()
	authController := featureAuth.NewController()
	clustersController := featureClusters.NewController()
	notificationsController := featureNotifications.NewController()
	realtimeController := featureRealtime.NewController()
	qrController := featureQR.NewController()

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Post("/api/login", echoHandler(authController.LoginAPI))
	r.Post("/api/auth/login", echoHandler(authController.LoginAPI))
	r.Post("/api/auth/send-otp", echoHandler(authController.SendOTPAPI))
	r.Post("/api/auth/verify-otp", echoHandler(authController.VerifyOTPAPI))
	r.Post("/api/auth/logout", echoHandler(authController.LogoutAPI))
	r.Get("/api/auth/me", echoHandler(authController.MeAPI))
	r.Post("/api/auth/change-password", echoHandler(authController.ChangePasswordAPI))
	r.Get("/api/stats", echoHandler(requestsController.StatsAPI))
	r.Get("/api/events", echoHandler(realtimeController.EventsAPI))
	r.Get("/api/ws", echoHandler(realtimeController.WebSocketAPI))
	r.Get("/api/request-trend", echoHandler(requestsController.TrendAPI))
	r.Get("/api/requests", echoHandler(requestsController.ListAPI))
	r.Post("/api/requests", echoHandler(requestsController.CreateAPI))
	r.Get("/api/requests/{id}", echoHandler(requestsController.GetAPI, "id"))
	r.Put("/api/requests/{id}", echoHandler(requestsController.EditAPI, "id"))
	r.Post("/api/requests/{id}/edit", echoHandler(requestsController.EditAPI, "id"))
	r.Post("/api/requests/{id}/cancel", echoHandler(requestsController.CancelAPI, "id"))
	r.Post("/api/requests/{id}/approve", echoHandler(requestsController.ApproveAPI, "id"))
	r.Post("/api/requests/bulk-approve", echoHandler(requestsController.BulkApproveAPI))
	r.Post("/api/requests/{id}/reject", echoHandler(requestsController.RejectAPI, "id"))
	r.Post("/api/requests/{id}/reject-mm", echoHandler(requestsController.RejectAPI, "id"))
	r.Post("/api/requests/{id}/assign", echoHandler(requestsController.AssignAPI, "id"))
	r.Post("/api/requests/{id}/assign-truck", echoHandler(requestsController.ForDockingAPI, "id"))
	r.Post("/api/requests/{id}/for-docking", echoHandler(requestsController.ForDockingAPI, "id"))
	r.Post("/api/requests/{id}/dock", echoHandler(requestsController.DockAPI, "id"))
	r.Post("/api/requests/{id}/mark-docked", echoHandler(requestsController.DockAPI, "id"))
	r.Post("/api/requests/{id}/confirm", echoHandler(requestsController.ConfirmAPI, "id"))
	r.Get("/api/requests/{id}/events", echoHandler(requestsController.EventsAPI, "id"))
	r.Get("/api/clusters", echoHandler(clustersController.ListAPI))
	r.Get("/api/qr", echoHandler(qrController.DriverAPI))
	r.Get("/api/notifications", echoHandler(notificationsController.ListAPI))
	r.Patch("/api/notifications/{id}/read", echoHandler(notificationsController.ReadAPI, "id"))
	r.Patch("/api/notifications/{id}/sound-played", echoHandler(notificationsController.SoundPlayedAPI, "id"))
	r.Get("/api/realtime/notifications", echoHandler(realtimeController.NotificationsSSEAPI))
	r.Get("/api/users", echoHandler(usersController.ListAPI))
	r.Post("/api/users", echoHandler(usersController.CreateAPI))
	r.Put("/api/users/{id}", echoHandler(usersController.UpdateAPI, "id"))
	r.Patch("/api/users/{id}/disable", echoHandler(usersController.DisableAPI, "id"))
}

func echoHandler(handler echo.HandlerFunc, paramNames ...string) http.HandlerFunc {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}

		code := http.StatusInternalServerError
		message := "Internal server error"
		if he, ok := err.(*echo.HTTPError); ok {
			code = he.Code
			switch value := he.Message.(type) {
			case string:
				message = value
			case error:
				message = value.Error()
			}
		}
		_ = c.JSON(code, map[string]string{"error": message})
	}

	return func(w http.ResponseWriter, r *http.Request) {
		c := e.NewContext(r, w)
		if len(paramNames) > 0 {
			values := make([]string, 0, len(paramNames))
			for _, name := range paramNames {
				values = append(values, chi.URLParam(r, name))
			}
			c.SetParamNames(paramNames...)
			c.SetParamValues(values...)
		}
		if err := handler(c); err != nil {
			e.HTTPErrorHandler(err, c)
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
