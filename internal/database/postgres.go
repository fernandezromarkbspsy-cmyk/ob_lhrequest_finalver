package database

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DB_DSN")
	}

	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	name := os.Getenv("DB_NAME")
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASSWORD")
	sslmode := os.Getenv("DB_SSLMODE")

	if sslmode == "" {
		sslmode = "require"
	}

	if dsn == "" && (host == "" || port == "" || name == "" || user == "" || pass == "") {
		log.Println("Database connection skipped: DB_HOST, DB_PORT, DB_NAME, DB_USER, and DB_PASSWORD are required")
		return
	}

	if dsn == "" && strings.HasPrefix(host, "db.") && !strings.HasSuffix(host, ".supabase.co") {
		log.Fatal("Database connection failed: Supabase direct host must look like db.<project-ref>.supabase.co")
	}

	if dsn == "" {
		dsn = fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Manila connect_timeout=10",
			host,
			user,
			pass,
			name,
			port,
			sslmode,
		)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Database pool failed:", err)
	}

	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	DB = db

	log.Println("Connected to Supabase Postgres")
}
