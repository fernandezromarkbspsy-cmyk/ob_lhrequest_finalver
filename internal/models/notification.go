package models

import "time"

type Notification struct {
	ID              uint       `gorm:"primaryKey"`
	Role            string     `gorm:"size:40;not null;index"`
	RequestID       uint       `gorm:"column:request_id;not null;index"`
	EventType       string     `gorm:"column:event_type;size:80;not null;index"`
	Message         string     `gorm:"size:255;not null"`
	ShouldPlaySound bool       `gorm:"column:should_play_sound;not null;default:true;index"`
	SoundPlayedAt   *time.Time `gorm:"column:sound_played_at"`
	ReadAt          *time.Time `gorm:"column:read_at"`
	CreatedAt       time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
}
