package service

import (
	"context"
	"time"

	"github.com/kingrain94/audit-log-api/internal/api/dto"
	"github.com/kingrain94/audit-log-api/internal/domain"
	"github.com/kingrain94/audit-log-api/internal/repository"
)

type TenantService struct {
	repo repository.Repository
}

func NewTenantService(repo repository.Repository) *TenantService {
	return &TenantService{repo: repo}
}

func (s *TenantService) Create(ctx context.Context, req dto.CreateTenantRequest) (dto.CreateTenantResponse, error) {
	tenant := &domain.Tenant{
		Name: req.Name,
	}

	createdTenant, err := s.repo.Tenant().Create(ctx, tenant)
	if err != nil {
		return dto.CreateTenantResponse{}, err
	}

	return dto.CreateTenantResponse{
		ID:        createdTenant.ID,
		Name:      createdTenant.Name,
		CreatedAt: createdTenant.CreatedAt,
		UpdatedAt: createdTenant.UpdatedAt,
	}, nil
}

func (s *TenantService) GetByID(ctx context.Context, id string) (*domain.Tenant, error) {
	return s.repo.Tenant().GetByID(ctx, id)
}

func (s *TenantService) Update(ctx context.Context, tenant *domain.Tenant) error {
	tenant.UpdatedAt = time.Now()
	return s.repo.Tenant().Update(ctx, tenant)
}

func (s *TenantService) Delete(ctx context.Context, id string) error {
	return s.repo.Tenant().Delete(ctx, id)
}

func (s *TenantService) List(ctx context.Context) ([]dto.CreateTenantResponse, error) {
	tenants, err := s.repo.Tenant().List(ctx)
	if err != nil {
		return []dto.CreateTenantResponse{}, err
	}

	tenantResponses := make([]dto.CreateTenantResponse, len(tenants))
	for i, tenant := range tenants {
		tenantResponses[i] = dto.CreateTenantResponse{
			ID:        tenant.ID,
			Name:      tenant.Name,
			CreatedAt: tenant.CreatedAt,
			UpdatedAt: tenant.UpdatedAt,
		}
	}
	return tenantResponses, nil
}
