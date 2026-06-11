package requests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang-dashboard/internal/cache"
	"golang-dashboard/internal/events"
	"golang-dashboard/internal/jobs"
	"golang-dashboard/internal/models"
)

var responseCache = cache.NewMemory()

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

func (s Service) Stats() (Stats, error) {
	if !s.repo.Available() {
		return Stats{}, nil
	}
	if cached, ok := responseCache.Get("requests:stats"); ok {
		if stats, ok := cached.(Stats); ok {
			return stats, nil
		}
	}
	stats, err := s.repo.Stats(time.Now())
	if err == nil {
		responseCache.Set("requests:stats", stats, 10*time.Second)
	}
	return stats, err
}

func (s Service) List(filter ListFilter) (ListResult, error) {
	filter.Page = clamp(filter.Page, 1, 100000)
	filter.PerPage = clamp(filter.PerPage, 1, 100)
	if filter.PerPage == 0 {
		filter.PerPage = 20
	}
	if !s.repo.Available() {
		return ListResult{Rows: []Row{}, Page: filter.Page, PerPage: filter.PerPage}, nil
	}
	cacheKey := fmt.Sprintf("requests:list:%s:%s:%s:%s:%s:%d:%d", filter.Queue, filter.Status, filter.Search, filter.DateFrom, filter.DateTo, filter.Page, filter.PerPage)
	if cached, ok := responseCache.Get(cacheKey); ok {
		if result, ok := cached.(ListResult); ok {
			return result, nil
		}
	}

	requests, total, err := s.repo.List(filter)
	if err != nil {
		return ListResult{}, AppError{Code: http.StatusInternalServerError, Message: "Unable to load requests"}
	}
	rows := make([]Row, 0, len(requests))
	for _, request := range requests {
		rows = append(rows, toRow(request))
	}
	result := ListResult{
		Rows:    rows,
		Count:   len(rows),
		Total:   total,
		Page:    filter.Page,
		PerPage: filter.PerPage,
	}
	responseCache.Set(cacheKey, result, 5*time.Second)
	return result, nil
}

func (s Service) Trend(now time.Time) (time.Time, time.Time, []TrendPoint, error) {
	start, end := trendWindow(now)
	points := make([]TrendPoint, 0, int(end.Sub(start).Hours()))
	for hour := start; hour.Before(end); hour = hour.Add(time.Hour) {
		points = append(points, TrendPoint{Label: hour.Format("3PM"), Count: 0})
	}
	if !s.repo.Available() {
		return start, end, points, nil
	}

	timestamps, err := s.repo.RequestTimestamps(start, end)
	if err != nil {
		return start, end, nil, AppError{Code: http.StatusInternalServerError, Message: "Unable to load request trend"}
	}
	for _, timestamp := range timestamps {
		if timestamp.Before(start) || !timestamp.Before(end) {
			continue
		}
		index := int(timestamp.Sub(start).Hours())
		if index >= 0 && index < len(points) {
			points[index].Count++
		}
	}
	return start, end, points, nil
}

func (s Service) Get(id uint) (Row, error) {
	if !s.repo.Available() {
		return Row{}, AppError{Code: http.StatusServiceUnavailable, Message: "Database is not configured"}
	}
	request, err := s.repo.Find(id)
	if err != nil {
		return Row{}, AppError{Code: http.StatusNotFound, Message: "Request not found"}
	}
	return toRow(request), nil
}

func (s Service) Events(id uint) (map[string]interface{}, error) {
	if !s.repo.Available() {
		return nil, AppError{Code: http.StatusServiceUnavailable, Message: "Database is not configured"}
	}
	records, err := s.repo.RequestEvents(id)
	if err != nil {
		return nil, AppError{Code: http.StatusInternalServerError, Message: "Unable to load request events"}
	}
	return map[string]interface{}{"events": records, "count": len(records)}, nil
}

func (s Service) Create(payload Payload) (Row, error) {
	clusterName, region, dockNo, backlogs, backlogsTS, ok := s.requestDetails(payload)
	if !ok {
		return Row{}, AppError{Code: http.StatusBadRequest, Message: "Cluster, region, and dock are required"}
	}

	now := time.Now()
	request := models.Request{
		RequestTimestamp: now,
		Cluster:          clusterName,
		Region:           region,
		DockNo:           dockNo,
		Backlogs:         backlogs,
		BacklogsTS:       backlogsTS,
		TruckSize:        strings.TrimSpace(payload.TruckSize),
		TruckType:        strings.TrimSpace(payload.TruckType),
		OBOpsPIC:         strings.TrimSpace(payload.OBOpsPIC),
		Status:           StatusPending,
	}

	if !s.repo.Available() {
		request.ID = uint(now.Unix())
		s.publish(events.RequestCreated, "create", "", request)
		responseCache.Clear()
		return toRow(request), nil
	}
	if err := s.repo.Create(&request); err != nil {
		return Row{}, AppError{Code: http.StatusInternalServerError, Message: "Unable to create request"}
	}
	s.publish(events.RequestCreated, "create", "", request)
	responseCache.Clear()
	return toRow(request), nil
}

