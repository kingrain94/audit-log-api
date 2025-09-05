package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"

	"os"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

type Claims struct {
	UserID   string   `json:"user_id"`
	Roles    []string `json:"roles"`
	TenantID string   `json:"tenant_id"`
	jwt.RegisteredClaims
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Define command line flags
	userID := flag.String("user", "", "User ID for the token")
	roles := flag.String("roles", "", "Comma-separated list of roles")
	expirationHours := flag.Int("exp", 24, "Token expiration in hours")
	tenantID := flag.String("tenant", "", "Tenant ID for the token")
	flag.Parse()

	if *userID == "" {
		log.Fatal("User ID is required")
	}

	if *tenantID == "" {
		log.Fatal("Tenant ID is required")
	}

	// Parse roles
	rolesList := []string{}
	if *roles != "" {
		rolesList = strings.Split(*roles, ",")
	}

	// Create claims
	claims := &Claims{
		UserID:   *userID,
		Roles:    rolesList,
		TenantID: *tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(*expirationHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Get JWT secret from environment
	jwtSecret := []byte(getEnvOrDefault("JWT_SECRET_KEY", "your-default-secret-key"))

	// Sign the token
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		log.Fatalf("Error signing token: %v", err)
	}

	fmt.Printf("Generated JWT Token:\n%s\n", tokenString)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
