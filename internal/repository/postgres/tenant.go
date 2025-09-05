package postgres

import (
	"context"

	"gorm.io/gorm"

	"github.com/kingrain94/audit-log-api/internal/domain"
)

type TenantRepository struct {
	writerDB *gorm.DB
	readerDB *gorm.DB
}

func NewTenantRepository(writerDB, readerDB *gorm.DB) *TenantRepository {
	return &TenantRepository{
		writerDB: writerDB,
		readerDB: readerDB,
	}
}

func (r *TenantRepository) Create(ctx context.Context, tenant *domain.Tenant) (*domain.Tenant, error) {
	if err := r.writerDB.WithContext(ctx).Create(tenant).Error; err != nil {
		return nil, err
	}
	return tenant, nil
}

func (r *TenantRepository) GetByID(ctx context.Context, id string) (*domain.Tenant, error) {
	var tenant domain.Tenant
	if err := r.readerDB.WithContext(ctx).First(&tenant, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *TenantRepository) Update(ctx context.Context, tenant *domain.Tenant) error {
	return r.writerDB.WithContext(ctx).Save(tenant).Error
}

func (r *TenantRepository) Delete(ctx context.Context, id string) error {
	return r.writerDB.WithContext(ctx).Delete(&domain.Tenant{}, "id = ?", id).Error
}

func (r *TenantRepository) List(ctx context.Context) ([]domain.Tenant, error) {
	var tenants []domain.Tenant
	if err := r.readerDB.WithContext(ctx).Find(&tenants).Error; err != nil {
		return nil, err
	}
	return tenants, nil
}
