package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kingrain94/audit-log-api/internal/config"
	"github.com/kingrain94/audit-log-api/internal/repository"
	"github.com/kingrain94/audit-log-api/internal/service/queue"
	"github.com/kingrain94/audit-log-api/pkg/logger"
)

type CleanupWorker struct {
	sqsService   *queue.SQSService
	repository   repository.PostgresRepository
	logger       *logger.Logger
	workerCount  int
	pollInterval time.Duration
	maxMessages  int32
	waitTime     int32
	shutdownChan chan struct{}
	waitGroup    sync.WaitGroup
}

func NewCleanupWorker(
	sqsService *queue.SQSService,
	repository repository.PostgresRepository,
	logger *logger.Logger,
	workerCount int,
	pollInterval time.Duration,
) *CleanupWorker {
	return &CleanupWorker{
		sqsService:   sqsService,
		repository:   repository,
		logger:       logger,
		workerCount:  workerCount,
		pollInterval: pollInterval,
		maxMessages:  10,
		waitTime:     20,
		shutdownChan: make(chan struct{}),
	}
}

func (w *CleanupWorker) Start() {
	w.logger.Info("Starting Cleanup workers...")

	// Start multiple worker goroutines
	for i := 0; i < w.workerCount; i++ {
		w.waitGroup.Add(1)
		go w.runWorker(i)
	}
}

func (w *CleanupWorker) Stop() {
	w.logger.Info("Stopping Cleanup workers...")
	close(w.shutdownChan)
	w.waitGroup.Wait()
	w.logger.Info("All Cleanup workers stopped")
}

func (w *CleanupWorker) runWorker(workerID int) {
	defer w.waitGroup.Done()

	w.logger.Infof("Cleanup Worker %d started", workerID)

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.shutdownChan:
			w.logger.Infof("Cleanup Worker %d shutting down", workerID)
			return
		case <-ticker.C:
			if err := w.processMessages(context.Background()); err != nil {
				w.logger.Errorf("Cleanup Worker %d failed to process messages: %v", workerID, err)
			}
		}
	}
}

func (w *CleanupWorker) processMessages(ctx context.Context) error {
	// Get cleanup queue URL from config
	config := config.DefaultSQSConfig()
	cleanupQueueURL := config.CleanupQueueURL

	messages, err := w.sqsService.ReceiveMessages(ctx, cleanupQueueURL, w.maxMessages, w.waitTime)
	if err != nil {
		return fmt.Errorf("failed to receive messages: %w", err)
	}

	for _, msg := range messages {
		if msg.Message.Type == queue.MessageTypeCleanup {
			if err := w.processCleanupMessage(ctx, msg.Message); err != nil {
				w.logger.Errorf("Failed to process cleanup message: %v", err)
				continue
			}

			// Only delete the message if processing was successful
			if err := w.sqsService.DeleteMessage(ctx, cleanupQueueURL, msg.ReceiptHandle); err != nil {
				w.logger.Errorf("Failed to delete message: %v", err)
			}
		}
	}

	return nil
}

func (w *CleanupWorker) processCleanupMessage(ctx context.Context, msg queue.Message) error {
	w.logger.Infof("Processing cleanup message for tenant %s (before: %s)",
		msg.TenantID, msg.BeforeDate.Format(time.RFC3339))

	// Delete logs before the specified date for the tenant
	deletedCount, err := w.repository.AuditLog().DeleteBeforeDate(ctx, msg.TenantID, msg.BeforeDate)
	if err != nil {
		return fmt.Errorf("failed to delete logs for tenant %s: %w", msg.TenantID, err)
	}

	w.logger.Infof("Successfully deleted %d logs for tenant %s (before: %s)",
		deletedCount, msg.TenantID, msg.BeforeDate.Format(time.RFC3339))

	return nil
}