func (s Service) Update(id uint, action string, payload Payload) (Row, error) {
	if !s.repo.Available() {
		return Row{}, AppError{Code: http.StatusServiceUnavailable, Message: "Database is not configured"}
	}
	request, err := s.repo.Find(id)
	if err != nil {
		return Row{}, AppError{Code: http.StatusNotFound, Message: "Request not found"}
	}

	previousStatus := normalizeStatus(request)
	now := time.Now()
	switch action {
	case "edit":
		if request.Status != StatusPending && request.Status != StatusRejectedByMM && request.Status != "" {
			break
		}
		clusterName, region, dockNo, backlogs, backlogsTS, ok := s.requestDetails(payload)
		if !ok {
			return Row{}, AppError{Code: http.StatusBadRequest, Message: "Cluster, region, and dock are required"}
		}
		request.Cluster = clusterName
		request.Region = region
		request.DockNo = dockNo
		request.Backlogs = backlogs
		request.BacklogsTS = backlogsTS
		request.TruckSize = strings.TrimSpace(payload.TruckSize)
		request.TruckType = strings.TrimSpace(payload.TruckType)
		request.OBOpsPIC = strings.TrimSpace(payload.OBOpsPIC)
		request.UpdatedAt = now
	case "cancel":
		request.Status = StatusCancelled
		request.RejectionRemarks = strings.TrimSpace(payload.Remarks)
		request.RejectedAt = &now
	case "approve":
		request.Status = StatusApproved
		request.OBFTE = strings.TrimSpace(payload.OBFTE)
		request.ApprovedAt = &now
	case "reject":
		request.Status = StatusRejectedByMM
		request.RejectionRemarks = strings.TrimSpace(payload.Remarks)
		request.RejectedAt = &now
	case "assign":
		request.Status = StatusAssigned
		request.MidmileFTE = strings.TrimSpace(payload.MidmileFTE)
		request.ProvideTime = &now
	case "for-docking":
		request.Status = StatusForDocking
		request.MidmileFTE = strings.TrimSpace(payload.MidmileFTE)
		request.PlateNumber = strings.TrimSpace(payload.PlateNumber)
		request.ProvideTime = &now
	case "dock":
		dockedTime, err := parseInputTime(payload.DockingTime)
		if err != nil {
			return Row{}, AppError{Code: http.StatusBadRequest, Message: "Invalid docking time"}
		}
		request.DriverID = strings.TrimSpace(payload.DriverID)
		request.LinehaulTripNo = strings.TrimSpace(payload.LinehaulTripNo)
		request.DockedTime = &dockedTime
		request.Status = StatusDocked
	case "confirm":
		request.Status = StatusConfirmed
		request.ConfirmedAt = &now
	}

	if err := validateAction(action, request); err != nil {
		return Row{}, err
	}
	if err := s.repo.Save(&request); err != nil {
		return Row{}, AppError{Code: http.StatusInternalServerError, Message: "Unable to update request"}
	}
	s.publish(eventTypeForAction(action), action, previousStatus, request)
	responseCache.Clear()
	return toRow(request), nil
}

func (s Service) BulkApprove(payload BulkApprovePayload) (ListResult, error) {
	if !s.repo.Available() {
		return ListResult{}, AppError{Code: http.StatusServiceUnavailable, Message: "Database is not configured"}
	}
	if len(payload.IDs) == 0 {
		return ListResult{}, AppError{Code: http.StatusBadRequest, Message: "Request IDs are required"}
	}
	requests, err := s.repo.EligibleForBulkApprove(payload.IDs)
	if err != nil {
		return ListResult{}, AppError{Code: http.StatusInternalServerError, Message: "Unable to load requests"}
	}

	now := time.Now()
	rows := make([]Row, 0, len(requests))
	for i := range requests {
		previousStatus := normalizeStatus(requests[i])
		requests[i].Status = StatusApproved
		requests[i].OBFTE = strings.TrimSpace(payload.OBFTE)
		requests[i].ApprovedAt = &now
		if err := s.repo.Save(&requests[i]); err != nil {
			return ListResult{}, AppError{Code: http.StatusInternalServerError, Message: "Unable to approve requests"}
		}
		s.publish(events.RequestApproved, "approve", previousStatus, requests[i])
		rows = append(rows, toRow(requests[i]))
	}
	responseCache.Clear()
	return ListResult{Rows: rows, Count: len(rows), Total: int64(len(rows)), Page: 1, PerPage: len(rows)}, nil
}

