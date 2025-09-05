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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type TenantHandlerTestSuite struct {
	suite.Suite
	router      *gin.Engine
	mockService *MockTenantService
	handler     *TenantHandler
}

type MockTenantService struct {
	mock.Mock
}

func (m *MockTenantService) Create(ctx context.Context, req dto.CreateTenantRequest) (dto.CreateTenantResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(dto.CreateTenantResponse), args.Error(1)
}

func (m *MockTenantService) GetByID(ctx context.Context, id string) (*domain.Tenant, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Tenant), args.Error(1)
}

func (m *MockTenantService) Update(ctx context.Context, tenant *domain.Tenant) error {
	args := m.Called(ctx, tenant)
	return args.Error(0)
}

func (m *MockTenantService) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockTenantService) List(ctx context.Context) ([]dto.CreateTenantResponse, error) {
	args := m.Called(ctx)
	return args.Get(0).([]dto.CreateTenantResponse), args.Error(1)
}

func (s *TenantHandlerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)
	s.router = gin.New()
	s.mockService = new(MockTenantService)
	s.handler = NewTenantHandler(s.mockService)

	// Setup routes
	s.router.POST("/tenants", s.handler.CreateTenant)
	s.router.GET("/tenants", s.handler.ListTenants)
}

func TestTenantHandler(t *testing.T) {
	suite.Run(t, new(TenantHandlerTestSuite))
}

func (s *TenantHandlerTestSuite) TestCreateTenant_Success() {
	// Arrange
	now := time.Now()
	req := dto.CreateTenantRequest{
		Name: "Test Tenant",
	}

	expectedResponse := dto.CreateTenantResponse{
		ID:        "tenant1",
		Name:      req.Name,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.mockService.On("Create", mock.Anything, req).Return(expectedResponse, nil)

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/tenants", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")

	// Act
	s.handler.CreateTenant(c)

	// Assert
	s.Equal(http.StatusCreated, w.Code)
	var response dto.CreateTenantResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.Equal(expectedResponse.ID, response.ID)
	s.Equal(expectedResponse.Name, response.Name)
	s.mockService.AssertExpectations(s.T())
}

func (s *TenantHandlerTestSuite) TestListTenants_Success() {
	// Arrange
	now := time.Now()
	expectedTenants := []dto.CreateTenantResponse{
		{
			ID:        "tenant1",
			Name:      "Tenant 1",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "tenant2",
			Name:      "Tenant 2",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	s.mockService.On("List", mock.Anything).Return(expectedTenants, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodGet, "/tenants", nil)

	// Act
	s.handler.ListTenants(c)

	// Assert
	s.Equal(http.StatusOK, w.Code)
	var response []dto.CreateTenantResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.Len(response, 2)
	s.Equal(expectedTenants[0].ID, response[0].ID)
	s.Equal(expectedTenants[0].Name, response[0].Name)
	s.Equal(expectedTenants[1].ID, response[1].ID)
	s.Equal(expectedTenants[1].Name, response[1].Name)
	s.mockService.AssertExpectations(s.T())
}
