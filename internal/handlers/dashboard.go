package handlers

import (
        "context"
        "net/http"
        "strconv"
        "strings"
        "time"

        appauth "golang-dashboard/internal/auth"
        "golang-dashboard/internal/cache"
        "golang-dashboard/internal/database"
        "golang-dashboard/internal/events"
        "golang-dashboard/internal/models"

        "github.com/labstack/echo/v4"
        qrcode "github.com/skip2/go-qrcode"
)

const (
        StatusPendingOps = "PENDING_OPS"
        StatusPendingMM  = "PENDING_MM"
        StatusAssigned   = "ASSIGNED"
        StatusForDocking = "FOR_DOCKING"
        StatusDocked     = "DOCKED"
        StatusConfirmed  = "CONFIRMED"
        StatusCanceled   = "CANCELED"
        StatusRejected   = "REJECTED"
)

var manilaLocation *time.Location

func init() {
        loc, err := time.LoadLocation("Asia/Manila")
        if err != nil {
                loc = time.UTC
        }
        manilaLocation = loc
}

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
        Name     string `json:"name" form:"name"`
        Role     string `json:"role" form:"role"`
        Email    string `json:"email" form:"email"`
        OpsID    string `json:"ops_id" form:"ops_id"`
        IsActive bool   `json:"is_active" form:"is_active"`
}

func Dashboard(c echo.Context) error {
        stats := loadStats()
        data := map[string]interface{}{
                "Title":       "Dashboard",
                "ActiveMenu":  "dashboard",
                "Stats":       stats,
                "RecentRows":  loadRecentRows(8),
                "PendingOps":  stats.PendingOps,
                "PendingMM":   stats.PendingMM,
                "PendingDock": stats.ForDocking,
        }

        return c.Render(http.StatusOK, "dashboard.html", data)
}

func LHRequests(c echo.Context) error {
        stats := loadStats()
        data := map[string]interface{}{
                "Title":       "LH Request",
                "ActiveMenu":  "lh-request",
                "Queue":       "ops",
                "Requests":    loadRequestRows(""),
                "PendingOps":  stats.PendingOps,
                "PendingMM":   stats.PendingMM,
                "PendingDock": stats.ForDocking,
        }

        return c.Render(http.StatusOK, "lh_requests.html", data)
}

func TruckRequests(c echo.Context) error {
        stats := loadStats()
        data := map[string]interface{}{
                "Title":       "Truck Request",
                "ActiveMenu":  "truck-request",
                "Queue":       "mm",
                "Requests":    loadRequestRows(""),
                "PendingOps":  stats.PendingOps,
                "PendingMM":   stats.PendingMM,
                "PendingDock": stats.ForDocking,
        }

        return c.Render(http.StatusOK, "truck_requests.html", data)
}

func DockOfficer(c echo.Context) error {
        stats := loadStats()
        data := map[string]interface{}{
                "Title":       "Dock Officer",
                "ActiveMenu":  "dock-officer",
                "Queue":       "dock",
                "Requests":    loadRequestRows("dock"),
                "PendingOps":  stats.PendingOps,
                "PendingMM":   stats.PendingMM,
                "PendingDock": stats.ForDocking,
        }

        return c.Render(http.StatusOK, "dock_officer.html", data)
}

func Settings(c echo.Context) error {
        stats := loadStats()
        data := map[string]interface{}{
                "Title":       "Settings",
                "ActiveMenu":  "settings",
                "PendingOps":  stats.PendingOps,
                "PendingMM":   stats.PendingMM,
                "PendingDock": stats.ForDocking,
        }

        return c.Render(http.StatusOK, "settings.html", data)
}