func (s Service) requestDetails(payload Payload) (string, string, string, int, *time.Time, bool) {
	clusterName := strings.TrimSpace(payload.Cluster)
	region := strings.TrimSpace(payload.Region)
	dockNo := strings.TrimSpace(payload.DockNo)
	backlogs := payload.Backlogs
	var backlogsTS *time.Time
	if s.repo.Available() && payload.ClusterID > 0 {
		cluster, err := s.repo.LookupCluster(payload.ClusterID)
		if err != nil {
			return "", "", "", 0, nil, false
		}
		clusterName = strings.TrimSpace(cluster.ClusterName)
		region = strings.TrimSpace(cluster.Region)
		backlogs = cluster.Backlogs
		backlogsTS = cluster.BacklogsTS
		if dockNo == "" {
			dockNo = strings.TrimSpace(cluster.DockNumber)
		}
	}
	return clusterName, region, dockNo, backlogs, backlogsTS, clusterName != "" && region != "" && dockNo != ""
}

func (s Service) publish(eventType, action, previousStatus string, request models.Request) {
	status := normalizeStatus(request)
	payload := map[string]interface{}{
		"cluster":           request.Cluster,
		"region":            request.Region,
		"dock_no":           request.DockNo,
		"plate_number":      request.PlateNumber,
		"linehaul_trip_no":  request.LinehaulTripNo,
		"request_timestamp": formatTime(request.RequestTimestamp),
	}
	if s.repo.Available() {
		payloadJSON, _ := json.Marshal(payload)
		if err := s.repo.CreateRequestEvent(models.RequestEvent{
			RequestID:      request.ID,
			EventType:      eventType,
			Action:         action,
			Status:         status,
			PreviousStatus: previousStatus,
			Payload:        payloadJSON,
		}); err == nil {
			records := notificationRecords(eventType, request)
			jobs.Default.Enqueue(func() {
				_ = s.repo.CreateNotifications(records)
			})
		}
	}
	events.DefaultBus.Publish(events.Event{
		ID:             strconv.FormatInt(time.Now().UnixNano(), 10),
		Type:           eventType,
		OccurredAt:     time.Now(),
		Aggregate:      "request",
		AggregateID:    request.ID,
		Action:         action,
		Status:         status,
		PreviousStatus: previousStatus,
		Payload:        payload,
	})
}

func validateAction(action string, request models.Request) error {
	switch action {
	case "reject", "cancel":
		if strings.TrimSpace(request.RejectionRemarks) == "" {
			return AppError{Code: http.StatusBadRequest, Message: "Remarks are required"}
		}
	case "for-docking":
		if strings.TrimSpace(request.PlateNumber) == "" {
			return AppError{Code: http.StatusBadRequest, Message: "Plate number is required"}
		}
	case "dock":
		if strings.TrimSpace(request.DriverID) == "" || strings.TrimSpace(request.LinehaulTripNo) == "" {
			return AppError{Code: http.StatusBadRequest, Message: "Driver ID and LH Trip Number are required"}
		}
	}
	return nil
}

func eventTypeForAction(action string) string {
	switch action {
	case "create":
		return events.RequestCreated
	case "approve":
		return events.RequestApproved
	case "edit":
		return events.RequestEdited
	case "cancel":
		return events.RequestCancelled
	case "reject":
		return events.RequestRejectedByMM
	case "assign":
		return events.TruckAssigned
	case "for-docking":
		return events.TruckForDocking
	case "dock":
		return events.TruckDocked
	case "confirm":
		return events.RequestConfirmed
	default:
		return events.RequestEdited
	}
}

func notificationRecords(eventType string, request models.Request) []models.Notification {
	roles := notificationRoles(eventType)
	notifications := make([]models.Notification, 0, len(roles))
	for _, role := range roles {
		notifications = append(notifications, models.Notification{
			Role:      role,
			RequestID: request.ID,
			EventType: eventType,
			Message:   notificationMessage(eventType, request),
		})
	}
	return notifications
}

