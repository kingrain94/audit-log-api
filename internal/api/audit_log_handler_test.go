package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kingrain94/audit-log-api/internal/api/dto"
	"github.com/kingrain94/audit-log-api/internal/domain"
	contextutils "github.com/kingrain94/audit-log-api/internal/utils"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type AuditLogHandlerTestSuite struct {
	suite.Suite
	router      *gin.Engine
	mockService *MockAuditLogService
	handler     *AuditLogHandler
}

type MockAuditLogService struct {
	mock.Mock
}

func (m *MockAuditLogService) Create(ctx context.Context, req dto.CreateAuditLogRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockAuditLogService) BulkCreate(ctx context.Context, reqs []dto.CreateAuditLogRequest) error {
	args := m.Called(ctx, reqs)
	return args.Error(0)
}

func (m *MockAuditLogService) GetByID(ctx context.Context, id string) (*dto.AuditLogResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.AuditLogResponse), args.Error(1)
}

func (m *MockAuditLogService) List(ctx context.Context, filter *domain.AuditLogFilter, usePagination bool) ([]dto.AuditLogResponse, error) {
	args := m.Called(ctx, filter, usePagination)
	return args.Get(0).([]dto.AuditLogResponse), args.Error(1)
}

func (m *MockAuditLogService) GetStats(ctx context.Context, filter *domain.AuditLogFilter) (*dto.GetAuditLogStatsResponse, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(*dto.GetAuditLogStatsResponse), args.Error(1)
}

func (m *MockAuditLogService) GetStatsV2(ctx context.Context, filter *domain.AuditLogFilter) (*dto.GetAuditLogStatsResponse, error) {
	args := m.Called(ctx, filter)
	return args.Get(0).(*dto.GetAuditLogStatsResponse), args.Error(1)
}

func (m *MockAuditLogService) ScheduleArchive(ctx context.Context, tenantID string, beforeDate time.Time) error {
	args := m.Called(ctx, tenantID, beforeDate)
	return args.Error(0)
}

func (s *AuditLogHandlerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	s.router = gin.New()
	s.mockService = new(MockAuditLogService)
	s.handler = NewAuditLogHandler(s.mockService)

	// Setup routes
	s.router.POST("/logs", s.handler.CreateLog)
	s.router.POST("/logs/bulk", s.handler.BulkCreateLogs)
	s.router.GET("/logs/:id", s.handler.GetLog)
	s.router.GET("/logs", s.handler.ListLogs)
}

func TestAuditLogHandler(t *testing.T) {
	suite.Run(t, new(AuditLogHandlerTestSuite))
}

func (s *AuditLogHandlerTestSuite) TestCreateLog_Success() {
	// Arrange
	now := time.Now()
	req := dto.CreateAuditLogRequest{
		TenantID:     "tenant1",
		UserID:       "user1",
		Action:       "create",
		ResourceType: "user",
		ResourceID:   "resource1",
		Message:      "Test message",
		Severity:     "info",
		Timestamp:    now,
	}

	s.mockService.On("Create", mock.Anything, mock.MatchedBy(func(r dto.CreateAuditLogRequest) bool {
		return r.TenantID == req.TenantID &&
			r.UserID == req.UserID &&
			r.Action == req.Action &&
			r.ResourceType == req.ResourceType &&
			r.ResourceID == req.ResourceID &&
			r.Message == req.Message &&
			r.Severity == req.Severity
	})).Return(nil)

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/logs", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(string(contextutils.TenantIDKey), "tenant1")

	// Act
	s.handler.CreateLog(c)

	// Assert
	s.Equal(http.StatusCreated, w.Code)
	s.mockService.AssertExpectations(s.T())
}

func (s *AuditLogHandlerTestSuite) TestBulkCreateLogs_Success() {
	// Arrange
	now := time.Now()
	reqs := []dto.CreateAuditLogRequest{
		{
			TenantID:     "tenant1",
			UserID:       "user1",
			Action:       "create",
			ResourceType: "user",
			ResourceID:   "resource1",
			Message:      "Test message 1",
			Severity:     "info",
			Timestamp:    now,
		},
		{
			TenantID:     "tenant1",
			UserID:       "user2",
			Action:       "update",
			ResourceType: "user",
			ResourceID:   "resource2",
			Message:      "Test message 2",
			Severity:     "info",
			Timestamp:    now,
		},
	}

	s.mockService.On("BulkCreate", mock.Anything, mock.MatchedBy(func(r []dto.CreateAuditLogRequest) bool {
		if len(r) != len(reqs) {
			return false
		}
		for i := range r {
			if r[i].TenantID != reqs[i].TenantID ||
				r[i].UserID != reqs[i].UserID ||
				r[i].Action != reqs[i].Action ||
				r[i].ResourceType != reqs[i].ResourceType ||
				r[i].ResourceID != reqs[i].ResourceID ||
				r[i].Message != reqs[i].Message ||
				r[i].Severity != reqs[i].Severity {
				return false
			}
		}
		return true
	})).Return(nil)

	body, _ := json.Marshal(reqs)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/logs/bulk", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(string(contextutils.TenantIDKey), "tenant1")

	// Act
	s.handler.BulkCreateLogs(c)

	// Assert
	s.Equal(http.StatusCreated, w.Code)
	s.mockService.AssertExpectations(s.T())
}

func (s *AuditLogHandlerTestSuite) TestGetLog_Success() {
	// Arrange
	logID := "log1"
	expectedLog := &dto.AuditLogResponse{
		ID:        logID,
		TenantID:  "tenant1",
		UserID:    "user1",
		Action:    "create",
		Message:   "Test message",
		Timestamp: time.Now(),
	}

	s.mockService.On("GetByID", mock.Anything, logID).Return(expectedLog, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/logs/"+logID, nil)
	c.Params = []gin.Param{{Key: "id", Value: logID}}
	c.Set(string(contextutils.TenantIDKey), "tenant1")

	// Act
	s.handler.GetLog(c)

	// Assert
	s.Equal(http.StatusOK, w.Code)
	var response dto.AuditLogResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.Equal(expectedLog.ID, response.ID)
	s.mockService.AssertExpectations(s.T())
}

func (s *AuditLogHandlerTestSuite) TestListLogs_Success() {
	// Arrange
	expectedLogs := []dto.AuditLogResponse{
		{
			ID:        "log1",
			TenantID:  "tenant1",
			UserID:    "user1",
			Action:    "create",
			Message:   "Test message 1",
			Timestamp: time.Now(),
		},
		{
			ID:        "log2",
			TenantID:  "tenant1",
			UserID:    "user2",
			Action:    "update",
			Message:   "Test message 2",
			Timestamp: time.Now(),
		},
	}

	s.mockService.On("List", mock.Anything, mock.AnythingOfType("*domain.AuditLogFilter"), true).Return(expectedLogs, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/logs?page=1&page_size=10&start_time=2024-01-01T00:00:00Z&end_time=2024-12-31T23:59:59Z", nil)
	c.Set(string(contextutils.TenantIDKey), "tenant1")

	// Act
	s.handler.ListLogs(c)

	// Assert
	s.Equal(http.StatusOK, w.Code)
	var response []dto.AuditLogResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.Len(response, 2)
	s.Equal(expectedLogs[0].ID, response[0].ID)
	s.Equal(expectedLogs[1].ID, response[1].ID)
	s.mockService.AssertExpectations(s.T())
}
