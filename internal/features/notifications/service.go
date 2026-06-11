package notifications

import (
	"net/http"
	"strings"
	"time"
)

type AppError struct {
	Code    int
	Message string
}

func (e AppError) Error() string {
	return e.Message
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

func (s Service) List(role string) (map[string]interface{}, error) {
	if !s.repo.Available() {
		return nil, AppError{Code: http.StatusServiceUnavailable, Message: "Database is not configured"}
	}
	notifications, err := s.repo.List(normalizeRole(role), 100)
	if err != nil {
		return nil, AppError{Code: http.StatusInternalServerError, Message: "Unable to load notifications"}
	}
	return map[string]interface{}{"notifications": notifications, "count": len(notifications)}, nil
}

func (s Service) Read(id uint) (map[string]interface{}, error) {
	if !s.repo.Available() {
		return nil, AppError{Code: http.StatusServiceUnavailable, Message: "Database is not configured"}
	}
	now := time.Now()
	if err := s.repo.MarkRead(id, now); err != nil {
		return nil, AppError{Code: http.StatusInternalServerError, Message: "Unable to mark notification as read"}
	}
	return map[string]interface{}{"id": id, "read_at": now}, nil
}

func (s Service) SoundPlayed(id uint) (map[string]interface{}, error) {
	if !s.repo.Available() {
		return nil, AppError{Code: http.StatusServiceUnavailable, Message: "Database is not configured"}
	}
	now := time.Now()
	if err := s.repo.MarkSoundPlayed(id, now); err != nil {
		return nil, AppError{Code: http.StatusInternalServerError, Message: "Unable to mark notification sound as played"}
	}
	return map[string]interface{}{"id": id, "sound_played_at": now}, nil
}

func normalizeRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "fte_ops", "fte_mm", "ops_pic", "dock_officer":
		return strings.ToLower(strings.TrimSpace(role))
	case "doc_officer":
		return "dock_officer"
	default:
		return ""
	}
}
