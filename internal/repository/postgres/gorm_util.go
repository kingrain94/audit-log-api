package postgres

import (
	"context"

	"github.com/kingrain94/audit-log-api/internal/utils"
	"gorm.io/gorm"
)

// getTenantScope returns a scoped database instance with tenant isolation
func getTenantScope(db *gorm.DB, ctx context.Context) (*gorm.DB, error) {
	tenantID, err := utils.GetTenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return db.WithContext(ctx).Where("tenant_id = ?", tenantID), nil
}
