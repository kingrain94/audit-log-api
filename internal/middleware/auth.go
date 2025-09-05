package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/kingrain94/audit-log-api/internal/config"
	"github.com/kingrain94/audit-log-api/internal/utils"
)

type AuthMiddleware struct {
	config *config.Config
}

func NewAuthMiddleware(config *config.Config) *AuthMiddleware {
	return &AuthMiddleware{
		config: config,
	}
}

func (m *AuthMiddleware) JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		bearerToken := strings.Split(authHeader, " ")
		if len(bearerToken) != 2 || strings.ToLower(bearerToken[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		token := bearerToken[1]
		claims := jwt.MapClaims{}

		_, err := jwt.ParseWithClaims(token, &claims, func(token *jwt.Token) (any, error) {
			return []byte(m.config.JWTSecretKey), nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Set claims in context
		c.Set(string(utils.TenantIDKey), claims["tenant_id"])
		c.Set(string(utils.ClaimsKey), claims)
		c.Next()
	}
}

// RequireRole middleware checks if the user has the required role
func (m *AuthMiddleware) RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, exists := c.Get(string(utils.ClaimsKey))
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "No authentication found"})
			return
		}

		claimsMap, ok := claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Invalid claims type"})
			return
		}

		if !hasRole(claimsMap, role) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			return
		}

		c.Next()
	}
}

func (m *AuthMiddleware) GenerateToken(userID, tenantID string, roles []string) (string, error) {
	claims := jwt.MapClaims{
		"user_id":   userID,
		"tenant_id": tenantID,
		"roles":     roles,
		"exp":       time.Now().Add(time.Duration(m.config.JWTExpirationHours) * time.Hour).Unix(),
		"iat":       time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.config.JWTSecretKey))
}

// hasRole checks if the user has the required role
func hasRole(claims jwt.MapClaims, requiredRole string) bool {
	rolesInterface, exists := claims["roles"]
	if !exists {
		return false
	}

	roles, ok := rolesInterface.([]any)
	if !ok {
		return false
	}

	for _, role := range roles {
		if roleStr, ok := role.(string); ok && roleStr == requiredRole {
			return true
		}
	}
	return false
}
