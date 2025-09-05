package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/kingrain94/audit-log-api/internal/config"
	"github.com/kingrain94/audit-log-api/internal/repository/postgres"
	"github.com/kingrain94/audit-log-api/internal/service/queue"
	"github.com/kingrain94/audit-log-api/internal/worker"
	"github.com/kingrain94/audit-log-api/pkg/logger"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	// Initialize logger
	appLogger := logger.NewLogger(os.Getenv("APP_ENV"))

	// Initialize PostgreSQL with database connections
	dbConnections, err := config.NewDatabaseConnections()
	if err != nil {
		appLogger.Fatal("Failed to connect to PostgreSQL", err)
	}
	defer dbConnections.Close()

	pgRepo := postgres.NewPostgresRepository(dbConnections)

	// Initialize SQS
	sqsConfig := config.DefaultSQSConfig()
	sqsClient, err := sqsConfig.GetClient()
	if err != nil {
		appLogger.Fatal("Failed to connect to SQS", err)
	}
	sqsService := queue.NewSQSService(sqsClient, sqsConfig)

	// Create cleanup worker
	cleanupWorker := worker.NewCleanupWorker(
		sqsService,
		pgRepo,
		appLogger,
		1,             // worker count
		5*time.Second, // poll interval
	)

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start worker
	go func() {
		appLogger.Info("Starting cleanup worker...")
		cleanupWorker.Start()
	}()

	// Wait for shutdown signal
	<-sigChan
	appLogger.Info("Shutting down cleanup worker...")

	// Stop worker
	cleanupWorker.Stop()
	appLogger.Info("Cleanup worker stopped")
}
