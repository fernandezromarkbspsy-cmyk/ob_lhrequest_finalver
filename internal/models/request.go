package models

import "time"

type Request struct {
	ID               uint       `gorm:"primaryKey"`
	RequestTimestamp time.Time  `gorm:"column:request_timestamp;not null;index"`
	Cluster          string     `gorm:"size:150;not null;index"`
	Region           string     `gorm:"size:100;not null;index"`
	DockNo           string     `gorm:"column:dock_no;size:50;not null"`
	Backlogs         int        `gorm:"not null;default:0"`
	BacklogsTS       time.Time  `gorm:"column:backlogs_ts"`
	OBFTE            string     `gorm:"column:ob_fte;size:100"`
	OBOpsPIC         string     `gorm:"column:ob_ops_pic;size:100"`
	MidmileFTE       string     `gorm:"column:midmile_fte;size:100"`
	TruckSize        string     `gorm:"column:truck_size;size:50"`
	TruckType        string     `gorm:"column:truck_type;size:100"`
	PlateNumber      string     `gorm:"column:plate_number;size:50"`
	DriverID         string     `gorm:"column:driver_id;size:100"`
	ProvideTime      *time.Time `gorm:"column:provide_time"`
	LinehaulTripNo   string     `gorm:"column:linehaul_trip_no;size:100"`
	DockedTime       *time.Time `gorm:"column:docked_time"`
	Status           string     `gorm:"size:40;not null;default:PENDING_OPS;index"`
	RejectionRemarks string     `gorm:"column:rejection_remarks;size:500"`
	ApprovedAt       *time.Time `gorm:"column:approved_at"`
	RejectedAt       *time.Time `gorm:"column:rejected_at"`
	ConfirmedAt      *time.Time `gorm:"column:confirmed_at"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at"`
}
