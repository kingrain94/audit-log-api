package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/kingrain94/audit-log-api/internal/config"
	"github.com/kingrain94/audit-log-api/internal/repository/opensearch"
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

	// Initialize OpenSearch
	osConfig := config.DefaultOpenSearchConfig()
	osClient, err := osConfig.GetClient()
	if err != nil {
		appLogger.Fatal("Failed to connect to OpenSearch", err)
	}
	osRepo := opensearch.NewRepository(osClient, osConfig)

	appLogger.Info("OpenSearch connection established for index worker")

	// Initialize SQS
	sqsConfig := config.DefaultSQSConfig()
	sqsClient, err := sqsConfig.GetClient()
	if err != nil {
		appLogger.Fatal("Failed to connect to SQS", err)
	}
	sqsService := queue.NewSQSService(sqsClient, sqsConfig)

	appLogger.Info("SQS connection established for index worker")

	// Initialize SQS worker
	sqsWorker := worker.NewSQSWorker(
		sqsService,
		osRepo,
		appLogger,
		1,             // 3 worker goroutines
		5*time.Second, // Poll every 5 seconds
	)

	// Start the worker
	sqsWorker.Start()
	appLogger.Info("SQS worker started")

	// Wait for interrupt signal to gracefully shutdown the worker
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Stop the worker
	appLogger.Info("Shutting down worker...")
	sqsWorker.Stop()
	appLogger.Info("Worker stopped")
	appLogger.Sync()
}
