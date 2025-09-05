package postgres

import (
	"gorm.io/gorm"

	"github.com/kingrain94/audit-log-api/internal/config"
	"github.com/kingrain94/audit-log-api/internal/repository"
)

type postgresRepository struct {
	writerDB     *gorm.DB
	readerDB     *gorm.DB
	auditLogRepo repository.AuditLogRepository
	tenantRepo   repository.TenantRepository
}

func NewPostgresRepository(dbConnections *config.DatabaseConnections) repository.PostgresRepository {
	return &postgresRepository{
		writerDB:     dbConnections.Writer,
		readerDB:     dbConnections.Reader,
		auditLogRepo: NewAuditLogRepository(dbConnections.Writer, dbConnections.Reader),
		tenantRepo:   NewTenantRepository(dbConnections.Writer, dbConnections.Reader),
	}
}

func (r *postgresRepository) AuditLog() repository.AuditLogRepository {
	return r.auditLogRepo
}

func (r *postgresRepository) Tenant() repository.TenantRepository {
	return r.tenantRepo
}
