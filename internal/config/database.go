package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type ConnectionPoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

func DefaultConnectionPoolConfig() *ConnectionPoolConfig {
	return &ConnectionPoolConfig{
		MaxOpenConns:    50,
		MaxIdleConns:    10,
		ConnMaxLifetime: 1 * time.Hour,
	}
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntWithDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvDurationWithDefault returns environment variable as duration or default if not set
func getEnvDurationWithDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// getWriterConfig loads writer database configuration from environment variables
func getWriterConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Host:     getEnvWithDefault("POSTGRES_WRITER_HOST", "localhost"),
		Port:     getEnvWithDefault("POSTGRES_WRITER_PORT", "5432"),
		User:     getEnvWithDefault("POSTGRES_WRITER_USER", "postgres"),
		Password: getEnvWithDefault("POSTGRES_WRITER_PASSWORD", ""),
		DBName:   getEnvWithDefault("POSTGRES_WRITER_DB_NAME", "audit_log"),
		SSLMode:  getEnvWithDefault("POSTGRES_WRITER_SSL_MODE", "disable"),
	}
}

// getReaderConfig loads reader database configuration from environment variables
func getReaderConfig() *DatabaseConfig {
	return &DatabaseConfig{
		Host:     getEnvWithDefault("POSTGRES_READER_HOST", "localhost"),
		Port:     getEnvWithDefault("POSTGRES_READER_PORT", "5432"),
		User:     getEnvWithDefault("POSTGRES_READER_USER", "postgres"),
		Password: getEnvWithDefault("POSTGRES_READER_PASSWORD", ""),
		DBName:   getEnvWithDefault("POSTGRES_READER_DB_NAME", "audit_log"),
		SSLMode:  getEnvWithDefault("POSTGRES_READER_SSL_MODE", "disable"),
	}
}

// getConnectionPoolConfig loads connection pool configuration from environment variables
func getConnectionPoolConfig() *ConnectionPoolConfig {
	return &ConnectionPoolConfig{
		MaxOpenConns:    getEnvIntWithDefault("DB_MAX_OPEN_CONNS", 50),
		MaxIdleConns:    getEnvIntWithDefault("DB_MAX_IDLE_CONNS", 10),
		ConnMaxLifetime: getEnvDurationWithDefault("DB_CONN_MAX_LIFETIME", 1*time.Hour),
	}
}

// buildDSN creates PostgreSQL connection string from configuration
func (c *DatabaseConfig) buildDSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}

// configureConnectionPool applies connection pool settings to the database connection
func configureConnectionPool(gormDB *gorm.DB, poolConfig *ConnectionPoolConfig) error {
	sqlDB, err := gormDB.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB from gorm.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(poolConfig.MaxOpenConns)
	sqlDB.SetMaxIdleConns(poolConfig.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(poolConfig.ConnMaxLifetime)

	return nil
}

// createDatabaseConnection creates a GORM database connection with connection pool tuning
func createDatabaseConnection(config *DatabaseConfig, poolConfig *ConnectionPoolConfig) (*gorm.DB, error) {
	dsn := config.buildDSN()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	if err := configureConnectionPool(db, poolConfig); err != nil {
		return nil, fmt.Errorf("failed to configure connection pool: %w", err)
	}

	return db, nil
}

// NewWriterDatabase creates a database connection optimized for write operations
func NewWriterDatabase() (*gorm.DB, error) {
	config := getWriterConfig()
	poolConfig := getConnectionPoolConfig()
	return createDatabaseConnection(config, poolConfig)
}

// NewReaderDatabase creates a database connection optimized for read operations
func NewReaderDatabase() (*gorm.DB, error) {
	config := getReaderConfig()
	poolConfig := getConnectionPoolConfig()
	return createDatabaseConnection(config, poolConfig)
}

// DatabaseConnections holds both writer and reader database connections
type DatabaseConnections struct {
	Writer *gorm.DB
	Reader *gorm.DB
}

// NewDatabaseConnections creates both writer and reader database connections
func NewDatabaseConnections() (*DatabaseConnections, error) {
	writer, err := NewWriterDatabase()
	if err != nil {
		return nil, fmt.Errorf("failed to create writer database connection: %w", err)
	}

	reader, err := NewReaderDatabase()
	if err != nil {
		return nil, fmt.Errorf("failed to create reader database connection: %w", err)
	}

	return &DatabaseConnections{
		Writer: writer,
		Reader: reader,
	}, nil
}

// Close closes both writer and reader database connections
func (dc *DatabaseConnections) Close() error {
	var writerErr, readerErr error

	if dc.Writer != nil {
		if sqlDB, err := dc.Writer.DB(); err == nil {
			writerErr = sqlDB.Close()
		}
	}

	if dc.Reader != nil {
		if sqlDB, err := dc.Reader.DB(); err == nil {
			readerErr = sqlDB.Close()
		}
	}

	if writerErr != nil {
		return fmt.Errorf("failed to close writer database connection: %w", writerErr)
	}
	if readerErr != nil {
		return fmt.Errorf("failed to close reader database connection: %w", readerErr)
	}

	return nil
}
