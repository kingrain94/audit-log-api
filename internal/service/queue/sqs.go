package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/kingrain94/audit-log-api/internal/config"
	"github.com/kingrain94/audit-log-api/internal/domain"
)

type MessageType string

const (
	MessageTypeIndex     MessageType = "INDEX"
	MessageTypeBulkIndex MessageType = "BULK_INDEX"
	MessageTypeArchive   MessageType = "ARCHIVE"
	MessageTypeCleanup   MessageType = "CLEANUP"
)

type Message struct {
	Type      MessageType       `json:"type"`
	TenantID  string            `json:"tenant_id"`
	Logs      []domain.AuditLog `json:"logs,omitempty"`
	Timestamp time.Time         `json:"timestamp"`

	// Fields for archive/cleanup operations
	BeforeDate time.Time `json:"before_date,omitempty"`
}

type ReceivedMessage struct {
	Message       Message
	ReceiptHandle *string
}

type SQSService struct {
	client          *sqs.Client
	indexQueueURL   string
	archiveQueueURL string
	cleanupQueueURL string
}

func NewSQSService(client *sqs.Client, config *config.SQSConfig) *SQSService {
	return &SQSService{
		client:          client,
		indexQueueURL:   config.IndexQueueURL,
		archiveQueueURL: config.ArchiveQueueURL,
		cleanupQueueURL: config.CleanupQueueURL,
	}
}

func (s *SQSService) SendIndexMessage(ctx context.Context, log *domain.AuditLog) error {
	msg := Message{
		Type:      MessageTypeIndex,
		TenantID:  log.TenantID,
		Logs:      []domain.AuditLog{*log},
		Timestamp: log.Timestamp,
	}

	return s.sendMessage(ctx, msg, s.indexQueueURL)
}

func (s *SQSService) SendBulkIndexMessage(ctx context.Context, logs []domain.AuditLog) error {
	if len(logs) == 0 {
		return nil
	}

	msg := Message{
		Type:      MessageTypeBulkIndex,
		TenantID:  logs[0].TenantID,
		Logs:      logs,
		Timestamp: logs[0].Timestamp,
	}

	return s.sendMessage(ctx, msg, s.indexQueueURL)
}

func (s *SQSService) SendArchiveMessage(ctx context.Context, tenantID string, beforeDate time.Time) error {
	msg := Message{
		Type:       MessageTypeArchive,
		TenantID:   tenantID,
		BeforeDate: beforeDate,
		Timestamp:  time.Now(),
	}

	return s.sendMessage(ctx, msg, s.archiveQueueURL)
}

func (s *SQSService) SendCleanupMessage(ctx context.Context, tenantID string, beforeDate time.Time) error {
	msg := Message{
		Type:       MessageTypeCleanup,
		TenantID:   tenantID,
		BeforeDate: beforeDate,
		Timestamp:  time.Now(),
	}

	return s.sendMessage(ctx, msg, s.cleanupQueueURL)
}

func (s *SQSService) sendMessage(ctx context.Context, msg Message, queueURL string) error {
	msgBody, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	input := &sqs.SendMessageInput{
		MessageBody: aws.String(string(msgBody)),
		QueueUrl:    aws.String(queueURL),
	}

	_, err = s.client.SendMessage(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

func (s *SQSService) ReceiveMessages(ctx context.Context, queueURL string, maxMessages int32, waitTimeSeconds int32) ([]ReceivedMessage, error) {
	input := &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: maxMessages,
		WaitTimeSeconds:     waitTimeSeconds,
	}

	output, err := s.client.ReceiveMessage(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to receive messages: %w", err)
	}

	var messages []ReceivedMessage
	for _, msg := range output.Messages {
		var message Message
		if err := json.Unmarshal([]byte(*msg.Body), &message); err != nil {
			return nil, fmt.Errorf("failed to unmarshal message: %w", err)
		}
		messages = append(messages, ReceivedMessage{
			Message:       message,
			ReceiptHandle: msg.ReceiptHandle,
		})
	}

	return messages, nil
}

func (s *SQSService) DeleteMessage(ctx context.Context, queueURL string, receiptHandle *string) error {
	input := &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: receiptHandle,
	}

	_, err := s.client.DeleteMessage(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	return nil
}
