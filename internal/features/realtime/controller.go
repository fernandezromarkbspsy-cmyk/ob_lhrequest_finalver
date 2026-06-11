package realtime

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang-dashboard/internal/database"
	"golang-dashboard/internal/events"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

type Controller struct {
	upgrader websocket.Upgrader
	service  Service
}

func NewController() Controller {
	return Controller{
		service: NewService(NewRepository(database.DB)),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := strings.TrimSpace(r.Header.Get("Origin"))
				if origin == "" {
					return true
				}
				allowed := strings.TrimSpace(os.Getenv("FRONTEND_URL"))
				if allowed == "" {
					allowed = "http://localhost:5173"
				}
				originURL, originErr := url.Parse(origin)
				allowedURL, allowedErr := url.Parse(allowed)
				return originErr == nil && allowedErr == nil && originURL.Host == allowedURL.Host && originURL.Scheme == allowedURL.Scheme
			},
		},
	}
}

func (c Controller) EventsAPI(ctx echo.Context) error {
	response := ctx.Response()
	response.Header().Set(echo.HeaderContentType, "text/event-stream")
	response.Header().Set(echo.HeaderCacheControl, "no-cache")
	response.Header().Set(echo.HeaderConnection, "keep-alive")
	response.WriteHeader(http.StatusOK)

	flusher, ok := response.Writer.(http.Flusher)
	if !ok {
		return echo.NewHTTPError(http.StatusInternalServerError, "Streaming is not supported")
	}

	stream, unsubscribe := events.DefaultBus.Subscribe(16)
	defer unsubscribe()

	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	if err := writeSSE(response, flusher, "system.connected", map[string]string{"status": "connected"}); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Request().Context().Done():
			return nil
		case event := <-stream:
			if err := writeSSE(response, flusher, event.Type, event); err != nil {
				return err
			}
		case <-heartbeat.C:
			if _, err := fmt.Fprint(response, ": heartbeat\n\n"); err != nil {
				return err
			}
			flusher.Flush()
		}
	}
}

func (c Controller) NotificationsSSEAPI(ctx echo.Context) error {
	response := ctx.Response()
	response.Header().Set(echo.HeaderContentType, "text/event-stream")
	response.Header().Set(echo.HeaderCacheControl, "no-cache")
	response.Header().Set(echo.HeaderConnection, "keep-alive")
	response.WriteHeader(http.StatusOK)

	flusher, ok := response.Writer.(http.Flusher)
	if !ok {
		return echo.NewHTTPError(http.StatusInternalServerError, "Streaming is not supported")
	}

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Request().Context().Done():
			return nil
		case <-ticker.C:
			notifications, err := c.service.PendingSoundNotifications(ctx.QueryParam("role"))
			if err != nil {
				if appErr, ok := err.(AppError); ok {
					return echo.NewHTTPError(appErr.Code, appErr.Message)
				}
				return echo.NewHTTPError(http.StatusInternalServerError, "Internal server error")
			}
			if len(notifications) == 0 {
				if _, err := fmt.Fprint(response, ": heartbeat\n\n"); err != nil {
					return err
				}
				flusher.Flush()
				continue
			}
			for _, notification := range notifications {
				if err := writeSSE(response, flusher, "notification", notification); err != nil {
					return err
				}
			}
		}
	}
}

func (c Controller) WebSocketAPI(ctx echo.Context) error {
	conn, err := c.upgrader.Upgrade(ctx.Response(), ctx.Request(), nil)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Unable to open websocket")
	}
	defer conn.Close()

	stream, unsubscribe := events.DefaultBus.Subscribe(16)
	defer unsubscribe()

	if err := conn.WriteJSON(map[string]interface{}{"type": "system.connected", "occurred_at": time.Now()}); err != nil {
		return nil
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-ctx.Request().Context().Done():
			return nil
		case <-done:
			return nil
		case event := <-stream:
			if err := conn.WriteJSON(event); err != nil {
				return nil
			}
		case <-heartbeat.C:
			if err := conn.WriteJSON(map[string]interface{}{"type": "heartbeat", "occurred_at": time.Now()}); err != nil {
				return nil
			}
		}
	}
}

func writeSSE(response *echo.Response, flusher http.Flusher, eventName string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(response, "event: %s\ndata: %s\n\n", eventName, data); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}
