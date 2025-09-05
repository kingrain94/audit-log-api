package config

import (
	"os"
	"strconv"
)

type Config struct {
	ServerPort         int    `json:"server_port"`
	JWTSecretKey       string `json:"jwt_secret_key"`
	JWTExpirationHours int    `json:"jwt_expiration_hours"`
	DefaultRateLimit   int    `json:"default_rate_limit"`
	GlobalRateLimit    int    `json:"global_rate_limit"`
}

func Load() (*Config, error) {
	serverPort, _ := strconv.Atoi(os.Getenv("SERVER_PORT"))
	if serverPort == 0 {
		serverPort = 10000
	}

	jwtExpirationHours, _ := strconv.Atoi(os.Getenv("JWT_EXPIRATION_HOURS"))
	if jwtExpirationHours == 0 {
		jwtExpirationHours = 24
	}

	defaultRateLimit, _ := strconv.Atoi(os.Getenv("DEFAULT_RATE_LIMIT"))
	if defaultRateLimit == 0 {
		defaultRateLimit = 1000 // 1000 requests per minute per tenant
	}

	globalRateLimit, _ := strconv.Atoi(os.Getenv("GLOBAL_RATE_LIMIT"))
	if globalRateLimit == 0 {
		globalRateLimit = 10000 // 10000 requests per minute globally per IP
	}

	return &Config{
		ServerPort:         serverPort,
		JWTSecretKey:       os.Getenv("JWT_SECRET_KEY"),
		JWTExpirationHours: jwtExpirationHours,
		DefaultRateLimit:   defaultRateLimit,
		GlobalRateLimit:    globalRateLimit,
	}, nil
}
