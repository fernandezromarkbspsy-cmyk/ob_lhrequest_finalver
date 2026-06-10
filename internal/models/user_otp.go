package models

import "time"

type UserOTP struct {
	ID        uint       `gorm:"primaryKey"`
	Email     string     `gorm:"size:150;not null;index"`
	CodeHash  string     `gorm:"column:code_hash;size:255;not null"`
	ExpiresAt time.Time  `gorm:"column:expires_at;not null;index"`
	UsedAt    *time.Time `gorm:"column:used_at"`
	CreatedAt time.Time  `gorm:"column:created_at;not null;autoCreateTime"`
}
