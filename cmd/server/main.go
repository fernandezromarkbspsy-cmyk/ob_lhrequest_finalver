package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"golang-dashboard/internal/database"
	"golang-dashboard/internal/models"
	"golang-dashboard/internal/routes"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	_ = godotenv.Load()

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

	e := echo.New()
	e.HTTPErrorHandler = jsonErrorHandler
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{frontendOrigin()},
		AllowMethods:     []string{echo.GET, echo.POST, echo.PUT, echo.PATCH, echo.DELETE, echo.OPTIONS},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
		AllowCredentials: true,
	}))

	routes.RegisterRoutes(e)

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
	e.Logger.Fatal(e.Start(addr))
}

func frontendOrigin() string {
	if origin := os.Getenv("FRONTEND_URL"); origin != "" {
		return origin
	}
	return "http://localhost:5173"
}

func jsonErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	code := http.StatusInternalServerError
	message := "Internal server error"
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		if text, ok := he.Message.(string); ok {
			message = text
		}
	}

	if err := c.JSON(code, map[string]string{"error": message}); err != nil {
		c.Logger().Error(err)
	}
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
}
