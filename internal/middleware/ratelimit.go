package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

type rateLimitEntry struct {
	count   int
	resetAt time.Time
}

type ipRateLimiter struct {
	mu      sync.Mutex
	entries map[string]*rateLimitEntry
	limit   int
	window  time.Duration
}

func NewIPRateLimiter(limit int, window time.Duration) echo.MiddlewareFunc {
	rl := &ipRateLimiter{
		entries: make(map[string]*rateLimitEntry),
		limit:   limit,
		window:  window,
	}
	go rl.cleanup()
	return rl.middleware()
}

func (rl *ipRateLimiter) middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ip := c.RealIP()
			rl.mu.Lock()
			now := time.Now()
			entry, ok := rl.entries[ip]
			if !ok || now.After(entry.resetAt) {
				entry = &rateLimitEntry{count: 0, resetAt: now.Add(rl.window)}
				rl.entries[ip] = entry
			}
			entry.count++
			over := entry.count > rl.limit
			rl.mu.Unlock()

			if over {
				return echo.NewHTTPError(http.StatusTooManyRequests, "Too many requests, please try again later")
			}
			return next(c)
		}
	}
}

func (rl *ipRateLimiter) cleanup() {
	for {
		time.Sleep(5 * time.Minute)
		rl.mu.Lock()
		now := time.Now()
		for ip, entry := range rl.entries {
			if now.After(entry.resetAt) {
				delete(rl.entries, ip)
			}
		}
		rl.mu.Unlock()
	}
}
