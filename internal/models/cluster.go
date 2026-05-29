package models

import "time"

type Cluster struct {
	ID          uint      `gorm:"primaryKey"`
	ClusterName string    `gorm:"column:cluster_name;size:150;not null"`
	HubName     string    `gorm:"column:hub_name;size:150;not null"`
	Region      string    `gorm:"size:100;not null;index"`
	DockNumber  string    `gorm:"column:dock_number;size:50;not null"`
	Backlogs    int       `gorm:"not null;default:0"`
	BacklogsTS  time.Time `gorm:"column:backlogs_ts"`
	Active      bool      `gorm:"not null;default:true"`
	CreatedAt   time.Time `gorm:"column:created_at"`
}