func LoginAPI(c echo.Context) error {
        if database.DB == nil {
                return echo.NewHTTPError(http.StatusServiceUnavailable, "Database is not configured")
        }

        payload := loginPayload{}
        if err := c.Bind(&payload); err != nil {
                return echo.NewHTTPError(http.StatusBadRequest, "Invalid login payload")
        }

        loginType := strings.ToLower(strings.TrimSpace(payload.LoginType))
        email := strings.TrimSpace(payload.Email)
        opsID := strings.TrimSpace(payload.OpsID)

        user := models.User{}
        query := database.DB.Where("is_active = ?", true)

        switch loginType {
        case "fte":
                if email == "" {
                        return echo.NewHTTPError(http.StatusBadRequest, "Email is required")
                }
                query = query.Where(
                        "LOWER(COALESCE(email, '')) = LOWER(?) AND (is_fte = ? OR role IN ?)",
                        email,
                        true,
                        []string{"fte_ops", "fte_mm"},
                )
        case "backroom":
                if opsID == "" {
                        return echo.NewHTTPError(http.StatusBadRequest, "Ops ID is required")
                }
                query = query.Where(
                        "LOWER(COALESCE(ops_id, '')) = LOWER(?) AND (is_fte = ? OR role IN ?)",
                        opsID,
                        false,
                        []string{"ops_pic", "dock_officer", "doc_officer", "data_team", "admin"},
                )
        default:
                return echo.NewHTTPError(http.StatusBadRequest, "Choose FTE or Backroom")
        }

        if err := query.First(&user).Error; err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, "Invalid credentials")
        }

        tokenStr, err := appauth.IssueToken(user.ID, user.Role, user.Name)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, "Unable to create session")
        }

        appauth.SetSessionCookie(c, tokenStr)

        return c.JSON(http.StatusOK, map[string]interface{}{
                "id":       user.ID,
                "name":     user.Name,
                "role":     user.Role,
                "email":    user.Email,
                "ops_id":   user.OpsID,
                "is_fte":   user.IsFTE,
                "redirect": redirectForRole(user.Role),
        })
}

func LogoutAPI(c echo.Context) error {
        appauth.ClearSessionCookie(c)
        return c.JSON(http.StatusOK, map[string]string{"status": "logged out"})
}