func notificationRoles(eventType string) []string {
	switch eventType {
	case events.RequestCreated, events.RequestRejectedByMM:
		return []string{"fte_ops"}
	case events.RequestApproved:
		return []string{"fte_mm"}
	case events.TruckAssigned, events.TruckForDocking:
		return []string{"dock_officer", "doc_officer"}
	case events.RequestConfirmed:
		return []string{"fte_ops", "fte_mm"}
	default:
		return nil
	}
}

func notificationMessage(eventType string, request models.Request) string {
	switch eventType {
	case events.RequestCreated:
		return "New linehaul request needs FTE Ops review: " + request.Cluster
	case events.RequestApproved:
		return "Approved request needs Midmile truck action: " + request.Cluster
	case events.RequestRejectedByMM:
		return "Request was rejected by Midmile: " + request.Cluster
	case events.TruckAssigned, events.TruckForDocking:
		return "Truck is ready for dock action: " + request.Cluster
	case events.RequestConfirmed:
		return "Request confirmed: " + request.Cluster
	default:
		return "Request updated: " + request.Cluster
	}
}

func trendWindow(now time.Time) (time.Time, time.Time) {
	location := now.Location()
	year, month, day := now.Date()
	todayStart := time.Date(year, month, day, 0, 0, 0, 0, location)
	todaySixAM := todayStart.Add(6 * time.Hour)
	todaySixPM := todayStart.Add(18 * time.Hour)
	if now.Before(todaySixAM) {
		start := todaySixPM.AddDate(0, 0, -1)
		return start, todaySixAM
	}
	return todaySixPM, todaySixPM.Add(12 * time.Hour)
}

func toRow(request models.Request) Row {
	status := normalizeStatus(request)
	return Row{
		ID:               request.ID,
		RequestTimestamp: formatTime(request.RequestTimestamp),
		RequestDate:      formatDate(request.RequestTimestamp),
		Cluster:          request.Cluster,
		Region:           request.Region,
		DockNo:           request.DockNo,
		Backlogs:         request.Backlogs,
		TruckSize:        request.TruckSize,
		TruckType:        request.TruckType,
		PlateNumber:      request.PlateNumber,
		DriverID:         request.DriverID,
		LinehaulTripNo:   request.LinehaulTripNo,
		DockingTime:      formatTimePtr(request.DockedTime),
		Status:           status,
		StatusLabel:      statusLabel(status),
		OBFTE:            request.OBFTE,
		OBOpsPIC:         request.OBOpsPIC,
		MidmileFTE:       request.MidmileFTE,
		Remarks:          request.RejectionRemarks,
	}
}

func normalizeStatus(request models.Request) string {
	if request.Status != "" {
		return normalizeStatusValue(request.Status)
	}
	if request.RejectedAt != nil || request.RejectionRemarks != "" {
		return StatusRejectedByMM
	}
	if request.DockedTime != nil {
		return StatusDocked
	}
	if request.ConfirmedAt != nil {
		return StatusForDocking
	}
	if request.ApprovedAt != nil || request.ProvideTime != nil {
		return StatusApproved
	}
	return StatusPending
}

func normalizeStatusValue(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "PENDING_OPS", StatusPending:
		return StatusPending
	case "PENDING_MM", StatusApproved:
		return StatusApproved
	case "CANCELED", StatusCancelled:
		return StatusCancelled
	case "REJECTED", StatusRejectedByMM:
		return StatusRejectedByMM
	case StatusAssigned, StatusForDocking, StatusDocked, StatusConfirmed, "ALL":
		return strings.ToUpper(strings.TrimSpace(status))
	default:
		return strings.ToUpper(strings.TrimSpace(status))
	}
}

func statusLabel(status string) string {
	switch status {
	case StatusPending:
		return "Pending"
	case StatusApproved:
		return "Approved"
	case StatusAssigned:
		return "Assigned"
	case StatusForDocking:
		return "For Docking"
	case StatusDocked:
		return "Docked"
	case StatusConfirmed:
		return "Confirmed"
	case StatusCancelled:
		return "Cancelled"
	case StatusRejectedByMM:
		return "Rejected by MM"
	default:
		return "Pending"
	}
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.Format("Jan 02, 2006 03:04 PM")
}

func formatTimePtr(value *time.Time) string {
	if value == nil {
		return ""
	}
	return formatTime(*value)
}

func formatDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("2006-01-02")
}

func parseInputTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Now(), nil
	}
	for _, layout := range []string{"2006-01-02T15:04", time.RFC3339, "2006-01-02 15:04"} {
		if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, AppError{Code: http.StatusBadRequest, Message: "Invalid docking time"}
}

func formatTrendPeriod(start, end time.Time) string {
	return start.Format("Jan 02, 3 PM") + " - " + end.Format("Jan 02, 3 PM")
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
