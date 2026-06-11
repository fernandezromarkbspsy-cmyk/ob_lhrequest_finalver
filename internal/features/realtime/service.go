package realtime

import (
	"net/http"
	"strings"

	"golang-dashboard/internal/models"
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

func (s Service) PendingSoundNotifications(role string) ([]models.Notification, error) {
	if !s.repo.Available() {
		return []models.Notification{}, nil
	}
	notifications, err := s.repo.PendingSoundNotifications(normalizeRole(role), 25)
	if err != nil {
		return nil, AppError{Code: http.StatusInternalServerError, Message: "Unable to load notifications"}
	}
	return notifications, nil
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
