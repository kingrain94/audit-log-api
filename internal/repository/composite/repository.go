package composite

import (
	"github.com/kingrain94/audit-log-api/internal/config"
	"github.com/kingrain94/audit-log-api/internal/repository"
	"github.com/kingrain94/audit-log-api/internal/repository/opensearch"
	"github.com/kingrain94/audit-log-api/internal/repository/postgres"
	opensearchclient "github.com/opensearch-project/opensearch-go/v2"
)

type compositeRepository struct {
	postgresRepo repository.PostgresRepository
	osRepo       repository.OpenSearchRepository
}

func NewCompositeRepository(dbConnections *config.DatabaseConnections, osClient *opensearchclient.Client, osConfig *config.OpenSearchConfig) repository.Repository {
	return &compositeRepository{
		postgresRepo: postgres.NewPostgresRepository(dbConnections),
		osRepo:       opensearch.NewRepository(osClient, osConfig),
	}
}

func (r *compositeRepository) AuditLog() repository.AuditLogRepository {
	return r.postgresRepo.AuditLog()
}

func (r *compositeRepository) Tenant() repository.TenantRepository {
	return r.postgresRepo.Tenant()
}

func (r *compositeRepository) OpenSearch() repository.OpenSearchRepository {
	return r.osRepo
}
