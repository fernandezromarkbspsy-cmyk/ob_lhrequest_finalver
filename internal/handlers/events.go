package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang-dashboard/internal/events"

	"github.com/labstack/echo/v4"
)

func EventsAPI(c echo.Context) error {
	response := c.Response()
	response.Header().Set(echo.HeaderContentType, "text/event-stream")
	response.Header().Set(echo.HeaderCacheControl, "no-cache")
	response.Header().Set(echo.HeaderConnection, "keep-alive")
	response.Header().Set("X-Accel-Buffering", "no")
	response.WriteHeader(http.StatusOK)

	flusher, ok := response.Writer.(http.Flusher)
	if !ok {
		return echo.NewHTTPError(http.StatusInternalServerError, "Streaming is not supported")
	}

	lastID := c.Request().Header.Get("Last-Event-ID")

	stream, unsubscribe := events.DefaultBus.Subscribe(32)
	defer unsubscribe()

	if err := writeSSE(response, flusher, "system.connected", map[string]string{"status": "connected"}, ""); err != nil {
		return err
	}

	for _, ev := range events.DefaultBus.EventsSince(lastID) {
		if err := writeSSE(response, flusher, ev.Type, ev, ev.ID); err != nil {
			return err
		}
	}

	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case event, ok := <-stream:
			if !ok {
				return nil
			}
			if err := writeSSE(response, flusher, event.Type, event, event.ID); err != nil {
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

func writeSSE(response *echo.Response, flusher http.Flusher, eventName string, payload interface{}, id string) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	msg := ""
	if id != "" {
		msg += fmt.Sprintf("id: %s\n", id)
	}
	msg += fmt.Sprintf("event: %s\ndata: %s\n\n", eventName, data)
	if _, err := fmt.Fprint(response, msg); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}