func StatsAPI(c echo.Context) error {
        ctx := context.Background()

        type statsJSON struct {
                TotalToday      int64 `json:"total_today"`
                PendingOps      int64 `json:"pending_ops"`
                PendingMM       int64 `json:"pending_mm"`
                ForDocking      int64 `json:"for_docking"`
                ConfirmedTrucks int64 `json:"confirmed_trucks"`
                Rejected        int64 `json:"rejected"`
        }

        if cached, ok := cache.Get[statsJSON](ctx, cache.KeyStats); ok {
                return c.JSON(http.StatusOK, cached)
        }

        stats := loadStats()
        result := statsJSON{
                TotalToday:      stats.TotalToday,
                PendingOps:      stats.PendingOps,
                PendingMM:       stats.PendingMM,
                ForDocking:      stats.ForDocking,
                ConfirmedTrucks: stats.ConfirmedTrucks,
                Rejected:        stats.Rejected,
        }
        cache.Set(ctx, cache.KeyStats, result, cache.TTLStats)
        return c.JSON(http.StatusOK, result)
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

func CreateRequestAPI(c echo.Context) error {
        payload := requestPayload{}
        if err := c.Bind(&payload); err != nil {
                return echo.NewHTTPError(http.StatusBadRequest, "Invalid request payload")
        }

        clusterName := strings.TrimSpace(payload.Cluster)
        region := strings.TrimSpace(payload.Region)
        dockNo := strings.TrimSpace(payload.DockNo)
        backlogs := payload.Backlogs

        if database.DB != nil && payload.ClusterID > 0 {
                cluster := clusterRecord{}
                if err := database.DB.Table("clusters").
                        Select("id, cluster_name, region, COALESCE(dock_number, '') AS dock_number, COALESCE(backlogs, 0) AS backlogs").
                        Where("id = ? AND active IS DISTINCT FROM ?", payload.ClusterID, false).
                        First(&cluster).Error; err != nil {
                        return echo.NewHTTPError(http.StatusBadRequest, "Invalid cluster")
                }
                clusterName = strings.TrimSpace(cluster.ClusterName)
                region = strings.TrimSpace(cluster.Region)
                backlogs = cluster.Backlogs
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
                TruckSize:        strings.TrimSpace(payload.TruckSize),
                TruckType:        strings.TrimSpace(payload.TruckType),
                OBOpsPIC:         strings.TrimSpace(payload.OBOpsPIC),
                Status:           StatusPendingOps,
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
        if database.DB != nil {
                id, err := strconv.Atoi(c.Param("id"))
                if err == nil && id > 0 {
                        var current models.Request
                        if err := database.DB.Select("status").First(&current, id).Error; err == nil {
                                if current.Status != StatusPendingOps && current.Status != StatusRejected && current.Status != "" {
                                        return echo.NewHTTPError(http.StatusConflict, "Request cannot be edited in its current status")
                                }
                        }
                }
        }

        return updateRequest(c, "edit", func(request *models.Request, payload requestPayload, now time.Time) {
                clusterName, region, dockNo, backlogs, ok := requestDetailsFromPayload(payload)
                if !ok {
                        return
                }

                request.Cluster = clusterName
                request.Region = region
                request.DockNo = dockNo
                request.Backlogs = backlogs
                request.TruckSize = strings.TrimSpace(payload.TruckSize)
                request.TruckType = strings.TrimSpace(payload.TruckType)
                request.OBOpsPIC = strings.TrimSpace(payload.OBOpsPIC)
                request.UpdatedAt = now
        })
}

func CancelRequestAPI(c echo.Context) error {
        return updateRequest(c, "cancel", func(request *models.Request, payload requestPayload, now time.Time) {
                request.Status = StatusCanceled
                request.RejectionRemarks = strings.TrimSpace(payload.Remarks)
                request.RejectedAt = &now
        })
}

type clusterRecord struct {
        ID          uint   `gorm:"column:id"`
        ClusterName string `gorm:"column:cluster_name"`
        Region      string `gorm:"column:region"`
        DockNumber  string `gorm:"column:dock_number"`
        Backlogs    int    `gorm:"column:backlogs"`
}

func requestDetailsFromPayload(payload requestPayload) (string, string, string, int, bool) {
        clusterName := strings.TrimSpace(payload.Cluster)
        region := strings.TrimSpace(payload.Region)
        dockNo := strings.TrimSpace(payload.DockNo)
        backlogs := payload.Backlogs

        if database.DB != nil && payload.ClusterID > 0 {
                cluster := clusterRecord{}
                if err := database.DB.Table("clusters").
                        Select("id, cluster_name, region, COALESCE(dock_number, '') AS dock_number, COALESCE(backlogs, 0) AS backlogs").
                        Where("id = ? AND active IS DISTINCT FROM ?", payload.ClusterID, false).
                        First(&cluster).Error; err != nil {
                        return "", "", "", 0, false
                }
                clusterName = strings.TrimSpace(cluster.ClusterName)
                region = strings.TrimSpace(cluster.Region)
                backlogs = cluster.Backlogs
                if dockNo == "" {
                        dockNo = strings.TrimSpace(cluster.DockNumber)
                }
        }

        return clusterName, region, dockNo, backlogs, clusterName != "" && region != "" && dockNo != ""
}

func ApproveRequestAPI(c echo.Context) error {
        return updateRequest(c, "approve", func(request *models.Request, payload requestPayload, now time.Time) {
                request.Status = StatusPendingMM
                request.OBFTE = strings.TrimSpace(payload.OBFTE)
                request.ApprovedAt = &now
        })
}

func RejectRequestAPI(c echo.Context) error {
        return updateRequest(c, "reject", func(request *models.Request, payload requestPayload, now time.Time) {
                request.Status = StatusRejected
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

func ClustersAPI(c echo.Context) error {
        type clusterOption struct {
                ID       uint   `json:"id"`
                Cluster  string `json:"cluster"`
                Region   string `json:"region"`
                DockNo   string `json:"dock_no"`
                Backlogs int    `json:"backlogs"`
        }

        ctx := context.Background()

        if cached, ok := cache.Get[[]clusterOption](ctx, cache.KeyClusters); ok {
                return c.JSON(http.StatusOK, cached)
        }

        options := []clusterOption{}
        if database.DB == nil {
                return c.JSON(http.StatusOK, options)
        }

        clusters := []clusterRecord{}
        if err := database.DB.Table("clusters").
                Select("id, cluster_name, region, COALESCE(dock_number, '') AS dock_number, COALESCE(backlogs, 0) AS backlogs").
                Where("active IS DISTINCT FROM ? AND COALESCE(cluster_name, '') <> ''", false).
                Order("cluster_name asc, region asc, dock_number asc").
                Find(&clusters).Error; err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, "Unable to load clusters")
        }

        seen := map[string]bool{}
        for _, cluster := range clusters {
                key := cluster.ClusterName + "|" + cluster.Region + "|" + cluster.DockNumber
                if seen[key] {
                        continue
                }
                seen[key] = true
                options = append(options, clusterOption{
                        ID:       cluster.ID,
                        Cluster:  cluster.ClusterName,
                        Region:   cluster.Region,
                        DockNo:   cluster.DockNumber,
                        Backlogs: cluster.Backlogs,
                })
        }

        cache.Set(ctx, cache.KeyClusters, options, cache.TTLClusters)
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

func CreateUserAPI(c echo.Context) error {
        if database.DB == nil {
                return echo.NewHTTPError(http.StatusServiceUnavailable, "Database is not configured")
        }

        sessionUser, ok := c.Get("auth_user").(*appauth.SessionUser)
        if !ok || sessionUser == nil || !canManageRoles(sessionUser.Role) {
                return echo.NewHTTPError(http.StatusForbidden, "Only FTE Ops and FTE MM can add roles")
        }

        payload := userPayload{IsActive: true}
        if err := c.Bind(&payload); err != nil {
                return echo.NewHTTPError(http.StatusBadRequest, "Invalid user payload")
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

        user := models.User{
                Name:     name,
                Role:     role,
                IsFTE:    isFTE,
                IsActive: true,
        }
        if isFTE {
                user.Email = stringPtr(email)
        } else {
                user.OpsID = stringPtr(opsID)
        }

        if err := database.DB.Create(&user).Error; err != nil {
                if isUniqueConstraintError(err) {
                        return echo.NewHTTPError(http.StatusConflict, "A user with this identifier already exists")
                }
                return echo.NewHTTPError(http.StatusInternalServerError, "Unable to create user")
        }

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

        if err := database.DB.Save(&request).Error; err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, "Unable to update request")
        }

        eventType := events.RequestUpdated
        if normalizeStatus(request) != previousStatus {
                eventType = events.RequestStatusChanged
        }
        publishRequestEvent(eventType, action, previousStatus, request)

        return c.JSON(http.StatusOK, requestToRow(request))
}

func publishRequestEvent(eventType, action, previousStatus string, request models.Request) {
        ctx := context.Background()

        // Invalidate stats cache on any mutation
        cache.Delete(ctx, cache.KeyStats)

        status := normalizeStatus(request)
        ev := events.Event{
                Type:           eventType,
                OccurredAt:     time.Now(),
                Aggregate:      "request",
                AggregateID:    request.ID,
                Action:         action,
                Status:         status,
                PreviousStatus: previousStatus,
                Payload: map[string]interface{}{
                        "cluster":           request.Cluster,
                        "region":            request.Region,
                        "dock_no":           request.DockNo,
                        "plate_number":      request.PlateNumber,
                        "linehaul_trip_no":  request.LinehaulTripNo,
                        "request_timestamp": formatTime(request.RequestTimestamp),
                },
        }

        if cache.Client != nil {
                // Multi-instance: publish to Redis; each instance's subscriber distributes locally
                cache.Publish(ctx, cache.ChannelSSE, ev)
        } else {
                // Single-instance: publish directly to local bus
                events.DefaultBus.Publish(ev)
        }
}

func canManageRoles(role string) bool {
        role = normalizeRole(role)
        return role == "fte_ops" || role == "fte_mm" || role == "admin"
}

func normalizeRole(role string) string {
        switch strings.ToLower(strings.TrimSpace(role)) {
        case "fte_ops", "fte_mm", "ops_pic", "data_team", "admin", "dock_officer":
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

func isUniqueConstraintError(err error) bool {
        msg := err.Error()
        return strings.Contains(msg, "unique constraint") ||
                strings.Contains(msg, "duplicate key") ||
                strings.Contains(msg, "violates unique")
}

func stringPtr(value string) *string {
        return &value
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
        case "data_team":
                return "Data Team"
        case "admin":
                return "Admin"
        default:
                return role
        }
}

func manilaToday() time.Time {
        now := time.Now().In(manilaLocation)
        y, m, d := now.Date()
        return time.Date(y, m, d, 0, 0, 0, 0, manilaLocation)
}

func loadStats() AppStats {
        stats := AppStats{}
        if database.DB == nil {
                return stats
        }

        todayStart := manilaToday()

        type statsResult struct {
                TotalToday      int64 `gorm:"column:total_today"`
                PendingOps      int64 `gorm:"column:pending_ops"`
                PendingMM       int64 `gorm:"column:pending_mm"`
                ForDocking      int64 `gorm:"column:for_docking"`
                ConfirmedTrucks int64 `gorm:"column:confirmed_trucks"`
                Rejected        int64 `gorm:"column:rejected"`
        }

        var r statsResult
        database.DB.Raw(`
                SELECT
                        COUNT(*) FILTER (WHERE request_timestamp >= ?) AS total_today,
                        COUNT(*) FILTER (WHERE status IN ('PENDING_OPS','REJECTED') OR status = '') AS pending_ops,
                        COUNT(*) FILTER (WHERE status = 'PENDING_MM') AS pending_mm,
                        COUNT(*) FILTER (WHERE status = 'FOR_DOCKING') AS for_docking,
                        COUNT(*) FILTER (WHERE status IN ('ASSIGNED','FOR_DOCKING','DOCKED','CONFIRMED')) AS confirmed_trucks,
                        COUNT(*) FILTER (WHERE status IN ('REJECTED','CANCELED')) AS rejected
                FROM requests
        `, todayStart).Scan(&r)

        stats.TotalToday = r.TotalToday
        stats.PendingOps = r.PendingOps
        stats.PendingMM = r.PendingMM
        stats.ForDocking = r.ForDocking
        stats.ConfirmedTrucks = r.ConfirmedTrucks
        stats.Rejected = r.Rejected
        return stats
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
        now = now.In(manilaLocation)
        year, month, day := now.Date()
        todayStart := time.Date(year, month, day, 0, 0, 0, 0, manilaLocation)
        todaySixAM := todayStart.Add(6 * time.Hour)
        todaySixPM := todayStart.Add(18 * time.Hour)

        if now.Before(todaySixAM) {
                start := todaySixPM.AddDate(0, 0, -1)
                return start, todaySixAM
        }

        return todaySixPM, todaySixPM.Add(12 * time.Hour)
}

type trendRow struct {
        Hour  time.Time `gorm:"column:hour"`
        Count int       `gorm:"column:count"`
}

func hourlyRequestTrend(start, end time.Time) []TrendPoint {
        points := make([]TrendPoint, 0, int(end.Sub(start).Hours()))
        for hour := start; hour.Before(end); hour = hour.Add(time.Hour) {
                points = append(points, TrendPoint{
                        Label: hour.In(manilaLocation).Format("3PM"),
                        Count: 0,
                })
        }

        if database.DB == nil {
                return points
        }

        var rows []trendRow
        database.DB.Raw(`
                SELECT
                        date_trunc('hour', request_timestamp AT TIME ZONE 'Asia/Manila') AS hour,
                        COUNT(*) AS count
                FROM requests
                WHERE request_timestamp >= ? AND request_timestamp < ?
                GROUP BY 1
                ORDER BY 1
        `, start, end).Scan(&rows)

        for _, row := range rows {
                h := row.Hour.In(manilaLocation)
                index := int(h.Sub(start.In(manilaLocation)).Hours())
                if index >= 0 && index < len(points) {
                        points[index].Count = row.Count
                }
        }

        return points
}

func formatTrendPeriod(start, end time.Time) string {
        return start.In(manilaLocation).Format("Jan 02, 3 PM") + " - " + end.In(manilaLocation).Format("Jan 02, 3 PM")
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
                query = query.Where("status IN ? OR status = ''", []string{StatusPendingOps, StatusRejected})
        case "mm":
                query = query.Where("status IN ?", []string{StatusPendingMM, StatusAssigned})
        case "dock":
                query = query.Where("status IN ?", []string{StatusForDocking, StatusDocked})
        }

        if status = strings.ToUpper(strings.TrimSpace(status)); status != "" && status != "ALL" {
                query = query.Where("status = ?", status)
        }

        if search = strings.TrimSpace(search); search != "" {
                like := "%" + search + "%"
                query = query.Where(
                        "plate_number LIKE ? OR cluster LIKE ? OR linehaul_trip_no LIKE ? OR driver_id LIKE ? OR region LIKE ? OR dock_no LIKE ?",
                        like, like, like, like, like, like,
                )
        }

        if from, err := time.ParseInLocation("2006-01-02", dateFrom, manilaLocation); err == nil {
                query = query.Where("request_timestamp >= ?", from)
        }

        if to, err := time.ParseInLocation("2006-01-02", dateTo, manilaLocation); err == nil {
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
                return request.Status
        }
        if request.RejectedAt != nil || request.RejectionRemarks != "" {
                return StatusRejected
        }
        if request.DockedTime != nil {
                return StatusDocked
        }
        if request.ConfirmedAt != nil {
                return StatusForDocking
        }
        if request.ApprovedAt != nil || request.ProvideTime != nil {
                return StatusPendingMM
        }
        return StatusPendingOps
}

func statusLabel(status string) string {
        switch status {
        case StatusPendingOps:
                return "Pending"
        case StatusPendingMM:
                return "Approved"
        case StatusAssigned:
                return "Assigned"
        case StatusForDocking:
                return "For Docking"
        case StatusDocked:
                return "Docked"
        case StatusConfirmed:
                return "Confirmed"
        case StatusCanceled:
                return "Canceled"
        case StatusRejected:
                return "Rejected"
        default:
                return "Pending"
        }
}

func formatTime(value time.Time) string {
        if value.IsZero() {
                return "-"
        }
        return value.In(manilaLocation).Format("Jan 02, 2006 03:04 PM")
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
        return value.In(manilaLocation).Format("2006-01-02")
}

func parseInputTime(value string) (time.Time, error) {
        value = strings.TrimSpace(value)
        if value == "" {
                return time.Now(), nil
        }

        for _, layout := range []string{"2006-01-02T15:04", time.RFC3339, "2006-01-02 15:04"} {
                if parsed, err := time.ParseInLocation(layout, value, manilaLocation); err == nil {
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
        case "admin":
                return "/dashboard"
        default:
                return "/dashboard"
        }
}
