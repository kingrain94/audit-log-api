package service

import (
	"context"
	"fmt"
	"time"

	"github.com/kingrain94/audit-log-api/internal/api/dto"
	"github.com/kingrain94/audit-log-api/internal/domain"
	"github.com/kingrain94/audit-log-api/internal/repository"
)

//go:generate mockery --name WebSocketBroadcaster --output ../mocks
type WebSocketBroadcaster interface {
	BroadcastLog(log *dto.AuditLogResponse)
}

//go:generate mockery --name SQSService --output ../mocks
type SQSService interface {
	SendIndexMessage(ctx context.Context, log *domain.AuditLog) error
	SendBulkIndexMessage(ctx context.Context, logs []domain.AuditLog) error
	SendArchiveMessage(ctx context.Context, tenantID string, beforeDate time.Time) error
	SendCleanupMessage(ctx context.Context, tenantID string, beforeDate time.Time) error
}

type AuditLogService struct {
	repo        repository.Repository
	sqsSvc      SQSService
	broadcaster WebSocketBroadcaster
}

func NewAuditLogService(repo repository.Repository, sqsSvc SQSService) *AuditLogService {
	return &AuditLogService{
		repo:   repo,
		sqsSvc: sqsSvc,
	}
}

// SetWebSocketBroadcaster sets the WebSocket broadcaster
func (s *AuditLogService) SetWebSocketBroadcaster(broadcaster WebSocketBroadcaster) {
	s.broadcaster = broadcaster
}

func (s *AuditLogService) Create(ctx context.Context, req dto.CreateAuditLogRequest) error {
	auditLog := req.ToAuditLog()

	// Store in PostgreSQL
	if err := s.repo.AuditLog().Create(ctx, auditLog); err != nil {
		return fmt.Errorf("failed to store log in PostgreSQL: %w", err)
	}

	// Send message to SQS for asynchronous indexing
	if err := s.sqsSvc.SendIndexMessage(ctx, auditLog); err != nil {
		fmt.Printf("failed to send index message to SQS: %v\n", err)
	}

	// Broadcast to WebSocket clients if broadcaster is available
	if s.broadcaster != nil {
		s.broadcaster.BroadcastLog(dto.FromAuditLog(auditLog))
	}

	return nil
}

func (s *AuditLogService) BulkCreate(ctx context.Context, req []dto.CreateAuditLogRequest) error {
	auditLogs := make([]domain.AuditLog, len(req))
	for i := range req {
		auditLogs[i] = *req[i].ToAuditLog()
	}

	// Store in PostgreSQL
	if err := s.repo.AuditLog().BulkCreate(ctx, auditLogs); err != nil {
		return fmt.Errorf("failed to bulk store logs in PostgreSQL: %w", err)
	}

	// Send message to SQS for asynchronous bulk indexing
	if err := s.sqsSvc.SendBulkIndexMessage(ctx, auditLogs); err != nil {
		fmt.Printf("failed to send bulk index message to SQS: %v\n", err)
	}

	// Broadcast each log to WebSocket clients if broadcaster is available
	if s.broadcaster != nil {
		for _, log := range auditLogs {
			s.broadcaster.BroadcastLog(dto.FromAuditLog(&log))
		}
	}

	return nil
}

func (s *AuditLogService) GetByID(ctx context.Context, id string) (*dto.AuditLogResponse, error) {
	log, err := s.repo.AuditLog().GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return dto.FromAuditLog(log), nil
}

func (s *AuditLogService) List(ctx context.Context, filter *domain.AuditLogFilter, usePagination bool) ([]dto.AuditLogResponse, error) {
	// Set default values for pagination
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = 10
	}

	// Convert page and page size to limit and offset
	filter.Limit = filter.PageSize
	filter.Offset = (filter.Page - 1) * filter.PageSize

	// Use OpenSearch for searching if there are search criteria benefit from it
	if s.hasSearchCriteria(filter) {
		logs, err := s.repo.OpenSearch().Search(ctx, filter)
		if err != nil {
			return nil, err
		}
		return dto.FromAuditLogs(logs), nil
	}

	// Otherwise, use PostgreSQL for simple listing if there are no search criteria benefit from it
	logs, err := s.repo.AuditLog().List(ctx, *filter)
	if err != nil {
		return nil, err
	}
	return dto.FromAuditLogs(logs), nil
}

func (s *AuditLogService) GetStats(ctx context.Context, filter *domain.AuditLogFilter) (*dto.GetAuditLogStatsResponse, error) {
	// Use OpenSearch for aggregations if available, otherwise fall back to PostgreSQL
	logs, err := s.List(ctx, filter, false)
	if err != nil {
		return nil, err
	}

	stats := &dto.GetAuditLogStatsResponse{
		TotalLogs:      int64(len(logs)),
		ActionCounts:   make(map[string]int64),
		SeverityCounts: make(map[string]int64),
		ResourceCounts: make(map[string]int64),
	}

	for _, log := range logs {
		stats.ActionCounts[log.Action]++
		stats.SeverityCounts[log.Severity]++
		if log.ResourceType != "" {
			stats.ResourceCounts[log.ResourceType]++
		}
	}

	return stats, nil
}

func (s *AuditLogService) GetStatsV2(ctx context.Context, filter *domain.AuditLogFilter) (*dto.GetAuditLogStatsResponse, error) {
	stats, err := s.repo.AuditLog().GetStats(ctx, *filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log stats: %w", err)
	}

	// Convert domain stats to DTO
	response := &dto.GetAuditLogStatsResponse{
		TotalLogs:      stats.TotalLogs,
		ActionCounts:   make(map[string]int64, len(stats.ActionCounts)),
		SeverityCounts: make(map[string]int64, len(stats.SeverityCounts)),
		ResourceCounts: make(map[string]int64, len(stats.ResourceCounts)),
	}

	// Convert ActionType to string
	for action, count := range stats.ActionCounts {
		response.ActionCounts[string(action)] = count
	}

	// Convert SeverityLevel to string
	for severity, count := range stats.SeverityCounts {
		response.SeverityCounts[string(severity)] = count
	}

	// Copy resource counts as is
	for resourceType, count := range stats.ResourceCounts {
		response.ResourceCounts[resourceType] = count
	}

	return response, nil
}

// hasSearchCriteria checks if the filter contains search criteria that would benefit from OpenSearch
func (s *AuditLogService) hasSearchCriteria(filter *domain.AuditLogFilter) bool {
	return filter.UserID != "" ||
		filter.Action != "" ||
		filter.ResourceType != "" ||
		filter.Severity != "" ||
		filter.IPAddress != "" ||
		filter.UserAgent != "" ||
		filter.Message != "" ||
		filter.SessionID != ""
}

// ScheduleArchive schedules an archive operation by sending a message to SQS
func (s *AuditLogService) ScheduleArchive(ctx context.Context, tenantID string, beforeDate time.Time) error {
	return s.sqsSvc.SendArchiveMessage(ctx, tenantID, beforeDate)
}
