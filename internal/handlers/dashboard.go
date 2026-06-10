package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"golang-dashboard/internal/database"
	"golang-dashboard/internal/events"
	"golang-dashboard/internal/models"

	"github.com/labstack/echo/v4"
	qrcode "github.com/skip2/go-qrcode"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

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

type RequestRow struct {
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

type AppStats struct {
	TotalToday      int64
	PendingOps      int64
	PendingMM       int64
	ForDocking      int64
	ConfirmedTrucks int64
	Rejected        int64
}

type TrendPoint struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

type loginPayload struct {
	LoginType string `json:"login_type" form:"login_type"`
	Email     string `json:"email" form:"email"`
	OpsID     string `json:"ops_id" form:"ops_id"`
	Password  string `json:"password" form:"password"`
}

type otpPayload struct {
	Email string `json:"email" form:"email"`
	Code  string `json:"code" form:"code"`
}

type changePasswordPayload struct {
	CurrentPassword string `json:"current_password" form:"current_password"`
	NewPassword     string `json:"new_password" form:"new_password"`
}

type requestPayload struct {
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

type userPayload struct {
	Name      string `json:"name" form:"name"`
	Role      string `json:"role" form:"role"`
	Email     string `json:"email" form:"email"`
	OpsID     string `json:"ops_id" form:"ops_id"`
	Password  string `json:"password" form:"password"`
	ActorRole string `json:"actor_role" form:"actor_role"`
}

func LoginAPI(c echo.Context) error {
	payload := loginPayload{}
	if err := c.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid login payload")
	}

	loginType := strings.ToLower(strings.TrimSpace(payload.LoginType))
	email := strings.TrimSpace(payload.Email)
	opsID := strings.TrimSpace(payload.OpsID)

	if database.DB == nil {
		return c.JSON(http.StatusOK, demoUser(loginType, email, opsID))
	}

	user := models.User{}
	query := database.DB.Model(&models.User{}).Where("is_active = ?", true)

	switch loginType {
	case "fte":
		if email == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "Email is required")
		}
		query = query.Where(
			"LOWER(COALESCE(email, '')) = LOWER(?) AND role IN ?",
			email,
			[]string{"fte_ops", "fte_mm"},
		)
	case "backroom":
		if opsID == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "Ops ID is required")
		}
		query = query.Where(
			"LOWER(COALESCE(ops_id, '')) = LOWER(?) AND role IN ?",
			opsID,
			[]string{"ops_pic", "dock_officer", "doc_officer"},
		)
	default:
		return echo.NewHTTPError(http.StatusBadRequest, "Choose FTE or Backroom")
	}

	if err := query.First(&user).Error; err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Invalid credentials")
	}
	ensureUserUniqueID(&user)
	if user.PasswordHash != nil && *user.PasswordHash != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(payload.Password)); err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid credentials")
		}
	}

	response := map[string]interface{}{
		"id":       user.ID,
		"name":     user.Name,
		"role":     user.Role,
		"email":    user.Email,
		"ops_id":   user.OpsID,
		"is_fte":   isFTERole(user.Role),
		"redirect": redirectForRole(user.Role),
	}
	setSessionCookie(c, user)
	return c.JSON(http.StatusOK, response)
}

func LogoutAPI(c echo.Context) error {
	c.SetCookie(&http.Cookie{
		Name:     "soc5_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   os.Getenv("APP_ENV") == "production",
	})
	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

func MeAPI(c echo.Context) error {
	claims, ok := readSessionClaims(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "Not signed in")
	}
	return c.JSON(http.StatusOK, claims)
}

func SendOTPAPI(c echo.Context) error {
	if database.DB == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "Database is not configured")
	}

	payload := otpPayload{}
	if err := c.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid OTP payload")
	}
	email := strings.TrimSpace(payload.Email)
	if email == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Email is required")
	}

	var count int64
	database.DB.Model(&models.User{}).
		Where("LOWER(COALESCE(email, '')) = LOWER(?) AND role IN ? AND is_active = ?", email, []string{"fte_ops", "fte_mm"}, true).
		Count(&count)
	if count == 0 {
		return echo.NewHTTPError(http.StatusUnauthorized, "Invalid email")
	}

	code, err := randomOTP()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to generate OTP")
	}
	codeHash, err := hashPassword(code)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to store OTP")
	}

	otp := models.UserOTP{
		Email:     email,
		CodeHash:  codeHash,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	if err := database.DB.Create(&otp).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to store OTP")
	}
	if os.Getenv("APP_ENV") != "production" {
		fmt.Printf("OTP for %s: %s\n", email, code)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"ok": true, "expires_in_seconds": 600})
}

