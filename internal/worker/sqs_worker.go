package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kingrain94/audit-log-api/internal/config"
	"github.com/kingrain94/audit-log-api/internal/repository/opensearch"
	"github.com/kingrain94/audit-log-api/internal/service/queue"
	"github.com/kingrain94/audit-log-api/pkg/logger"
)

type SQSWorker struct {
	sqsService   *queue.SQSService
	osRepository opensearch.Repository
	logger       *logger.Logger
	workerCount  int
	pollInterval time.Duration
	maxMessages  int32
	waitTime     int32
	shutdownChan chan struct{}
	waitGroup    sync.WaitGroup
}

func NewSQSWorker(
	sqsService *queue.SQSService,
	osRepository opensearch.Repository,
	logger *logger.Logger,
	workerCount int,
	pollInterval time.Duration,
) *SQSWorker {
	return &SQSWorker{
		sqsService:   sqsService,
		osRepository: osRepository,
		logger:       logger,
		workerCount:  workerCount,
		pollInterval: pollInterval,
		maxMessages:  10, // Process up to 10 messages at a time
		waitTime:     20, // Long polling: wait up to 20 seconds for messages
		shutdownChan: make(chan struct{}),
	}
}

func (w *SQSWorker) Start() {
	w.logger.Info("Starting SQS workers...")

	// Start multiple worker goroutines
	for i := 0; i < w.workerCount; i++ {
		w.waitGroup.Add(1)
		go w.runWorker(i)
	}
}

func (w *SQSWorker) Stop() {
	w.logger.Info("Stopping SQS workers...")
	close(w.shutdownChan)
	w.waitGroup.Wait()
	w.logger.Info("All SQS workers stopped")
}

func (w *SQSWorker) runWorker(workerID int) {
	defer w.waitGroup.Done()

	w.logger.Infof("Worker %d started", workerID)

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.shutdownChan:
			w.logger.Infof("Worker %d shutting down", workerID)
			return
		case <-ticker.C:
			if err := w.processMessages(context.Background()); err != nil {
				w.logger.Errorf("Worker %d failed to process messages: %v", workerID, err)
			}
		}
	}
}

func (w *SQSWorker) processMessages(ctx context.Context) error {
	// Get index queue URL from config
	config := config.DefaultSQSConfig()
	indexQueueURL := config.IndexQueueURL

	messages, err := w.sqsService.ReceiveMessages(ctx, indexQueueURL, w.maxMessages, w.waitTime)
	if err != nil {
		return fmt.Errorf("failed to receive messages: %w", err)
	}

	for _, msg := range messages {
		if err := w.processMessage(ctx, msg.Message); err != nil {
			w.logger.Errorf("Failed to process message: %v", err)
			continue
		}

		// Only delete the message if processing was successful
		if err := w.sqsService.DeleteMessage(ctx, indexQueueURL, msg.ReceiptHandle); err != nil {
			w.logger.Errorf("Failed to delete message: %v", err)
		}
	}

	return nil
}

func (w *SQSWorker) processMessage(ctx context.Context, msg queue.Message) error {
	w.logger.Infof("Processing message of type %s for tenant %s", msg.Type, msg.TenantID)

	switch msg.Type {
	case queue.MessageTypeIndex:
		if len(msg.Logs) != 1 {
			return fmt.Errorf("invalid number of logs for INDEX message: %d", len(msg.Logs))
		}
		return w.osRepository.Index(ctx, &msg.Logs[0])

	case queue.MessageTypeBulkIndex:
		if len(msg.Logs) == 0 {
			return fmt.Errorf("empty logs array for BULK_INDEX message")
		}
		return w.osRepository.BulkIndex(ctx, msg.Logs)
	default:
		return fmt.Errorf("unknown message type: %s", msg.Type)
	}
}
