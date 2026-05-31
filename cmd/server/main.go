package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang-dashboard/internal/cache"
	"golang-dashboard/internal/database"
	"golang-dashboard/internal/events"
	"golang-dashboard/internal/models"
	"golang-dashboard/internal/observability"
	"golang-dashboard/internal/routes"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	_ = godotenv.Load()

	if _, err := time.LoadLocation("Asia/Manila"); err != nil {
		log.Println("Warning: Asia/Manila timezone not available, falling back to UTC")
	}

	database.Connect()
	if database.DB != nil {
		if os.Getenv("APP_ENV") != "production" {
			database.DB.AutoMigrate(
				&models.Cluster{},
				&models.User{},
				&models.Request{},
			)
		}
		if os.Getenv("RUN_MIGRATIONS") == "true" {
			if err := database.RunMigrations(database.DB); err != nil {
				log.Fatal("Database migrations failed:", err)
			}
		}
	}

	cache.Connect()

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(observability.Middleware())

	e.Static("/truck_label", "web/truck_label")

	routes.RegisterRoutes(e)

	// Production: serve built React SPA with SPA catch-all
	distIndex := "web/dist/index.html"
	if _, err := os.Stat(distIndex); err == nil {
		e.Static("/assets", "web/dist/assets")
		e.GET("/*", func(c echo.Context) error {
			return c.File(distIndex)
		})
		log.Println("Serving React SPA from web/dist/")
	}

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	host := os.Getenv("APP_HOST")
	if host == "" {
		host = "0.0.0.0"
	}

	addr := host + ":" + port
	log.Println("API server running on", addr)

	// Redis SSE pub/sub: distribute events to this instance's local bus
	serverCtx, serverCancel := context.WithCancel(context.Background())
	defer serverCancel()
	go func() {
		ch := cache.Subscribe(serverCtx, cache.ChannelSSE)
		for payload := range ch {
			var ev events.Event
			if err := json.Unmarshal([]byte(payload), &ev); err == nil {
				events.DefaultBus.Publish(ev)
			}
		}
	}()

	go func() {
		if err := e.Start(addr); err != nil {
			log.Println("Server stopped:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Fatal("Forced shutdown:", err)
	}
	log.Println("Server exited")
}