func VerifyOTPAPI(c echo.Context) error {
	if database.DB == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "Database is not configured")
	}

	payload := otpPayload{}
	if err := c.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid OTP payload")
	}
	email := strings.TrimSpace(payload.Email)
	code := strings.TrimSpace(payload.Code)
	if email == "" || code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Email and OTP are required")
	}

	otp := models.UserOTP{}
	if err := database.DB.
		Where("LOWER(email) = LOWER(?) AND used_at IS NULL AND expires_at > ?", email, time.Now()).
		Order("created_at desc, id desc").
		First(&otp).Error; err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Invalid OTP")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(otp.CodeHash), []byte(code)); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Invalid OTP")
	}

	user := models.User{}
	if err := database.DB.
		Where("LOWER(COALESCE(email, '')) = LOWER(?) AND role IN ? AND is_active = ?", email, []string{"fte_ops", "fte_mm"}, true).
		First(&user).Error; err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Invalid email")
	}

	now := time.Now()
	database.DB.Model(&otp).Update("used_at", &now)
	ensureUserUniqueID(&user)
	setSessionCookie(c, user)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":        user.ID,
		"unique_id": user.UniqueID,
		"name":      user.Name,
		"role":      user.Role,
		"email":     user.Email,
		"is_fte":    user.IsFTE,
		"redirect":  redirectForRole(user.Role),
	})
}

func ChangePasswordAPI(c echo.Context) error {
	if database.DB == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "Database is not configured")
	}

	claims, ok := readSessionClaims(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "Not signed in")
	}
	userID, ok := numericClaim(claims["id"])
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "Invalid session")
	}

	payload := changePasswordPayload{}
	if err := c.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid password payload")
	}
	if strings.TrimSpace(payload.NewPassword) == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "New password is required")
	}

	user := models.User{}
	if err := database.DB.First(&user, userID).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "User not found")
	}
	if user.PasswordHash != nil && *user.PasswordHash != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(payload.CurrentPassword)); err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid current password")
		}
	}

	passwordHash, err := hashPassword(payload.NewPassword)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Password is invalid")
	}
	user.PasswordHash = stringPtr(passwordHash)
	user.FirstTimeLogin = false
	if err := database.DB.Save(&user).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to change password")
	}

	return c.JSON(http.StatusOK, map[string]bool{"ok": true})
}

func StatsAPI(c echo.Context) error {
	stats := loadStats()
	return c.JSON(http.StatusOK, map[string]interface{}{
		"total_today":      stats.TotalToday,
		"pending_ops":      stats.PendingOps,
		"pending_mm":       stats.PendingMM,
		"pending":          stats.PendingOps,
		"approved":         stats.PendingMM,
		"for_docking":      stats.ForDocking,
		"confirmed_trucks": stats.ConfirmedTrucks,
		"rejected":         stats.Rejected,
	})
}

func RequestTrendAPI(c echo.Context) error {
	start, end := requestTrendWindow(time.Now())
	points := hourlyRequestTrend(start, end)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"start":        start.Format(time.RFC3339),
		"end":          end.Format(time.RFC3339),
		"period_label": formatTrendPeriod(start, end),
		"points":       points,
	})
}

func RequestsAPI(c echo.Context) error {
	rows := queryRequestRows(c.QueryParam("queue"), c.QueryParam("status"), c.QueryParam("search"), c.QueryParam("date_from"), c.QueryParam("date_to"))
	return c.JSON(http.StatusOK, map[string]interface{}{
		"requests": rows,
		"count":    len(rows),
	})
}

func RequestAPI(c echo.Context) error {
	if database.DB == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "Database is not configured")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request ID")
	}

	request := models.Request{}
	if err := database.DB.First(&request, id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Request not found")
	}

	return c.JSON(http.StatusOK, requestToRow(request))
}

func RequestEventsAPI(c echo.Context) error {
	if database.DB == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "Database is not configured")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request ID")
	}

	events := []models.RequestEvent{}
	if err := database.DB.
		Where("request_id = ?", id).
		Order("created_at asc, id asc").
		Find(&events).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to load request events")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"events": events,
		"count":  len(events),
	})
}

