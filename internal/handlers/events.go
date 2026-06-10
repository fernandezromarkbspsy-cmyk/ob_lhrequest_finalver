package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang-dashboard/internal/database"
	"golang-dashboard/internal/events"
	"golang-dashboard/internal/models"

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

func RealtimeNotificationsAPI(c echo.Context) error {
	response := c.Response()
	response.Header().Set(echo.HeaderContentType, "text/event-stream")
	response.Header().Set(echo.HeaderCacheControl, "no-cache")
	response.Header().Set(echo.HeaderConnection, "keep-alive")
	response.WriteHeader(http.StatusOK)

	flusher, ok := response.Writer.(http.Flusher)
	if !ok {
		return echo.NewHTTPError(http.StatusInternalServerError, "Streaming is not supported")
	}

	role := normalizeRole(c.QueryParam("role"))
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case <-ticker.C:
			if database.DB == nil {
				if _, err := fmt.Fprint(response, ": heartbeat\n\n"); err != nil {
					return err
				}
				flusher.Flush()
				continue
			}

			query := database.DB.Where("should_play_sound = ? AND sound_played_at IS NULL", true)
			if strings.TrimSpace(role) != "" {
				query = query.Where("role = ?", role)
			}

			notifications := []models.Notification{}
			if err := query.Order("created_at asc, id asc").Limit(25).Find(&notifications).Error; err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, "Unable to load notifications")
			}
			for _, notification := range notifications {
				if err := writeSSE(response, flusher, "notification", notification); err != nil {
					return err
				}
			}
		}
	}
}

func NotificationSoundPlayedAPI(c echo.Context) error {
	if database.DB == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "Database is not configured")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid notification ID")
	}

	now := time.Now()
	if err := database.DB.Model(&models.Notification{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"should_play_sound": false,
			"sound_played_at":   &now,
		}).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to mark notification sound as played")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"id": id, "sound_played_at": now})
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
