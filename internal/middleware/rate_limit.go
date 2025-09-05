package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/kingrain94/audit-log-api/internal/config"
	"github.com/kingrain94/audit-log-api/internal/utils"
	"github.com/kingrain94/audit-log-api/pkg/logger"
)

type RateLimitMiddleware struct {
	redis  *redis.Client
	config *config.Config
	logger *logger.Logger
}

func NewRateLimitMiddleware(redis *redis.Client, config *config.Config, logger *logger.Logger) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		redis:  redis,
		config: config,
		logger: logger,
	}
}

// TenantRateLimit implements per-tenant rate limiting
func (m *RateLimitMiddleware) TenantRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, err := utils.GetTenantIDFromContext(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID required for rate limiting"})
			c.Abort()
			return
		}

		// Get tenant-specific rate limit (default: 1000 requests per minute)
		limit := m.getTenantRateLimit(tenantID)

		// Create Redis key for this tenant
		key := fmt.Sprintf("rate_limit:tenant:%s", tenantID)

		// Check current request count
		current, err := m.redis.Get(c.Request.Context(), key).Int()
		if err != nil && err != redis.Nil {
			m.logger.Error("Redis error in rate limiting", err)
			// Allow request to continue on Redis error (fail open)
			c.Next()
			return
		}

		if current >= limit {
			c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
				"limit": limit,
				"reset": time.Now().Add(time.Minute).Unix(),
			})
			c.Abort()
			return
		}

		// Increment counter
		pipe := m.redis.Pipeline()
		pipe.Incr(c.Request.Context(), key)
		pipe.Expire(c.Request.Context(), key, time.Minute)
		_, err = pipe.Exec(c.Request.Context())

		if err != nil {
			m.logger.Error("Redis pipeline error in rate limiting", err)
		}

		// Add rate limit headers
		remaining := limit - (current + 1)
		if remaining < 0 {
			remaining = 0
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10))

		c.Next()
	}
}

// GlobalRateLimit implements global rate limiting based on IP
func (m *RateLimitMiddleware) GlobalRateLimit(limit int) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		key := fmt.Sprintf("rate_limit:global:%s", clientIP)

		// Check current request count
		current, err := m.redis.Get(c.Request.Context(), key).Int()
		if err != nil && err != redis.Nil {
			m.logger.Error("Redis error in global rate limiting", err)
			c.Next()
			return
		}

		if current >= limit {
			c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Global rate limit exceeded",
				"limit": limit,
				"reset": time.Now().Add(time.Minute).Unix(),
			})
			c.Abort()
			return
		}

		// Increment counter
		pipe := m.redis.Pipeline()
		pipe.Incr(c.Request.Context(), key)
		pipe.Expire(c.Request.Context(), key, time.Minute)
		_, err = pipe.Exec(c.Request.Context())

		if err != nil {
			m.logger.Error("Redis pipeline error in global rate limiting", err)
		}

		// Add rate limit headers
		remaining := limit - (current + 1)
		if remaining < 0 {
			remaining = 0
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10))

		c.Next()
	}
}

// getTenantRateLimit retrieves the rate limit for a specific tenant
// In a real implementation, this would query the database
func (m *RateLimitMiddleware) getTenantRateLimit(tenantID string) int {
	// TODO: Query tenant table for custom rate limit
	// For now, return default from config
	if m.config.DefaultRateLimit > 0 {
		return m.config.DefaultRateLimit
	}
	return 1000 // Default: 1000 requests per minute
}