func CreateRequestAPI(c echo.Context) error {
	payload := requestPayload{}
	if err := c.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request payload")
	}

	clusterName := strings.TrimSpace(payload.Cluster)
	region := strings.TrimSpace(payload.Region)
	dockNo := strings.TrimSpace(payload.DockNo)
	backlogs := payload.Backlogs

	var backlogsTS *time.Time

	if database.DB != nil && payload.ClusterID > 0 {
		cluster, err := lookupCluster(payload.ClusterID)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid cluster")
		}
		clusterName = strings.TrimSpace(cluster.ClusterName)
		region = strings.TrimSpace(cluster.Region)
		backlogs = cluster.Backlogs
		backlogsTS = cluster.BacklogsTS
		if dockNo == "" {
			dockNo = strings.TrimSpace(cluster.DockNumber)
		}
	}

	if clusterName == "" || region == "" || dockNo == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Cluster, region, and dock are required")
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

	if database.DB == nil {
		request.ID = uint(time.Now().Unix())
		publishRequestEvent(events.RequestCreated, "create", "", request)
		return c.JSON(http.StatusCreated, requestToRow(request))
	}

	if err := database.DB.Create(&request).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to create request")
	}

	publishRequestEvent(events.RequestCreated, "create", "", request)
	return c.JSON(http.StatusCreated, requestToRow(request))
}

func EditRequestAPI(c echo.Context) error {
	return updateRequest(c, "edit", func(request *models.Request, payload requestPayload, now time.Time) {
		if request.Status != StatusPending && request.Status != StatusRejectedByMM && request.Status != "" {
			return
		}

		clusterName, region, dockNo, backlogs, backlogsTS, ok := requestDetailsFromPayload(payload)
		if !ok {
			return
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
	})
}

func CancelRequestAPI(c echo.Context) error {
	return updateRequest(c, "cancel", func(request *models.Request, payload requestPayload, now time.Time) {
		request.Status = StatusCancelled
		request.RejectionRemarks = strings.TrimSpace(payload.Remarks)
		request.RejectedAt = &now
	})
}

type clusterRecord struct {
	ID          uint       `gorm:"column:id"`
	ClusterName string     `gorm:"column:cluster_name"`
	HubName     string     `gorm:"column:hub_name"`
	Region      string     `gorm:"column:region"`
	DockNumber  string     `gorm:"column:dock_number"`
	Backlogs    int        `gorm:"column:backlogs"`
	BacklogsTS  *time.Time `gorm:"column:backlogs_ts"`
}

func requestDetailsFromPayload(payload requestPayload) (string, string, string, int, *time.Time, bool) {
	clusterName := strings.TrimSpace(payload.Cluster)
	region := strings.TrimSpace(payload.Region)
	dockNo := strings.TrimSpace(payload.DockNo)
	backlogs := payload.Backlogs
	var backlogsTS *time.Time

	if database.DB != nil && payload.ClusterID > 0 {
		cluster, err := lookupCluster(payload.ClusterID)
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

func ApproveRequestAPI(c echo.Context) error {
	return updateRequest(c, "approve", func(request *models.Request, payload requestPayload, now time.Time) {
		request.Status = StatusApproved
		request.OBFTE = strings.TrimSpace(payload.OBFTE)
		request.ApprovedAt = &now
	})
}

func RejectRequestAPI(c echo.Context) error {
	return updateRequest(c, "reject", func(request *models.Request, payload requestPayload, now time.Time) {
		request.Status = StatusRejectedByMM
		request.RejectionRemarks = strings.TrimSpace(payload.Remarks)
		request.RejectedAt = &now
	})
}

func AssignRequestAPI(c echo.Context) error {
	return updateRequest(c, "assign", func(request *models.Request, payload requestPayload, now time.Time) {
		request.Status = StatusAssigned
		request.MidmileFTE = strings.TrimSpace(payload.MidmileFTE)
		request.ProvideTime = &now
	})
}

func ForDockingRequestAPI(c echo.Context) error {
	return updateRequest(c, "for-docking", func(request *models.Request, payload requestPayload, now time.Time) {
		request.Status = StatusForDocking
		request.MidmileFTE = strings.TrimSpace(payload.MidmileFTE)
		request.PlateNumber = strings.TrimSpace(payload.PlateNumber)
		request.ProvideTime = &now
	})
}

func DockRequestAPI(c echo.Context) error {
	return updateRequest(c, "dock", func(request *models.Request, payload requestPayload, now time.Time) {
		dockedTime := now
		if parsed, err := parseInputTime(payload.DockingTime); err == nil {
			dockedTime = parsed
		}

		request.DriverID = strings.TrimSpace(payload.DriverID)
		request.LinehaulTripNo = strings.TrimSpace(payload.LinehaulTripNo)
		request.DockedTime = &dockedTime
		request.Status = StatusDocked
	})
}

func ConfirmRequestAPI(c echo.Context) error {
	return updateRequest(c, "confirm", func(request *models.Request, payload requestPayload, now time.Time) {
		request.Status = StatusConfirmed
		request.ConfirmedAt = &now
	})
}

type bulkApprovePayload struct {
	IDs   []uint `json:"ids" form:"ids"`
	OBFTE string `json:"ob_fte" form:"ob_fte"`
}

func BulkApproveRequestsAPI(c echo.Context) error {
	if database.DB == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "Database is not configured")
	}

	payload := bulkApprovePayload{}
	if err := c.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid bulk approve payload")
	}
	if len(payload.IDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Request IDs are required")
	}

	now := time.Now()
	requests := []models.Request{}
	if err := database.DB.Where("id IN ? AND status IN ?", payload.IDs, []string{StatusPending, StatusRejectedByMM}).Find(&requests).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to load requests")
	}

	for i := range requests {
		previousStatus := normalizeStatus(requests[i])
		requests[i].Status = StatusApproved
		requests[i].OBFTE = strings.TrimSpace(payload.OBFTE)
		requests[i].ApprovedAt = &now
		if err := database.DB.Save(&requests[i]).Error; err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Unable to approve requests")
		}
		publishRequestEvent(events.RequestApproved, "approve", previousStatus, requests[i])
	}

	rows := make([]RequestRow, 0, len(requests))
	for _, request := range requests {
		rows = append(rows, requestToRow(request))
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"requests": rows,
		"count":    len(rows),
	})
}

