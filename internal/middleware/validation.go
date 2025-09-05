package middleware

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/kingrain94/audit-log-api/pkg/logger"
)

type ValidationMiddleware struct {
	logger *logger.Logger
}

func NewValidationMiddleware(logger *logger.Logger) *ValidationMiddleware {
	return &ValidationMiddleware{
		logger: logger,
	}
}

// SanitizeInput middleware sanitizes input to prevent injection attacks
func (m *ValidationMiddleware) SanitizeInput() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Sanitize query parameters
		for key, values := range c.Request.URL.Query() {
			for i, value := range values {
				sanitized := m.sanitizeString(value)
				if sanitized != value {
					m.logger.Info("Sanitized query parameter",
						zap.String("key", key),
						zap.String("original", value),
						zap.String("sanitized", sanitized))
					c.Request.URL.Query()[key][i] = sanitized
				}
			}
		}

		// Sanitize headers (except Authorization)
		for key, values := range c.Request.Header {
			if strings.ToLower(key) == "authorization" {
				continue
			}
			for i, value := range values {
				sanitized := m.sanitizeString(value)
				if sanitized != value {
					m.logger.Info("Sanitized header",
						zap.String("key", key),
						zap.String("original", value),
						zap.String("sanitized", sanitized))
					c.Request.Header[key][i] = sanitized
				}
			}
		}

		c.Next()
	}
}

// ValidateContentType ensures only allowed content types
func (m *ValidationMiddleware) ValidateContentType(allowedTypes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "GET" || c.Request.Method == "DELETE" {
			c.Next()
			return
		}

		contentType := c.GetHeader("Content-Type")
		if contentType == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Content-Type header is required"})
			c.Abort()
			return
		}

		// Remove charset from content type
		contentType = strings.Split(contentType, ";")[0]
		contentType = strings.TrimSpace(contentType)

		allowed := false
		for _, allowedType := range allowedTypes {
			if contentType == allowedType {
				allowed = true
				break
			}
		}

		if !allowed {
			c.JSON(http.StatusUnsupportedMediaType, gin.H{
				"error":         "Unsupported Content-Type",
				"allowed_types": allowedTypes,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// ValidateRequestSize limits request body size
func (m *ValidationMiddleware) ValidateRequestSize(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxSize {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error":         "Request body too large",
				"max_size":      maxSize,
				"received_size": c.Request.ContentLength,
			})
			c.Abort()
			return
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)
		c.Next()
	}
}

// BlockSuspiciousPatterns blocks requests with suspicious patterns
func (m *ValidationMiddleware) BlockSuspiciousPatterns() gin.HandlerFunc {
	// Common SQL injection patterns
	sqlInjectionPatterns := []string{
		`(?i)(\bUNION\b.*\bSELECT\b)`,
		`(?i)(\bOR\b.*=.*\bOR\b)`,
		`(?i)(\bAND\b.*=.*\bAND\b)`,
		`(?i)(\bINSERT\b.*\bINTO\b)`,
		`(?i)(\bDELETE\b.*\bFROM\b)`,
		`(?i)(\bUPDATE\b.*\bSET\b)`,
		`(?i)(\bDROP\b.*\bTABLE\b)`,
		`(?i)(\bALTER\b.*\bTABLE\b)`,
		`--`,
		`/\*.*\*/`,
	}

	// XSS patterns
	xssPatterns := []string{
		`<script.*?>`,
		`javascript:`,
		`onload=`,
		`onclick=`,
		`onerror=`,
		`<iframe.*?>`,
		`<object.*?>`,
		`<embed.*?>`,
	}

	// Path traversal patterns
	pathTraversalPatterns := []string{
		`\.\.\/`,
		`\.\.\\`,
		`%2e%2e%2f`,
		`%2e%2e%5c`,
	}

	allPatterns := append(sqlInjectionPatterns, xssPatterns...)
	allPatterns = append(allPatterns, pathTraversalPatterns...)

	compiledPatterns := make([]*regexp.Regexp, len(allPatterns))
	for i, pattern := range allPatterns {
		compiledPatterns[i] = regexp.MustCompile(pattern)
	}

	return func(c *gin.Context) {
		// Check URL path
		if m.containsSuspiciousPattern(c.Request.URL.Path, compiledPatterns) {
			m.logger.Warn("Blocked suspicious request",
				zap.String("path", c.Request.URL.Path),
				zap.String("ip", c.ClientIP()))
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			c.Abort()
			return
		}

		// Check query parameters
		for key, values := range c.Request.URL.Query() {
			for _, value := range values {
				if m.containsSuspiciousPattern(value, compiledPatterns) {
					m.logger.Warn("Blocked suspicious query parameter",
						zap.String("key", key),
						zap.String("value", value),
						zap.String("ip", c.ClientIP()))
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
					c.Abort()
					return
				}
			}
		}

		// Check headers (except Authorization)
		for key, values := range c.Request.Header {
			if strings.ToLower(key) == "authorization" {
				continue
			}
			for _, value := range values {
				if m.containsSuspiciousPattern(value, compiledPatterns) {
					m.logger.Warn("Blocked suspicious header",
						zap.String("key", key),
						zap.String("value", value),
						zap.String("ip", c.ClientIP()))
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
					c.Abort()
					return
				}
			}
		}

		c.Next()
	}
}

func (m *ValidationMiddleware) sanitizeString(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// Remove control characters (except newline, carriage return, tab)
	result := ""
	for _, r := range input {
		if r >= 32 || r == '\n' || r == '\r' || r == '\t' {
			result += string(r)
		}
	}

	return result
}

func (m *ValidationMiddleware) containsSuspiciousPattern(input string, patterns []*regexp.Regexp) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}
