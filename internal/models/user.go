package models

import "time"

type User struct {
	ID             uint      `gorm:"primaryKey"`
	UniqueID       string    `gorm:"column:unique_id;size:100;uniqueIndex"`
	Name           string    `gorm:"size:100;not null"`
	Role           string    `gorm:"size:40;not null;index"`
	OpsID          *string   `gorm:"column:ops_id;size:100;index"`
	Email          *string   `gorm:"size:150;index"`
	IsFTE          bool      `gorm:"column:is_fte;not null;default:false"`
	IsActive       bool      `gorm:"column:is_active;not null;default:true"`
	FirstTimeLogin bool      `gorm:"column:first_time_login;not null;default:true"`
	CreatedAt      time.Time `gorm:"column:created_at;not null;autoCreateTime"`
	PasswordHash   *string   `gorm:"column:password_hash;size:255"`
}
