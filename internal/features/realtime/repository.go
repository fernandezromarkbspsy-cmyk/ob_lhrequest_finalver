package realtime

import (
	"golang-dashboard/internal/models"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return Repository{db: db}
}

func (r Repository) Available() bool {
	return r.db != nil
}

func (r Repository) PendingSoundNotifications(role string, limit int) ([]models.Notification, error) {
	query := r.db.Where("should_play_sound = ? AND sound_played_at IS NULL", true)
	if role != "" {
		query = query.Where("role = ?", role)
	}
	notifications := []models.Notification{}
	err := query.Order("created_at asc, id asc").Limit(limit).Find(&notifications).Error
	return notifications, err
}
