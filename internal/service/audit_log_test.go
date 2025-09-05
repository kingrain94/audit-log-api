package service

import (
	"context"
	"testing"
	"time"

	"github.com/kingrain94/audit-log-api/internal/api/dto"
	"github.com/kingrain94/audit-log-api/internal/domain"
	"github.com/kingrain94/audit-log-api/internal/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type AuditLogServiceTestSuite struct {
	suite.Suite
	mockRepo        *mocks.Repository
	mockAuditLog    *mocks.AuditLogRepository
	mockOpenSearch  *mocks.OpenSearchRepository
	mockSQS         *mocks.SQSService
	mockBroadcaster *mocks.WebSocketBroadcaster
	service         *AuditLogService
}

func (s *AuditLogServiceTestSuite) SetupTest() {
	s.mockRepo = new(mocks.Repository)
	s.mockAuditLog = new(mocks.AuditLogRepository)
	s.mockOpenSearch = new(mocks.OpenSearchRepository)
	s.mockSQS = new(mocks.SQSService)
	s.mockBroadcaster = new(mocks.WebSocketBroadcaster)

	s.mockRepo.On("AuditLog").Return(s.mockAuditLog)
	s.mockRepo.On("OpenSearch").Return(s.mockOpenSearch)

	s.service = NewAuditLogService(s.mockRepo, s.mockSQS)
	s.service.SetWebSocketBroadcaster(s.mockBroadcaster)
}

func TestAuditLogService(t *testing.T) {
	suite.Run(t, new(AuditLogServiceTestSuite))
}

func (s *AuditLogServiceTestSuite) TestCreate_Success() {
	// Arrange
	ctx := context.Background()
	req := dto.CreateAuditLogRequest{
		TenantID:     "tenant1",
		UserID:       "user1",
		Action:       "create",
		ResourceType: "user",
		ResourceID:   "resource1",
		Message:      "Test message",
		Severity:     "info",
		IPAddress:    "127.0.0.1",
		UserAgent:    "test-agent",
		SessionID:    "session1",
		Timestamp:    time.Now(),
	}

	s.mockAuditLog.On("Create", ctx, mock.AnythingOfType("*domain.AuditLog")).Return(nil)
	s.mockSQS.On("SendIndexMessage", ctx, mock.AnythingOfType("*domain.AuditLog")).Return(nil)
	s.mockBroadcaster.On("BroadcastLog", mock.AnythingOfType("*dto.AuditLogResponse")).Return()

	// Act
	err := s.service.Create(ctx, req)

	// Assert
	s.NoError(err)
	s.mockAuditLog.AssertExpectations(s.T())
	s.mockSQS.AssertExpectations(s.T())
	s.mockBroadcaster.AssertExpectations(s.T())
}

func (s *AuditLogServiceTestSuite) TestBulkCreate_Success() {
	// Arrange
	ctx := context.Background()
	reqs := []dto.CreateAuditLogRequest{
		{
			TenantID:     "tenant1",
			UserID:       "user1",
			Action:       "create",
			ResourceType: "user",
			ResourceID:   "resource1",
			Message:      "Test message 1",
			Severity:     "info",
			Timestamp:    time.Now(),
		},
		{
			TenantID:     "tenant1",
			UserID:       "user2",
			Action:       "update",
			ResourceType: "user",
			ResourceID:   "resource2",
			Message:      "Test message 2",
			Severity:     "info",
			Timestamp:    time.Now(),
		},
	}

	s.mockAuditLog.On("BulkCreate", ctx, mock.AnythingOfType("[]domain.AuditLog")).Return(nil)
	s.mockSQS.On("SendBulkIndexMessage", ctx, mock.AnythingOfType("[]domain.AuditLog")).Return(nil)
	s.mockBroadcaster.On("BroadcastLog", mock.AnythingOfType("*dto.AuditLogResponse")).Return().Times(2)

	// Act
	err := s.service.BulkCreate(ctx, reqs)

	// Assert
	s.NoError(err)
	s.mockAuditLog.AssertExpectations(s.T())
	s.mockSQS.AssertExpectations(s.T())
	s.mockBroadcaster.AssertExpectations(s.T())
}

func (s *AuditLogServiceTestSuite) TestList_WithSearchCriteria_UsesOpenSearch() {
	// Arrange
	ctx := context.Background()
	filter := &domain.AuditLogFilter{
		UserID:   "user1",
		Action:   "create",
		Page:     1,
		PageSize: 10,
	}

	expectedLogs := []domain.AuditLog{
		{
			ID:        "1",
			TenantID:  "tenant1",
			UserID:    "user1",
			Action:    "create",
			Message:   "Test message",
			CreatedAt: time.Now(),
		},
	}

	s.mockOpenSearch.On("Search", ctx, filter).Return(expectedLogs, nil)

	// Act
	result, err := s.service.List(ctx, filter, true)

	// Assert
	s.NoError(err)
	s.Len(result, 1)
	s.Equal(expectedLogs[0].ID, result[0].ID)
	s.Equal(expectedLogs[0].UserID, result[0].UserID)
	s.mockOpenSearch.AssertExpectations(s.T())
}

func (s *AuditLogServiceTestSuite) TestList_WithoutSearchCriteria_UsesPostgres() {
	// Arrange
	ctx := context.Background()
	filter := &domain.AuditLogFilter{
		Page:     1,
		PageSize: 10,
	}

	expectedLogs := []domain.AuditLog{
		{
			ID:        "1",
			TenantID:  "tenant1",
			UserID:    "user1",
			Action:    "create",
			Message:   "Test message",
			CreatedAt: time.Now(),
		},
	}

	s.mockAuditLog.On("List", ctx, mock.AnythingOfType("domain.AuditLogFilter")).Return(expectedLogs, nil)

	// Act
	result, err := s.service.List(ctx, filter, true)

	// Assert
	s.NoError(err)
	s.Len(result, 1)
	s.Equal(expectedLogs[0].ID, result[0].ID)
	s.Equal(expectedLogs[0].UserID, result[0].UserID)
	s.mockAuditLog.AssertExpectations(s.T())
}