func ClustersAPI(c echo.Context) error {
	type clusterOption struct {
		ID        uint   `json:"id"`
		Cluster   string `json:"cluster"`
		HubName   string `json:"hub_name"`
		Region    string `json:"region"`
		DockNo    string `json:"dock_no"`
		Backlogs  int    `json:"backlogs"`
		BacklogTS string `json:"backlogs_ts"`
	}

	options := []clusterOption{}
	if database.DB == nil {
		return c.JSON(http.StatusOK, options)
	}

	clusters := []clusterRecord{}
	if err := clusterLookupQuery().
		Where("COALESCE(cluster_name, '') <> ''").
		Order("cluster_name asc, hub_name asc, region asc, dock_number asc").
		Find(&clusters).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to load clusters")
	}

	seen := map[string]bool{}
	for _, cluster := range clusters {
		key := cluster.ClusterName + "|" + cluster.HubName + "|" + cluster.Region + "|" + cluster.DockNumber
		if seen[key] {
			continue
		}
		seen[key] = true
		backlogsTS := ""
		if cluster.BacklogsTS != nil {
			backlogsTS = cluster.BacklogsTS.Format(time.RFC3339)
		}
		options = append(options, clusterOption{
			ID:        cluster.ID,
			Cluster:   cluster.ClusterName,
			HubName:   cluster.HubName,
			Region:    cluster.Region,
			DockNo:    cluster.DockNumber,
			Backlogs:  cluster.Backlogs,
			BacklogTS: backlogsTS,
		})
	}

	return c.JSON(http.StatusOK, options)
}

func DriverQRAPI(c echo.Context) error {
	driverID := strings.TrimSpace(c.QueryParam("value"))
	if driverID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Driver ID is required")
	}

	png, err := qrcode.Encode(driverID, qrcode.Medium, 320)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to generate QR code")
	}

	return c.Blob(http.StatusOK, "image/png", png)
}

func NotificationsAPI(c echo.Context) error {
	if database.DB == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "Database is not configured")
	}

	role := normalizeRole(c.QueryParam("role"))
	query := database.DB.Model(&models.Notification{})
	if role != "" {
		query = query.Where("role = ?", role)
	}

	notifications := []models.Notification{}
	if err := query.Order("created_at desc, id desc").Limit(100).Find(&notifications).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to load notifications")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"notifications": notifications,
		"count":         len(notifications),
	})
}

func ReadNotificationAPI(c echo.Context) error {
	if database.DB == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "Database is not configured")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid notification ID")
	}

	now := time.Now()
	if err := database.DB.Model(&models.Notification{}).Where("id = ?", id).Update("read_at", &now).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to mark notification as read")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"id": id, "read_at": now})
}

