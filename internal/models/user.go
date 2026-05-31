package models

import "time"

type User struct {
	ID        uint      `gorm:"primaryKey"`
	Name      string    `gorm:"size:100;not null"`
	Role      string    `gorm:"size:40;not null"`
	OpsID     *string   `gorm:"column:ops_id;size:100;uniqueIndex"`
	Email     *string   `gorm:"size:150;uniqueIndex"`
	IsFTE     bool      `gorm:"column:is_fte;not null;default:false"`
	IsActive  bool      `gorm:"column:is_active;not null;default:true"`
	CreatedAt time.Time `gorm:"column:created_at"`
}
