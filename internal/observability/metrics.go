package observability

import (
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/labstack/echo/v4"
)

type Snapshot struct {
	RequestsTotal    uint64            `json:"requests_total"`
	ResponsesByCode  map[string]uint64 `json:"responses_by_code"`
	InFlightRequests int64             `json:"in_flight_requests"`
	ErrorsTotal      uint64            `json:"errors_total"`
	TotalLatencyMS   uint64            `json:"total_latency_ms"`
}

var requestsTotal atomic.Uint64
var errorsTotal atomic.Uint64
var inFlightRequests atomic.Int64
var totalLatencyMS atomic.Uint64
var responsesByCode = mapCounter{values: map[string]uint64{}}

type mapCounter struct {
	mu     sync.RWMutex
	values map[string]uint64
}

func (c *mapCounter) Add(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key]++
}

func (c *mapCounter) Snapshot() map[string]uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	copy := make(map[string]uint64, len(c.values))
	for k, v := range c.values {
		copy[k] = v
	}
	return copy
}

func Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			requestsTotal.Add(1)
			inFlightRequests.Add(1)
			defer inFlightRequests.Add(-1)

			err := next(c)
			status := c.Response().Status
			if err != nil {
				errorsTotal.Add(1)
				if he, ok := err.(*echo.HTTPError); ok {
					status = he.Code
				} else if status == 0 {
					status = http.StatusInternalServerError
				}
			} else if status >= 500 {
				errorsTotal.Add(1)
			}
			if status == 0 {
				status = http.StatusOK
			}
			responsesByCode.Add(strconv.Itoa(status))
			totalLatencyMS.Add(uint64(time.Since(start).Milliseconds()))
			return err
		}
	}
}

func Handler(c echo.Context) error {
	return c.JSON(http.StatusOK, Snapshot{
		RequestsTotal:    requestsTotal.Load(),
		ResponsesByCode:  responsesByCode.Snapshot(),
		InFlightRequests: inFlightRequests.Load(),
		ErrorsTotal:      errorsTotal.Load(),
		TotalLatencyMS:   totalLatencyMS.Load(),
	})
}
