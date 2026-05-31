package database

import (
	"fmt"
	"log"
	"os"
	"strconv"
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

	sqlDB.SetMaxOpenConns(envInt("DB_MAX_OPEN_CONNS", 25))
	sqlDB.SetMaxIdleConns(envInt("DB_MAX_IDLE_CONNS", 10))
	sqlDB.SetConnMaxLifetime(time.Duration(envInt("DB_CONN_MAX_LIFETIME_MINUTES", 60)) * time.Minute)
	sqlDB.SetConnMaxIdleTime(time.Duration(envInt("DB_CONN_MAX_IDLE_MINUTES", 15)) * time.Minute)

	DB = db

	log.Println("Connected to Postgres")
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
