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
		case <-c.Request().Context().Done():
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