func CreateUserAPI(c echo.Context) error {
	if database.DB == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "Database is not configured")
	}

	payload := userPayload{}
	if err := c.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid user payload")
	}

	if !canManageRoles(payload.ActorRole) {
		return echo.NewHTTPError(http.StatusForbidden, "Only FTE Ops and FTE MM can add roles")
	}

	name := strings.TrimSpace(payload.Name)
	role := normalizeRole(payload.Role)
	email := strings.TrimSpace(payload.Email)
	opsID := strings.TrimSpace(payload.OpsID)
	isFTE := isFTERole(role)
	if isFTE {
		opsID = ""
	} else {
		email = ""
	}

	if name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Name is required")
	}
	if role == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Choose a valid role")
	}
	if isFTE && email == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Email is required for FTE Ops and FTE MM")
	}
	if !isFTE && opsID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Ops ID is required for Backroom roles")
	}

	if exists := userIdentifierExists(role, email, opsID); exists {
		return echo.NewHTTPError(http.StatusConflict, "A user with this identifier already exists")
	}

	user := models.User{
		Name:     name,
		Role:     role,
		IsFTE:    isFTE,
		IsActive: true,
	}
	ensureUserUniqueID(&user)
	if isFTE {
		user.Email = stringPtr(email)
	} else {
		user.OpsID = stringPtr(opsID)
	}
	if passwordHash, err := hashPassword(payload.Password); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Password is invalid")
	} else if passwordHash != "" {
		user.PasswordHash = stringPtr(passwordHash)
	}

	if err := database.DB.Create(&user).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to create user")
	}

	events.DefaultBus.Publish(events.Event{
		ID:          strconv.FormatInt(time.Now().UnixNano(), 10),
		Type:        events.UserCreated,
		OccurredAt:  time.Now(),
		Aggregate:   "user",
		AggregateID: user.ID,
		Action:      "create",
		Payload: map[string]interface{}{
			"name": user.Name,
			"role": user.Role,
		},
	})

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"id":         user.ID,
		"name":       user.Name,
		"role":       user.Role,
		"role_label": roleLabel(user.Role),
		"email":      user.Email,
		"ops_id":     user.OpsID,
		"is_fte":     user.IsFTE,
		"is_active":  user.IsActive,
	})
}

func UsersAPI(c echo.Context) error {
	if database.DB == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "Database is not configured")
	}

	users := []models.User{}
	if err := database.DB.Order("created_at desc, id desc").Find(&users).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to load users")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"users": users,
		"count": len(users),
	})
}

func UpdateUserAPI(c echo.Context) error {
	if database.DB == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "Database is not configured")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid user ID")
	}

	payload := userPayload{}
	if err := c.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid user payload")
	}
	if !canManageRoles(payload.ActorRole) {
		return echo.NewHTTPError(http.StatusForbidden, "Only FTE Ops and FTE MM can update users")
	}

	user := models.User{}
	if err := database.DB.First(&user, id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "User not found")
	}

	name := strings.TrimSpace(payload.Name)
	role := normalizeRole(payload.Role)
	email := strings.TrimSpace(payload.Email)
	opsID := strings.TrimSpace(payload.OpsID)
	isFTE := isFTERole(role)

	if name == "" || role == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Name and valid role are required")
	}
	if isFTE && email == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Email is required for FTE roles")
	}
	if !isFTE && opsID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Ops ID is required for Backroom roles")
	}

	user.Name = name
	user.Role = role
	user.IsFTE = isFTE
	if isFTE {
		user.Email = stringPtr(email)
		user.OpsID = nil
	} else {
		user.OpsID = stringPtr(opsID)
		user.Email = nil
	}
	if passwordHash, err := hashPassword(payload.Password); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Password is invalid")
	} else if passwordHash != "" {
		user.PasswordHash = stringPtr(passwordHash)
	}

	if err := database.DB.Save(&user).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to update user")
	}

	events.DefaultBus.Publish(events.Event{
		ID:          strconv.FormatInt(time.Now().UnixNano(), 10),
		Type:        events.UserUpdated,
		OccurredAt:  time.Now(),
		Aggregate:   "user",
		AggregateID: user.ID,
		Action:      "update",
		Payload: map[string]interface{}{
			"name": user.Name,
			"role": user.Role,
		},
	})

	return c.JSON(http.StatusOK, user)
}

func DisableUserAPI(c echo.Context) error {
	if database.DB == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "Database is not configured")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid user ID")
	}

	payload := userPayload{}
	_ = c.Bind(&payload)
	if !canManageRoles(payload.ActorRole) {
		return echo.NewHTTPError(http.StatusForbidden, "Only FTE Ops and FTE MM can disable users")
	}

	if err := database.DB.Model(&models.User{}).Where("id = ?", id).Update("is_active", false).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to disable user")
	}

	events.DefaultBus.Publish(events.Event{
		ID:          strconv.FormatInt(time.Now().UnixNano(), 10),
		Type:        events.UserDisabled,
		OccurredAt:  time.Now(),
		Aggregate:   "user",
		AggregateID: uint(id),
		Action:      "disable",
	})

	return c.JSON(http.StatusOK, map[string]interface{}{"id": id, "is_active": false})
}

