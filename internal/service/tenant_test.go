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

type TenantServiceTestSuite struct {
	suite.Suite
	mockRepo   *mocks.Repository
	mockTenant *mocks.TenantRepository
	service    *TenantService
}

func (s *TenantServiceTestSuite) SetupTest() {
	s.mockRepo = new(mocks.Repository)
	s.mockTenant = new(mocks.TenantRepository)

	s.mockRepo.On("Tenant").Return(s.mockTenant)

	s.service = NewTenantService(s.mockRepo)
}

func TestTenantService(t *testing.T) {
	suite.Run(t, new(TenantServiceTestSuite))
}

func (s *TenantServiceTestSuite) TestCreate_Success() {
	// Arrange
	ctx := context.Background()
	req := dto.CreateTenantRequest{
		Name: "Test Tenant",
	}

	expectedTenant := &domain.Tenant{
		ID:        "tenant1",
		Name:      req.Name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	s.mockTenant.On("Create", ctx, mock.AnythingOfType("*domain.Tenant")).Return(expectedTenant, nil)

	// Act
	resp, err := s.service.Create(ctx, req)

	// Assert
	s.NoError(err)
	s.Equal(expectedTenant.ID, resp.ID)
	s.Equal(expectedTenant.Name, resp.Name)
	s.Equal(expectedTenant.CreatedAt, resp.CreatedAt)
	s.Equal(expectedTenant.UpdatedAt, resp.UpdatedAt)
	s.mockTenant.AssertExpectations(s.T())
}

func (s *TenantServiceTestSuite) TestGetByID_Success() {
	// Arrange
	ctx := context.Background()
	tenantID := "tenant1"
	expectedTenant := &domain.Tenant{
		ID:        tenantID,
		Name:      "Test Tenant",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	s.mockTenant.On("GetByID", ctx, tenantID).Return(expectedTenant, nil)

	// Act
	tenant, err := s.service.GetByID(ctx, tenantID)

	// Assert
	s.NoError(err)
	s.Equal(expectedTenant.ID, tenant.ID)
	s.Equal(expectedTenant.Name, tenant.Name)
	s.mockTenant.AssertExpectations(s.T())
}

func (s *TenantServiceTestSuite) TestUpdate_Success() {
	// Arrange
	ctx := context.Background()
	tenant := &domain.Tenant{
		ID:        "tenant1",
		Name:      "Updated Tenant",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	s.mockTenant.On("Update", ctx, mock.AnythingOfType("*domain.Tenant")).Return(nil)

	// Act
	err := s.service.Update(ctx, tenant)

	// Assert
	s.NoError(err)
	s.mockTenant.AssertExpectations(s.T())
}

func (s *TenantServiceTestSuite) TestDelete_Success() {
	// Arrange
	ctx := context.Background()
	tenantID := "tenant1"

	s.mockTenant.On("Delete", ctx, tenantID).Return(nil)

	// Act
	err := s.service.Delete(ctx, tenantID)

	// Assert
	s.NoError(err)
	s.mockTenant.AssertExpectations(s.T())
}

func (s *TenantServiceTestSuite) TestList_Success() {
	// Arrange
	ctx := context.Background()
	expectedTenants := []domain.Tenant{
		{
			ID:        "tenant1",
			Name:      "Tenant 1",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "tenant2",
			Name:      "Tenant 2",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	s.mockTenant.On("List", ctx).Return(expectedTenants, nil)

	// Act
	tenants, err := s.service.List(ctx)

	// Assert
	s.NoError(err)
	s.Len(tenants, 2)
	s.Equal(expectedTenants[0].ID, tenants[0].ID)
	s.Equal(expectedTenants[0].Name, tenants[0].Name)
	s.Equal(expectedTenants[1].ID, tenants[1].ID)
	s.Equal(expectedTenants[1].Name, tenants[1].Name)
	s.mockTenant.AssertExpectations(s.T())
}
