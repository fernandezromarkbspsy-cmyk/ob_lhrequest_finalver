package requests

import (
	"strings"
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

func (r Repository) Stats(now time.Time) (Stats, error) {
	stats := Stats{}
	start := now.Truncate(24 * time.Hour)
	if err := r.db.Model(&models.Request{}).Where("request_timestamp >= ?", start).Count(&stats.TotalToday).Error; err != nil {
		return stats, err
	}
	if err := r.db.Model(&models.Request{}).Where("status IN ? OR status = ''", []string{StatusPending, StatusRejectedByMM}).Count(&stats.PendingOps).Error; err != nil {
		return stats, err
	}
	if err := r.db.Model(&models.Request{}).Where("status = ?", StatusApproved).Count(&stats.PendingMM).Error; err != nil {
		return stats, err
	}
	if err := r.db.Model(&models.Request{}).Where("status = ?", StatusForDocking).Count(&stats.ForDocking).Error; err != nil {
		return stats, err
	}
	if err := r.db.Model(&models.Request{}).Where("status IN ?", []string{StatusAssigned, StatusForDocking, StatusDocked, StatusConfirmed}).Count(&stats.ConfirmedTrucks).Error; err != nil {
		return stats, err
	}
	err := r.db.Model(&models.Request{}).Where("status IN ?", []string{StatusRejectedByMM, StatusCancelled}).Count(&stats.Rejected).Error
	return stats, err
}

func (r Repository) List(filter ListFilter) ([]models.Request, int64, error) {
	query := r.filteredQuery(filter)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	rows := []models.Request{}
	err := query.
		Order("request_timestamp desc, id desc").
		Limit(filter.PerPage).
		Offset((filter.Page - 1) * filter.PerPage).
		Find(&rows).Error
	return rows, total, err
}

func (r Repository) RequestTimestamps(start, end time.Time) ([]time.Time, error) {
	var requests []models.Request
	err := r.db.
		Select("request_timestamp").
		Where("request_timestamp >= ? AND request_timestamp < ?", start, end).
		Find(&requests).Error
	if err != nil {
		return nil, err
	}

	values := make([]time.Time, 0, len(requests))
	for _, request := range requests {
		values = append(values, request.RequestTimestamp)
	}
	return values, nil
}

func (r Repository) Find(id uint) (models.Request, error) {
	request := models.Request{}
	err := r.db.First(&request, id).Error
	return request, err
}

func (r Repository) Save(request *models.Request) error {
	return r.db.Save(request).Error
}

func (r Repository) Create(request *models.Request) error {
	return r.db.Create(request).Error
}

func (r Repository) EligibleForBulkApprove(ids []uint) ([]models.Request, error) {
	requests := []models.Request{}
	err := r.db.Where("id IN ? AND status IN ?", ids, []string{StatusPending, StatusRejectedByMM}).Find(&requests).Error
	return requests, err
}

func (r Repository) CreateRequestEvent(event models.RequestEvent) error {
	return r.db.Create(&event).Error
}

func (r Repository) RequestEvents(requestID uint) ([]models.RequestEvent, error) {
	events := []models.RequestEvent{}
	err := r.db.
		Where("request_id = ?", requestID).
		Order("created_at asc, id asc").
		Find(&events).Error
	return events, err
}

func (r Repository) CreateNotifications(notifications []models.Notification) error {
	if len(notifications) == 0 {
		return nil
	}
	return r.db.Create(&notifications).Error
}

func (r Repository) LookupCluster(id uint) (ClusterRecord, error) {
	cluster := ClusterRecord{}
	err := r.clusterLookupQuery().
		Where("id = ?", id).
		First(&cluster).Error
	return cluster, err
}

func (r Repository) filteredQuery(filter ListFilter) *gorm.DB {
	query := r.db.Model(&models.Request{})
	switch strings.ToLower(filter.Queue) {
	case "ops":
		query = query.Where("status IN ? OR status = ''", []string{StatusPending, StatusRejectedByMM})
	case "mm":
		query = query.Where("status IN ?", []string{StatusApproved, StatusAssigned})
	case "dock":
		query = query.Where("status IN ?", []string{StatusForDocking, StatusDocked})
	}

	if status := normalizeStatusValue(filter.Status); status != "" && status != "ALL" {
		query = query.Where("status = ?", status)
	}

	if search := strings.TrimSpace(filter.Search); search != "" {
		like := "%" + strings.ToLower(search) + "%"
		query = query.Where(
			"LOWER(plate_number) LIKE ? OR LOWER(cluster) LIKE ? OR LOWER(linehaul_trip_no) LIKE ? OR LOWER(driver_id) LIKE ? OR LOWER(region) LIKE ? OR LOWER(dock_no) LIKE ?",
			like,
			like,
			like,
			like,
			like,
			like,
		)
	}

	if from, err := time.Parse("2006-01-02", filter.DateFrom); err == nil {
		query = query.Where("request_timestamp >= ?", from)
	}
	if to, err := time.Parse("2006-01-02", filter.DateTo); err == nil {
		query = query.Where("request_timestamp < ?", to.Add(24*time.Hour))
	}
	return query
}

func (r Repository) clusterLookupQuery() *gorm.DB {
	return r.db.Table("clusters").
		Select("id, cluster_name, COALESCE(hub_name, '') AS hub_name, region, COALESCE(dock_number, '') AS dock_number, COALESCE(backlogs, 0) AS backlogs, backlogs_ts")
}
