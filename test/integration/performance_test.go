package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/kingrain94/audit-log-api/internal/api"
	"github.com/kingrain94/audit-log-api/internal/api/dto"
	"github.com/kingrain94/audit-log-api/internal/mocks"
	"github.com/kingrain94/audit-log-api/pkg/logger"
)

func BenchmarkCreateAuditLog(b *testing.B) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockService := new(mocks.AuditLogService)
	handler := api.NewAuditLogHandler(mockService)
	logger.NewLogger("test")

	// Mock auth middleware that sets tenant context
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", "test-tenant-id")
		c.Set("claims", map[string]interface{}{
			"user_id":   "test-user",
			"tenant_id": "test-tenant-id",
			"roles":     []interface{}{"user"},
		})
		c.Next()
	})

	router.POST("/logs", handler.CreateLog)

	// Mock service response
	mockService.On("Create", mock.Anything, mock.AnythingOfType("dto.CreateAuditLogRequest")).Return(nil)

	// Test payload
	payload := dto.CreateAuditLogRequest{
		TenantID:     "test-tenant-id",
		UserID:       "test-user",
		Action:       "CREATE",
		ResourceType: "user",
		ResourceID:   "user123",
		Severity:     "INFO",
		Message:      "Test audit log entry",
		Timestamp:    time.Now(),
	}

	payloadBytes, _ := json.Marshal(payload)

	b.ResetTimer()
	b.ReportAllocs()

	// Run benchmark
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest("POST", "/logs", bytes.NewBuffer(payloadBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusCreated {
				b.Errorf("Expected status 201, got %d", w.Code)
			}
		}
	})
}

func BenchmarkListAuditLogs(b *testing.B) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockService := new(mocks.AuditLogService)
	handler := api.NewAuditLogHandler(mockService)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", "test-tenant-id")
		c.Set("claims", map[string]interface{}{
			"user_id":   "test-user",
			"tenant_id": "test-tenant-id",
			"roles":     []interface{}{"user"},
		})
		c.Next()
	})

	router.GET("/logs", handler.ListLogs)

	// Mock response
	mockLogs := make([]dto.AuditLogResponse, 100)
	for i := 0; i < 100; i++ {
		mockLogs[i] = dto.AuditLogResponse{
			ID:           fmt.Sprintf("log-%d", i),
			TenantID:     "test-tenant-id",
			UserID:       "test-user",
			Action:       "CREATE",
			ResourceType: "user",
			ResourceID:   fmt.Sprintf("user-%d", i),
			Severity:     "INFO",
			Message:      fmt.Sprintf("Test log entry %d", i),
			Timestamp:    time.Now(),
		}
	}

	mockService.On("List", mock.Anything, mock.AnythingOfType("*domain.AuditLogFilter"), true).Return(mockLogs, nil)

	b.ResetTimer()
	b.ReportAllocs()

	// Run benchmark
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest("GET", "/logs?start_time=2024-01-01T00:00:00Z&end_time=2024-12-31T23:59:59Z", nil)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				b.Errorf("Expected status 200, got %d", w.Code)
			}
		}
	})
}

