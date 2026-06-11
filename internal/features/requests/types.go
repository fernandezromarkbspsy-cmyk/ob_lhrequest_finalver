package requests

import "time"

const (
	StatusPending      = "PENDING"
	StatusApproved     = "APPROVED"
	StatusAssigned     = "ASSIGNED"
	StatusForDocking   = "FOR_DOCKING"
	StatusDocked       = "DOCKED"
	StatusConfirmed    = "CONFIRMED"
	StatusCancelled    = "CANCELLED"
	StatusRejectedByMM = "REJECTED_BY_MM"
)

type Row struct {
	ID               uint   `json:"id"`
	RequestTimestamp string `json:"request_timestamp"`
	RequestDate      string `json:"request_date"`
	Cluster          string `json:"cluster"`
	Region           string `json:"region"`
	DockNo           string `json:"dock_no"`
	Backlogs         int    `json:"backlogs"`
	TruckSize        string `json:"truck_size"`
	TruckType        string `json:"truck_type"`
	PlateNumber      string `json:"plate_number"`
	DriverID         string `json:"driver_id"`
	LinehaulTripNo   string `json:"linehaul_trip_no"`
	DockingTime      string `json:"docking_time"`
	Status           string `json:"status"`
	StatusLabel      string `json:"status_label"`
	OBFTE            string `json:"ob_fte"`
	OBOpsPIC         string `json:"ob_ops_pic"`
	MidmileFTE       string `json:"midmile_fte"`
	Remarks          string `json:"remarks"`
}

type Stats struct {
	TotalToday      int64 `json:"total_today"`
	PendingOps      int64 `json:"pending_ops"`
	PendingMM       int64 `json:"pending_mm"`
	ForDocking      int64 `json:"for_docking"`
	ConfirmedTrucks int64 `json:"confirmed_trucks"`
	Rejected        int64 `json:"rejected"`
}

type TrendPoint struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

type ListFilter struct {
	Queue    string
	Status   string
	Search   string
	DateFrom string
	DateTo   string
	Page     int
	PerPage  int
}

type ListResult struct {
	Rows    []Row `json:"requests"`
	Count   int   `json:"count"`
	Total   int64 `json:"total"`
	Page    int   `json:"page"`
	PerPage int   `json:"per_page"`
}

type Payload struct {
	ClusterID      uint   `json:"cluster_id" form:"cluster_id"`
	Cluster        string `json:"cluster" form:"cluster"`
	Region         string `json:"region" form:"region"`
	DockNo         string `json:"dock_no" form:"dock_no"`
	Backlogs       int    `json:"backlogs" form:"backlogs"`
	TruckSize      string `json:"truck_size" form:"truck_size"`
	TruckType      string `json:"truck_type" form:"truck_type"`
	OBOpsPIC       string `json:"ob_ops_pic" form:"ob_ops_pic"`
	OBFTE          string `json:"ob_fte" form:"ob_fte"`
	MidmileFTE     string `json:"midmile_fte" form:"midmile_fte"`
	PlateNumber    string `json:"plate_number" form:"plate_number"`
	DriverID       string `json:"driver_id" form:"driver_id"`
	LinehaulTripNo string `json:"linehaul_trip_no" form:"linehaul_trip_no"`
	DockingTime    string `json:"docking_time" form:"docking_time"`
	Remarks        string `json:"remarks" form:"remarks"`
}

type BulkApprovePayload struct {
	IDs   []uint `json:"ids" form:"ids"`
	OBFTE string `json:"ob_fte" form:"ob_fte"`
}

type ClusterRecord struct {
	ID          uint       `gorm:"column:id"`
	ClusterName string     `gorm:"column:cluster_name"`
	HubName     string     `gorm:"column:hub_name"`
	Region      string     `gorm:"column:region"`
	DockNumber  string     `gorm:"column:dock_number"`
	Backlogs    int        `gorm:"column:backlogs"`
	BacklogsTS  *time.Time `gorm:"column:backlogs_ts"`
}

type AppError struct {
	Code    int
	Message string
}

func (e AppError) Error() string {
	return e.Message
}
