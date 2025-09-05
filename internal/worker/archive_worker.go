package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/kingrain94/audit-log-api/internal/config"
	"github.com/kingrain94/audit-log-api/internal/domain"
	"github.com/kingrain94/audit-log-api/internal/repository"
	"github.com/kingrain94/audit-log-api/internal/service/queue"
	"github.com/kingrain94/audit-log-api/pkg/logger"
)

type ArchiveWorker struct {
	sqsService   *queue.SQSService
	repository   repository.PostgresRepository
	logger       *logger.Logger
	workerCount  int
	pollInterval time.Duration
	maxMessages  int32
	waitTime     int32
	shutdownChan chan struct{}
	waitGroup    sync.WaitGroup
	s3Client     *s3.Client
	s3Config     *config.S3Config
}

func NewArchiveWorker(
	sqsService *queue.SQSService,
	repository repository.PostgresRepository,
	logger *logger.Logger,
	workerCount int,
	pollInterval time.Duration,
	s3Client *s3.Client,
	s3Config *config.S3Config,
) *ArchiveWorker {
	return &ArchiveWorker{
		sqsService:   sqsService,
		repository:   repository,
		logger:       logger,
		workerCount:  workerCount,
		pollInterval: pollInterval,
		maxMessages:  10,
		waitTime:     20,
		shutdownChan: make(chan struct{}),
		s3Client:     s3Client,
		s3Config:     s3Config,
	}
}

func (w *ArchiveWorker) Start() {
	w.logger.Info("Starting Archive workers...")

	// Start multiple worker goroutines
	for i := 0; i < w.workerCount; i++ {
		w.waitGroup.Add(1)
		go w.runWorker(i)
	}
}

func (w *ArchiveWorker) Stop() {
	w.logger.Info("Stopping Archive workers...")
	close(w.shutdownChan)
	w.waitGroup.Wait()
	w.logger.Info("All Archive workers stopped")
}

func (w *ArchiveWorker) runWorker(workerID int) {
	defer w.waitGroup.Done()

	w.logger.Infof("Archive Worker %d started", workerID)

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.shutdownChan:
			w.logger.Infof("Archive Worker %d shutting down", workerID)
			return
		case <-ticker.C:
			if err := w.processMessages(context.Background()); err != nil {
				w.logger.Errorf("Archive Worker %d failed to process messages: %v", workerID, err)
			}
		}
	}
}

func (w *ArchiveWorker) processMessages(ctx context.Context) error {
	// Get archive queue URL from config
	config := config.DefaultSQSConfig()
	archiveQueueURL := config.ArchiveQueueURL

	messages, err := w.sqsService.ReceiveMessages(ctx, archiveQueueURL, w.maxMessages, w.waitTime)
	if err != nil {
		return fmt.Errorf("failed to receive messages: %w", err)
	}

	for _, msg := range messages {
		if msg.Message.Type == queue.MessageTypeArchive {
			if err := w.processArchiveMessage(ctx, msg.Message); err != nil {
				w.logger.Errorf("Failed to process archive message: %v", err)
				continue
			}

			// Only delete the message if processing was successful
			if err := w.sqsService.DeleteMessage(ctx, archiveQueueURL, msg.ReceiptHandle); err != nil {
				w.logger.Errorf("Failed to delete message: %v", err)
			}
		}
	}

	return nil
}

func (w *ArchiveWorker) processArchiveMessage(ctx context.Context, msg queue.Message) error {
	w.logger.Infof("Processing archive message for tenant %s (before: %s)",
		msg.TenantID, msg.BeforeDate.Format(time.RFC3339))

	filter := domain.AuditLogFilter{
		TenantID: msg.TenantID,
		EndTime:  msg.BeforeDate,
	}

	logs, err := w.repository.AuditLog().List(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to fetch logs for archival for tenant %s: %w", msg.TenantID, err)
	}

	if len(logs) == 0 {
		w.logger.Infof("No logs found for archival for tenant %s before %s", msg.TenantID, msg.BeforeDate.Format(time.RFC3339))
		// Still enqueue cleanup message even if no logs found
		return w.enqueueCleanupMessage(ctx, msg.TenantID, msg.BeforeDate)
	}

	w.logger.Infof("Found %d logs to archive for tenant %s before %s", len(logs), msg.TenantID, msg.BeforeDate.Format(time.RFC3339))

	// Archive the logs to S3
	if err := w.archiveLogsToS3(ctx, msg.TenantID, logs, msg.BeforeDate); err != nil {
		return fmt.Errorf("failed to archive logs for tenant %s: %w", msg.TenantID, err)
	}

	w.logger.Infof("Successfully archived %d logs for tenant %s to S3", len(logs), msg.TenantID)

	// Enqueue cleanup message after successful archival
	return w.enqueueCleanupMessage(ctx, msg.TenantID, msg.BeforeDate)
}

func (w *ArchiveWorker) archiveLogsToS3(ctx context.Context, tenantID string, logs []domain.AuditLog, beforeDate time.Time) error {
	// Create S3 key with timestamp and tenant
	s3Key := fmt.Sprintf("audit-logs/%s/audit_logs_%s_before_%s.json",
		tenantID,
		tenantID,
		beforeDate.Format("2006-01-02_15-04-05"))

	// Prepare archive data
	archiveData := map[string]interface{}{
		"tenant_id":   tenantID,
		"before_date": beforeDate,
		"archived_at": time.Now(),
		"log_count":   len(logs),
		"logs":        logs,
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(archiveData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal logs to JSON: %w", err)
	}

	// Upload to S3
	_, err = w.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &w.s3Config.BucketName,
		Key:         &s3Key,
		Body:        bytes.NewReader(jsonData),
		ContentType: &[]string{"application/json"}[0],
		Metadata: map[string]string{
			"tenant-id":   tenantID,
			"archived-at": time.Now().Format(time.RFC3339),
			"log-count":   fmt.Sprintf("%d", len(logs)),
			"before-date": beforeDate.Format(time.RFC3339),
		},
	})

	if err != nil {
		return fmt.Errorf("failed to upload archive to S3: %w", err)
	}

	w.logger.Infof("Successfully uploaded archive to S3: s3://%s/%s", w.s3Config.BucketName, s3Key)
	return nil
}

func (w *ArchiveWorker) enqueueCleanupMessage(ctx context.Context, tenantID string, beforeDate time.Time) error {
	if err := w.sqsService.SendCleanupMessage(ctx, tenantID, beforeDate); err != nil {
		return fmt.Errorf("failed to enqueue cleanup message: %w", err)
	}

	w.logger.Infof("Successfully enqueued cleanup message for tenant %s", tenantID)
	return nil
}
