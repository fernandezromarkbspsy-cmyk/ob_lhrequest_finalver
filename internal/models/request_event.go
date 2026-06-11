package models

import (
	"encoding/json"
	"time"
)

type RequestEvent struct {
	ID             uint            `gorm:"primaryKey"`
	RequestID      uint            `gorm:"column:request_id;not null;index"`
	EventType      string          `gorm:"column:event_type;size:80;not null;index"`
	Action         string          `gorm:"size:80"`
	Status         string          `gorm:"size:40;index"`
	PreviousStatus string          `gorm:"column:previous_status;size:40"`
	Payload        json.RawMessage `gorm:"type:jsonb"`
	CreatedAt      time.Time       `gorm:"column:created_at;not null;autoCreateTime"`
}
