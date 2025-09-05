package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/kingrain94/audit-log-api/docs"
	"github.com/kingrain94/audit-log-api/internal/api"
	"github.com/kingrain94/audit-log-api/internal/config"
	"github.com/kingrain94/audit-log-api/internal/middleware"
	"github.com/kingrain94/audit-log-api/internal/repository/composite"
	"github.com/kingrain94/audit-log-api/internal/service"
	"github.com/kingrain94/audit-log-api/internal/service/pubsub"
	"github.com/kingrain94/audit-log-api/internal/service/queue"
	"github.com/kingrain94/audit-log-api/pkg/logger"
)

// @title           Audit log Swagger API
// @version         1.0
// @description     This is a Audit log swagger server.

// @host      localhost:10000
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

// @externalDocs.description  OpenAPI
// @externalDocs.url          https://swagger.io/resources/open-api/
func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	// Initialize logger
	appLogger := logger.NewLogger(os.Getenv("APP_ENV"))

	cfg, err := config.Load()
	if err != nil {
		appLogger.Fatal("Failed to load config", err)
	}

	dbConnections, err := config.NewDatabaseConnections()
	if err != nil {
		appLogger.Fatal("Failed to connect to database", err)
	}
	defer dbConnections.Close()

	appLogger.Info("Database connections established - writer and reader connected")

	// Initialize OpenSearch
	osConfig := config.DefaultOpenSearchConfig()
	osClient, err := osConfig.GetClient()
	if err != nil {
		appLogger.Fatal("Failed to connect to OpenSearch", err)
	}

	// Initialize Redis
	redisConfig := config.DefaultRedisConfig()
	redisClient, err := redisConfig.GetClient()
	if err != nil {
		appLogger.Fatal("Failed to connect to Redis", err)
	}
	defer redisClient.Close()

	// Initialize Redis pub/sub
	redisPubSub := pubsub.NewRedisPubSub(redisClient, appLogger)

	// Initialize SQS
	sqsConfig := config.DefaultSQSConfig()
	sqsClient, err := sqsConfig.GetClient()
	if err != nil {
		appLogger.Fatal("Failed to connect to SQS", err)
	}
	sqsService := queue.NewSQSService(sqsClient, sqsConfig)

	repo := composite.NewCompositeRepository(dbConnections, osClient, osConfig)

	// Initialize services
	tenantService := service.NewTenantService(repo)
	auditLogService := service.NewAuditLogService(repo, sqsService)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(cfg)
	rateLimitMiddleware := middleware.NewRateLimitMiddleware(redisClient, cfg, appLogger)
	validationMiddleware := middleware.NewValidationMiddleware(appLogger)

	// Initialize server
	server := api.NewServer(
		tenantService,
		auditLogService,
		authMiddleware,
		rateLimitMiddleware,
		validationMiddleware,
		appLogger,
		redisPubSub,
	)

	// Wire up WebSocket broadcaster
	auditLogService.SetWebSocketBroadcaster(server.GetWebSocketHandler())

	// Start WebSocket hub
	server.StartWebSocketHub()

	// Initialize router
	router := gin.Default()

	// Swagger documentation endpoint
	docs.SwaggerInfo.Title = "Audit Log API"
	docs.SwaggerInfo.Description = "A comprehensive audit logging API system"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = fmt.Sprintf("localhost:%d", cfg.ServerPort)
	docs.SwaggerInfo.BasePath = "/api/v1"
	docs.SwaggerInfo.Schemes = []string{"http"}

	// Swagger UI endpoint
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Setup API routes
	apiGroup := router.Group("/api/v1")
	server.SetupRoutes(apiGroup)

	// Start server
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.ServerPort),
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Fatal("Failed to start server", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	appLogger.Info("Shutting down server...")

	// Shutdown the HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		appLogger.Fatal("Server forced to shutdown", err)
	}

	appLogger.Info("Server exiting")
	appLogger.Sync()
}
