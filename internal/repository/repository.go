package repository

import (
	"context"
	"time"

	"github.com/kingrain94/audit-log-api/internal/domain"
)

//go:generate mockery --name AuditLogRepository --output ../mocks
type AuditLogRepository interface {
	Create(ctx context.Context, log *domain.AuditLog) error
	GetByID(ctx context.Context, id string) (*domain.AuditLog, error)
	List(ctx context.Context, filter domain.AuditLogFilter) ([]domain.AuditLog, error)
	DeleteBeforeDate(ctx context.Context, tenantID string, beforeDate time.Time) (int64, error)
	BulkCreate(ctx context.Context, logs []domain.AuditLog) error
	GetRecentLogs(ctx context.Context, tenantID string, since time.Time) ([]domain.AuditLog, error)
	GetStats(ctx context.Context, filter domain.AuditLogFilter) (*domain.AuditLogStats, error)
}

//go:generate mockery --name OpenSearchRepository --output ../mocks
type OpenSearchRepository interface {
	Index(ctx context.Context, log *domain.AuditLog) error
	BulkIndex(ctx context.Context, logs []domain.AuditLog) error
	Search(ctx context.Context, filter *domain.AuditLogFilter) ([]domain.AuditLog, error)
	CreateIndex(ctx context.Context, tenantID string, t time.Time) error
	DeleteIndex(ctx context.Context, tenantID string) error
}

//go:generate mockery --name TenantRepository --output ../mocks
type TenantRepository interface {
	Create(ctx context.Context, tenant *domain.Tenant) (*domain.Tenant, error)
	GetByID(ctx context.Context, id string) (*domain.Tenant, error)
	Update(ctx context.Context, tenant *domain.Tenant) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]domain.Tenant, error)
}

//go:generate mockery --name PostgresRepository --output ../mocks
type PostgresRepository interface {
	AuditLog() AuditLogRepository
	Tenant() TenantRepository
}

//go:generate mockery --name Repository --output ../mocks
type Repository interface {
	PostgresRepository
	OpenSearch() OpenSearchRepository
}