func clusterLookupQuery() *gorm.DB {
	return database.DB.Table("clusters").
		Select("id, cluster_name, COALESCE(hub_name, '') AS hub_name, region, COALESCE(dock_number, '') AS dock_number, COALESCE(backlogs, 0) AS backlogs, backlogs_ts")
}

func lookupCluster(id uint) (clusterRecord, error) {
	cluster := clusterRecord{}
	err := clusterLookupQuery().
		Where("id = ?", id).
		First(&cluster).Error
	return cluster, err
}

func updateRequest(c echo.Context, action string, mutate func(*models.Request, requestPayload, time.Time)) error {
	if database.DB == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "Database is not configured")
	}

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request ID")
	}

	payload := requestPayload{}
	if err := c.Bind(&payload); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request payload")
	}

	request := models.Request{}
	if err := database.DB.First(&request, id).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Request not found")
	}

	previousStatus := normalizeStatus(request)
	mutate(&request, payload, time.Now())

	if err := validateRequestAction(action, request); err != nil {
		return err
	}

	if err := database.DB.Save(&request).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Unable to update request")
	}

	eventType := requestEventTypeForAction(action)
	publishRequestEvent(eventType, action, previousStatus, request)

	return c.JSON(http.StatusOK, requestToRow(request))
}

func validateRequestAction(action string, request models.Request) error {
	switch action {
	case "reject", "cancel":
		if strings.TrimSpace(request.RejectionRemarks) == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "Remarks are required")
		}
	case "for-docking":
		if strings.TrimSpace(request.PlateNumber) == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "Plate number is required")
		}
	case "dock":
		if strings.TrimSpace(request.DriverID) == "" || strings.TrimSpace(request.LinehaulTripNo) == "" {
			return echo.NewHTTPError(http.StatusBadRequest, "Driver ID and LH Trip Number are required")
		}
	}
	return nil
}

