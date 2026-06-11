package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang-dashboard/internal/database"
	"golang-dashboard/internal/jobs"
	"golang-dashboard/internal/models"
	"golang-dashboard/internal/routes"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	jobs.Default.Start(2)

	database.Connect()

	if database.DB != nil {
		database.DB.AutoMigrate(
			&models.Cluster{},
			&models.User{},
			&models.Request{},
			&models.RequestEvent{},
			&models.Notification{},
			&models.UserOTP{},
		)
		ensureWorkflowConstraints()
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(requestTimeout(60 * time.Second))
	r.Use(securityHeaders)
	r.Use(rateLimiter(rateLimitConfig()))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{frontendOrigin()},
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	routes.RegisterRoutes(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("APP_PORT")
	}
	if port == "" {
		port = "8080"
	}
	host := os.Getenv("APP_HOST")
	if host == "" && os.Getenv("PORT") == "" {
		host = "127.0.0.1"
	}

	addr := ":" + port
	if host != "" {
		addr = host + ":" + port
	}
	log.Println("Server running on", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

type rateLimitBucket struct {
	count      int
	resetAfter time.Time
}

func rateLimitConfig() (int, time.Duration) {
	limit, err := strconv.Atoi(os.Getenv("RATE_LIMIT_PER_MINUTE"))
	if err != nil || limit <= 0 {
		limit = 120
	}
	return limit, time.Minute
}

func requestTimeout(timeout time.Duration) func(http.Handler) http.Handler {
	timeoutMiddleware := middleware.Timeout(timeout)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/events") || strings.HasPrefix(r.URL.Path, "/api/realtime/notifications") || strings.HasPrefix(r.URL.Path, "/api/ws") {
				next.ServeHTTP(w, r)
				return
			}
			timeoutMiddleware(next).ServeHTTP(w, r)
		})
	}
}

func rateLimiter(limit int, window time.Duration) func(http.Handler) http.Handler {
	var mu sync.Mutex
	buckets := map[string]rateLimitBucket{}

	go func() {
		ticker := time.NewTicker(window)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			mu.Lock()
			for key, bucket := range buckets {
				if now.After(bucket.resetAfter) {
					delete(buckets, key)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := clientKey(r)
			now := time.Now()

			mu.Lock()
			bucket := buckets[key]
			if now.After(bucket.resetAfter) {
				bucket = rateLimitBucket{resetAfter: now.Add(window)}
			}
			bucket.count++
			buckets[key] = bucket
			remaining := limit - bucket.count
			mu.Unlock()

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(max(0, remaining)))
			if bucket.count > limit {
				w.Header().Set("Retry-After", strconv.Itoa(int(time.Until(bucket.resetAfter).Seconds())))
				http.Error(w, `{"error":"Rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func clientKey(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		return forwarded
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}

func frontendOrigin() string {
	if origin := os.Getenv("FRONTEND_URL"); origin != "" {
		return origin
	}
	return "http://localhost:5173"
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

func ensureWorkflowConstraints() {
	if err := database.DB.Exec(`
UPDATE requests
SET status = CASE status
	WHEN 'PENDING_OPS' THEN 'PENDING'
	WHEN 'PENDING_MM' THEN 'APPROVED'
	WHEN 'REJECTED' THEN 'REJECTED_BY_MM'
	WHEN 'CANCELED' THEN 'CANCELLED'
	ELSE status
END
WHERE status IN ('PENDING_OPS', 'PENDING_MM', 'REJECTED', 'CANCELED')
`).Error; err != nil {
		log.Println("Unable to normalize request statuses:", err)
	}

	statuses := "'PENDING', 'APPROVED', 'ASSIGNED', 'FOR_DOCKING', 'DOCKED', 'CONFIRMED', 'REJECTED_BY_MM', 'CANCELLED'"
	if err := database.DB.Exec(`ALTER TABLE requests DROP CONSTRAINT IF EXISTS requests_status_check`).Error; err != nil {
		log.Println("Unable to drop request status constraint:", err)
	}
	if err := database.DB.Exec(fmt.Sprintf(`ALTER TABLE requests ADD CONSTRAINT requests_status_check CHECK (status IN (%s))`, statuses)).Error; err != nil {
		log.Println("Unable to add request status constraint:", err)
	}

	roles := "'fte_ops', 'fte_mm', 'ops_pic', 'dock_officer', 'doc_officer'"
	if err := database.DB.Exec(`
DO $$
DECLARE
	role_constraint text;
BEGIN
	FOR role_constraint IN
		SELECT conname
		FROM pg_constraint
		WHERE conrelid = 'users'::regclass
			AND contype = 'c'
			AND pg_get_constraintdef(oid) ILIKE '%role%'
	LOOP
		EXECUTE format('ALTER TABLE users DROP CONSTRAINT %I', role_constraint);
	END LOOP;
END $$;
`).Error; err != nil {
		log.Println("Unable to drop user role constraint:", err)
	}
	if err := database.DB.Exec(fmt.Sprintf(`ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN (%s))`, roles)).Error; err != nil {
		log.Println("Unable to add user role constraint:", err)
	}

	indexStatements := []string{
		`CREATE INDEX IF NOT EXISTS idx_users_lower_email_active ON users (LOWER(COALESCE(email, '')), role, is_active)`,
		`CREATE INDEX IF NOT EXISTS idx_users_lower_ops_id_active ON users (LOWER(COALESCE(ops_id, '')), role, is_active)`,
		`CREATE INDEX IF NOT EXISTS idx_requests_status_created ON requests (status, request_timestamp DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_requests_plate_lower ON requests (LOWER(COALESCE(plate_number, '')))`,
		`CREATE INDEX IF NOT EXISTS idx_requests_trip_lower ON requests (LOWER(COALESCE(linehaul_trip_no, '')))`,
		`CREATE INDEX IF NOT EXISTS idx_requests_driver_lower ON requests (LOWER(COALESCE(driver_id, '')))`,
	}
	for _, statement := range indexStatements {
		if err := database.DB.Exec(statement).Error; err != nil {
			log.Println("Unable to ensure performance index:", err)
		}
	}
}