// TestHighConcurrencyCreateLogs tests the system under high concurrent load
func TestHighConcurrencyCreateLogs(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	mockService := new(mocks.AuditLogService)
	handler := api.NewAuditLogHandler(mockService)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", "test-tenant-id")
		c.Set("claims", map[string]interface{}{
			"user_id":   "test-user",
			"tenant_id": "test-tenant-id",
			"roles":     []interface{}{"user"},
		})
		c.Next()
	})

	router.POST("/logs", handler.CreateLog)

	// Mock service response with some latency simulation
	mockService.On("Create", mock.Anything, mock.AnythingOfType("dto.CreateAuditLogRequest")).Return(nil).Run(func(args mock.Arguments) {
		time.Sleep(1 * time.Millisecond) // Simulate some processing time
	})

	// Test parameters
	numGoroutines := 100
	requestsPerGoroutine := 10
	totalRequests := numGoroutines * requestsPerGoroutine

	payload := dto.CreateAuditLogRequest{
		TenantID:     "test-tenant-id",
		UserID:       "test-user",
		Action:       "CREATE",
		ResourceType: "user",
		ResourceID:   "user123",
		Severity:     "INFO",
		Message:      "High concurrency test",
		Timestamp:    time.Now(),
	}

	payloadBytes, _ := json.Marshal(payload)

	// Metrics
	var successCount int32
	var errorCount int32
	var totalLatency time.Duration
	var maxLatency time.Duration
	var minLatency time.Duration = time.Hour
	var mutex sync.Mutex

	startTime := time.Now()
	var wg sync.WaitGroup

	// Launch concurrent requests
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < requestsPerGoroutine; j++ {
				reqStart := time.Now()

				req, _ := http.NewRequest("POST", "/logs", bytes.NewBuffer(payloadBytes))
				req.Header.Set("Content-Type", "application/json")

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				reqLatency := time.Since(reqStart)

				mutex.Lock()
				totalLatency += reqLatency
				if reqLatency > maxLatency {
					maxLatency = reqLatency
				}
				if reqLatency < minLatency {
					minLatency = reqLatency
				}

				if w.Code == http.StatusCreated {
					successCount++
				} else {
					errorCount++
				}
				mutex.Unlock()
			}
		}()
	}

	wg.Wait()
	totalTime := time.Since(startTime)

	// Calculate metrics
	avgLatency := totalLatency / time.Duration(totalRequests)
	throughput := float64(totalRequests) / totalTime.Seconds()

	// Assertions and reporting
	t.Logf("=== High Concurrency Test Results ===")
	t.Logf("Total requests: %d", totalRequests)
	t.Logf("Successful requests: %d", successCount)
	t.Logf("Failed requests: %d", errorCount)
	t.Logf("Total time: %v", totalTime)
	t.Logf("Throughput: %.2f requests/second", throughput)
	t.Logf("Average latency: %v", avgLatency)
	t.Logf("Min latency: %v", minLatency)
	t.Logf("Max latency: %v", maxLatency)

	// Performance requirements from the challenge
	assert.Equal(t, int32(totalRequests), successCount, "All requests should succeed")
	assert.Equal(t, int32(0), errorCount, "No requests should fail")
	assert.True(t, throughput >= 1000, "Should handle at least 1000 requests/second, got %.2f", throughput)
	assert.True(t, avgLatency < 100*time.Millisecond, "Average latency should be under 100ms, got %v", avgLatency)
}

// TestMemoryUsageUnderLoad tests memory usage under sustained load
func TestMemoryUsageUnderLoad(t *testing.T) {
	// This test would ideally use runtime.MemStats to monitor memory usage
	// For now, we'll run a sustained load test

	gin.SetMode(gin.TestMode)
	mockService := new(mocks.AuditLogService)
	handler := api.NewAuditLogHandler(mockService)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_id", "test-tenant-id")
		c.Set("claims", map[string]interface{}{
			"user_id":   "test-user",
			"tenant_id": "test-tenant-id",
			"roles":     []interface{}{"user"},
		})
		c.Next()
	})

	router.POST("/logs", handler.CreateLog)
	router.GET("/logs", handler.ListLogs)

	mockService.On("Create", mock.Anything, mock.AnythingOfType("dto.CreateAuditLogRequest")).Return(nil)
	mockService.On("List", mock.Anything, mock.AnythingOfType("*domain.AuditLogFilter"), true).Return([]dto.AuditLogResponse{}, nil)

	// Run sustained load for 10 seconds
	duration := 10 * time.Second
	startTime := time.Now()
	requestCount := 0

	for time.Since(startTime) < duration {
		// Create request
		payload := dto.CreateAuditLogRequest{
			TenantID:     "test-tenant-id",
			UserID:       "test-user",
			Action:       "CREATE",
			ResourceType: "user",
			ResourceID:   fmt.Sprintf("user-%d", requestCount),
			Severity:     "INFO",
			Message:      "Sustained load test",
			Timestamp:    time.Now(),
		}

		payloadBytes, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/logs", bytes.NewBuffer(payloadBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if requestCount%100 == 0 {
			// Occasionally do a list request
			req, _ := http.NewRequest("GET", "/logs?start_time=2024-01-01T00:00:00Z&end_time=2024-12-31T23:59:59Z", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}

		requestCount++
	}

	totalTime := time.Since(startTime)
	throughput := float64(requestCount) / totalTime.Seconds()

	t.Logf("=== Sustained Load Test Results ===")
	t.Logf("Duration: %v", duration)
	t.Logf("Total requests: %d", requestCount)
	t.Logf("Average throughput: %.2f requests/second", throughput)

	// Should maintain reasonable throughput under sustained load
	assert.True(t, throughput >= 500, "Should maintain at least 500 requests/second under sustained load")
}