func publishRequestEvent(eventType, action, previousStatus string, request models.Request) {
	status := normalizeStatus(request)
	payload := map[string]interface{}{
		"cluster":           request.Cluster,
		"region":            request.Region,
		"dock_no":           request.DockNo,
		"plate_number":      request.PlateNumber,
		"linehaul_trip_no":  request.LinehaulTripNo,
		"request_timestamp": formatTime(request.RequestTimestamp),
	}

	if database.DB != nil {
		payloadJSON, _ := json.Marshal(payload)
		eventRecord := models.RequestEvent{
			RequestID:      request.ID,
			EventType:      eventType,
			Action:         action,
			Status:         status,
			PreviousStatus: previousStatus,
			Payload:        payloadJSON,
		}
		if err := database.DB.Create(&eventRecord).Error; err == nil {
			createNotifications(eventType, request)
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

func requestEventTypeForAction(action string) string {
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

func createNotifications(eventType string, request models.Request) {
	roles := notificationRoles(eventType)
	if len(roles) == 0 {
		return
	}

	message := notificationMessage(eventType, request)
	notifications := make([]models.Notification, 0, len(roles))
	for _, role := range roles {
		notifications = append(notifications, models.Notification{
			Role:      role,
			RequestID: request.ID,
			EventType: eventType,
			Message:   message,
		})
	}
	database.DB.Create(&notifications)
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

func canManageRoles(role string) bool {
	role = normalizeRole(role)
	return role == "fte_ops" || role == "fte_mm"
}

func normalizeRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "fte_ops", "fte_mm", "ops_pic", "dock_officer":
		return strings.ToLower(strings.TrimSpace(role))
	case "doc_officer":
		return "dock_officer"
	default:
		return ""
	}
}

func isFTERole(role string) bool {
	switch role {
	case "fte_ops", "fte_mm":
		return true
	default:
		return false
	}
}

func userIdentifierExists(role, email, opsID string) bool {
	var count int64
	if isFTERole(role) {
		database.DB.Model(&models.User{}).
			Where("LOWER(COALESCE(email, '')) = LOWER(?)", strings.TrimSpace(email)).
			Count(&count)
		return count > 0
	}

	database.DB.Model(&models.User{}).
		Where("LOWER(COALESCE(ops_id, '')) = LOWER(?)", strings.TrimSpace(opsID)).
		Count(&count)
	return count > 0
}

func stringPtr(value string) *string {
	return &value
}

func hashPassword(password string) (string, error) {
	password = strings.TrimSpace(password)
	if password == "" {
		return "", nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func randomOTP() (string, error) {
	max := big.NewInt(1000000)
	value, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", value.Int64()), nil
}

func numericClaim(value interface{}) (uint, bool) {
	switch typed := value.(type) {
	case float64:
		return uint(typed), typed > 0
	case int:
		return uint(typed), typed > 0
	case uint:
		return typed, typed > 0
	default:
		return 0, false
	}
}

func ensureUserUniqueID(user *models.User) {
	if strings.TrimSpace(user.UniqueID) != "" {
		return
	}
	prefix := "BR"
	if isFTERole(user.Role) {
		prefix = "FTE"
	}
	user.UniqueID = prefix + "-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	if database.DB != nil && user.ID > 0 {
		database.DB.Model(user).Update("unique_id", user.UniqueID)
	}
}

func setSessionCookie(c echo.Context, user models.User) {
	token, err := signSessionToken(map[string]interface{}{
		"id":        user.ID,
		"unique_id": user.UniqueID,
		"name":      user.Name,
		"role":      user.Role,
		"email":     user.Email,
		"ops_id":    user.OpsID,
		"exp":       time.Now().Add(12 * time.Hour).Unix(),
	})
	if err != nil {
		return
	}

	c.SetCookie(&http.Cookie{
		Name:     "soc5_token",
		Value:    token,
		Path:     "/",
		MaxAge:   int((12 * time.Hour).Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   os.Getenv("APP_ENV") == "production",
	})
}

func readSessionClaims(c echo.Context) (map[string]interface{}, bool) {
	cookie, err := c.Cookie("soc5_token")
	if err != nil || strings.TrimSpace(cookie.Value) == "" {
		return nil, false
	}

	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 3 {
		return nil, false
	}

	unsigned := parts[0] + "." + parts[1]
	if !hmac.Equal([]byte(parts[2]), []byte(signJWTPart(unsigned))) {
		return nil, false
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, false
	}

	claims := map[string]interface{}{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, false
	}

	if exp, ok := claims["exp"].(float64); ok && time.Now().Unix() > int64(exp) {
		return nil, false
	}

	return claims, true
}

func signSessionToken(claims map[string]interface{}) (string, error) {
	headerJSON, err := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	if err != nil {
		return "", err
	}
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	header := base64.RawURLEncoding.EncodeToString(headerJSON)
	payload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	unsigned := header + "." + payload
	return unsigned + "." + signJWTPart(unsigned), nil
}

func signJWTPart(unsigned string) string {
	mac := hmac.New(sha256.New, []byte(sessionSecret()))
	_, _ = mac.Write([]byte(unsigned))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func sessionSecret() string {
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		return secret
	}
	if secret := os.Getenv("APP_SECRET"); secret != "" {
		return secret
	}
	return "soc5-dev-session-secret"
}

func roleLabel(role string) string {
	switch role {
	case "ops_pic":
		return "Ops PIC"
	case "fte_ops":
		return "FTE Ops"
	case "fte_mm":
		return "FTE MM"
	case "dock_officer", "doc_officer":
		return "Dock Officer"
	default:
		return role
	}
}

func loadStats() AppStats {
	stats := AppStats{}
	if database.DB == nil {
		return stats
	}

	start := time.Now().Truncate(24 * time.Hour)
	database.DB.Model(&models.Request{}).Where("request_timestamp >= ?", start).Count(&stats.TotalToday)
	database.DB.Model(&models.Request{}).Where("status IN ? OR status = ''", []string{StatusPending, StatusRejectedByMM}).Count(&stats.PendingOps)
	database.DB.Model(&models.Request{}).Where("status = ?", StatusApproved).Count(&stats.PendingMM)
	database.DB.Model(&models.Request{}).Where("status = ?", StatusForDocking).Count(&stats.ForDocking)
	database.DB.Model(&models.Request{}).Where("status IN ?", []string{StatusAssigned, StatusForDocking, StatusDocked, StatusConfirmed}).Count(&stats.ConfirmedTrucks)
	database.DB.Model(&models.Request{}).Where("status IN ?", []string{StatusRejectedByMM, StatusCancelled}).Count(&stats.Rejected)

	return stats
}

func pendingCount(status string) int64 {
	if database.DB == nil {
		return 0
	}

	var count int64
	if status == StatusPending {
		database.DB.Model(&models.Request{}).Where("status IN ? OR status = ''", []string{StatusPending, StatusRejectedByMM}).Count(&count)
		return count
	}

	database.DB.Model(&models.Request{}).Where("status = ?", status).Count(&count)
	return count
}

func loadRecentRows(limit int) []RequestRow {
	if database.DB == nil {
		return []RequestRow{}
	}

	requests := []models.Request{}
	database.DB.Order("updated_at desc, request_timestamp desc").Limit(limit).Find(&requests)

	rows := make([]RequestRow, 0, len(requests))
	for _, request := range requests {
		rows = append(rows, requestToRow(request))
	}

	return rows
}

func requestTrendWindow(now time.Time) (time.Time, time.Time) {
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

func hourlyRequestTrend(start, end time.Time) []TrendPoint {
	points := make([]TrendPoint, 0, int(end.Sub(start).Hours()))
	for hour := start; hour.Before(end); hour = hour.Add(time.Hour) {
		points = append(points, TrendPoint{
			Label: hour.Format("3PM"),
			Count: 0,
		})
	}

	if database.DB == nil {
		return points
	}

	requests := []models.Request{}
	database.DB.
		Select("request_timestamp").
		Where("request_timestamp >= ? AND request_timestamp < ?", start, end).
		Find(&requests)

	for _, request := range requests {
		if request.RequestTimestamp.Before(start) || !request.RequestTimestamp.Before(end) {
			continue
		}
		index := int(request.RequestTimestamp.Sub(start).Hours())
		if index >= 0 && index < len(points) {
			points[index].Count++
		}
	}

	return points
}

func formatTrendPeriod(start, end time.Time) string {
	return start.Format("Jan 02, 3 PM") + " - " + end.Format("Jan 02, 3 PM")
}

func loadRequestRows(queue string) []RequestRow {
	return queryRequestRows(queue, "", "", "", "")
}

func queryRequestRows(queue, status, search, dateFrom, dateTo string) []RequestRow {
	if database.DB == nil {
		return []RequestRow{}
	}

	query := database.DB.Model(&models.Request{})
	switch strings.ToLower(queue) {
	case "ops":
		query = query.Where("status IN ? OR status = ''", []string{StatusPending, StatusRejectedByMM})
	case "mm":
		query = query.Where("status IN ?", []string{StatusApproved, StatusAssigned})
	case "dock":
		query = query.Where("status IN ?", []string{StatusForDocking, StatusDocked})
	}

	if status = normalizeStatusValue(status); status != "" && status != "ALL" {
		query = query.Where("status = ?", status)
	}

	if search = strings.TrimSpace(search); search != "" {
		like := "%" + search + "%"
		query = query.Where(
			"plate_number LIKE ? OR cluster LIKE ? OR linehaul_trip_no LIKE ? OR driver_id LIKE ? OR region LIKE ? OR dock_no LIKE ?",
			like,
			like,
			like,
			like,
			like,
			like,
		)
	}

	if from, err := time.Parse("2006-01-02", dateFrom); err == nil {
		query = query.Where("request_timestamp >= ?", from)
	}

	if to, err := time.Parse("2006-01-02", dateTo); err == nil {
		query = query.Where("request_timestamp < ?", to.Add(24*time.Hour))
	}

	requests := []models.Request{}
	query.Order("request_timestamp desc").Limit(250).Find(&requests)

	rows := make([]RequestRow, 0, len(requests))
	for _, request := range requests {
		rows = append(rows, requestToRow(request))
	}

	return rows
}

func requestToRow(request models.Request) RequestRow {
	status := normalizeStatus(request)
	return RequestRow{
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

	return time.Time{}, echo.NewHTTPError(http.StatusBadRequest, "Invalid docking time")
}

func redirectForRole(role string) string {
	switch role {
	case "fte_mm":
		return "/midmile/truck-request"
	case "dock_officer", "doc_officer":
		return "/dock/officer"
	default:
		return "/dashboard"
	}
}

func demoUser(loginType, email, opsID string) map[string]interface{} {
	role := "ops_pic"
	name := "Backroom Demo"
	redirect := "/dashboard"

	if loginType == "fte" {
		role = "fte_ops"
		name = "FTE Ops Demo"
		if strings.Contains(strings.ToLower(email), "mm") {
			role = "fte_mm"
			name = "FTE MM Demo"
			redirect = "/midmile/truck-request"
		}
	}

	if loginType == "backroom" && (strings.Contains(strings.ToLower(opsID), "dock") || strings.Contains(strings.ToLower(opsID), "doc")) {
		role = "dock_officer"
		name = "Dock Officer Demo"
		redirect = "/dock/officer"
	}

	return map[string]interface{}{
		"id":       0,
		"name":     name,
		"role":     role,
		"email":    email,
		"ops_id":   opsID,
		"is_fte":   role == "fte_ops" || role == "fte_mm",
		"redirect": redirect,
	}
}
