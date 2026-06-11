package notifications

import (
	"time"

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

func (r Repository) List(role string, limit int) ([]models.Notification, error) {
	query := r.db.Model(&models.Notification{})
	if role != "" {
		query = query.Where("role = ?", role)
	}
	notifications := []models.Notification{}
	err := query.Order("created_at desc, id desc").Limit(limit).Find(&notifications).Error
	return notifications, err
}

func (r Repository) MarkRead(id uint, now time.Time) error {
	return r.db.Model(&models.Notification{}).Where("id = ?", id).Update("read_at", &now).Error
}

func (r Repository) MarkSoundPlayed(id uint, now time.Time) error {
	return r.db.Model(&models.Notification{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"should_play_sound": false,
			"sound_played_at":   &now,
		}).Error
}
