package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/kingrain94/audit-log-api/internal/domain"
	"github.com/kingrain94/audit-log-api/internal/utils"
)

type AuditLogRepository struct {
	writerDB *gorm.DB
	readerDB *gorm.DB
}

func NewAuditLogRepository(writerDB, readerDB *gorm.DB) *AuditLogRepository {
	return &AuditLogRepository{
		writerDB: writerDB,
		readerDB: readerDB,
	}
}

func (r *AuditLogRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	if log.ID == "" {
		log.ID = uuid.New().String()
	}

	// Use writer database for create operations
	return r.writerDB.WithContext(ctx).Create(log).Error
}

func (r *AuditLogRepository) GetByID(ctx context.Context, id string) (*domain.AuditLog, error) {
	var log domain.AuditLog

	// Use reader database for read operations
	db, err := getTenantScope(r.readerDB, ctx)
	if err != nil {
		return nil, err
	}

	if err := db.First(&log, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

func (r *AuditLogRepository) List(ctx context.Context, filter domain.AuditLogFilter) ([]domain.AuditLog, error) {
	var logs []domain.AuditLog

	// Use reader database for read operations
	db := r.readerDB.WithContext(ctx)
	if filter.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	} else {
		db = db.Where("tenant_id = ?", filter.TenantID)
	}

	// Apply additional filters
	if filter.UserID != "" {
		db = db.Where("user_id = ?", filter.UserID)
	}
	if filter.Action != "" {
		db = db.Where("action = ?", filter.Action)
	}
	if filter.ResourceType != "" {
		db = db.Where("resource_type = ?", filter.ResourceType)
	}
	if filter.ResourceID != "" {
		db = db.Where("resource_id = ?", filter.ResourceID)
	}
	if filter.Severity != "" {
		db = db.Where("severity = ?", filter.Severity)
	}
	if !filter.StartTime.IsZero() {
		db = db.Where("timestamp >= ?", filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		db = db.Where("timestamp <= ?", filter.EndTime)
	}

	// Apply pagination
	if filter.Limit > 0 {
		db = db.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		db = db.Offset(filter.Offset)
	}

	// Apply sorting
	db = db.Order("timestamp DESC")

	if err := db.Find(&logs).Error; err != nil {
		return nil, err
	}

	return logs, nil
}

func (r *AuditLogRepository) DeleteBeforeDate(ctx context.Context, tenantID string, beforeDate time.Time) (int64, error) {
	// Use writer database for delete operations
	db := r.writerDB.WithContext(ctx)

	result := db.Where("tenant_id = ? AND timestamp < ?", tenantID, beforeDate).
		Delete(&domain.AuditLog{})

	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}

func (r *AuditLogRepository) BulkCreate(ctx context.Context, logs []domain.AuditLog) error {
	tenantID, err := utils.GetTenantIDFromContext(ctx)
	if err != nil {
		return err
	}

	// Generate UUIDs for logs without IDs
	for i := range logs {
		if logs[i].ID == "" {
			logs[i].ID = uuid.New().String()
		}
		logs[i].TenantID = tenantID
	}

	// Use writer database for create operations
	return r.writerDB.WithContext(ctx).CreateInBatches(logs, 100).Error
}

func (r *AuditLogRepository) GetStats(ctx context.Context, filter domain.AuditLogFilter) (*domain.AuditLogStats, error) {
	if filter.StartTime.IsZero() || filter.EndTime.IsZero() {
		return nil, fmt.Errorf("start time and end time are required")
	}
	if filter.TenantID == "" {
		tenantID, err := utils.GetTenantIDFromContext(ctx)
		if err != nil {
			return nil, err
		}
		filter.TenantID = tenantID
	}

	// Use reader database for read operations
	db, err := getTenantScope(r.readerDB, ctx)
	if err != nil {
		return nil, err
	}

	stats := &domain.AuditLogStats{
		ActionCounts:   make(map[domain.ActionType]int64),
		SeverityCounts: make(map[domain.SeverityLevel]int64),
		ResourceCounts: make(map[string]int64),
	}

	// Calculate time range duration
	duration := filter.EndTime.Sub(filter.StartTime)

	type countResult struct {
		Category string
		Key      string
		Count    int64
	}
	var results []countResult

	// Choose the appropriate source based on time range
	var query string
	if duration <= 24*time.Hour {
		// For last 24 hours, use hourly stats
		query = `
			SELECT category, key, SUM(count) as count FROM (
				SELECT 'action' as category, action as key, count
				FROM audit_logs_hourly_stats
				WHERE tenant_id = ? AND bucket >= ? AND bucket < ?
				UNION ALL
				SELECT 'severity', severity, count
				FROM audit_logs_hourly_stats
				WHERE tenant_id = ? AND bucket >= ? AND bucket < ?
				UNION ALL
				SELECT 'resource_type', resource_type, count
				FROM audit_logs_hourly_stats
				WHERE tenant_id = ? AND bucket >= ? AND bucket < ?
				AND resource_type != ''
			) t GROUP BY category, key`
		if err := db.Raw(query,
			filter.TenantID, filter.StartTime, filter.EndTime,
			filter.TenantID, filter.StartTime, filter.EndTime,
			filter.TenantID, filter.StartTime, filter.EndTime).
			Scan(&results).Error; err != nil {
			return nil, fmt.Errorf("failed to get hourly stats: %w", err)
		}
	} else {
		// For longer ranges, use the base table with optimized indexes
		query = `
			WITH time_filtered_logs AS (
				SELECT * FROM audit_logs 
				WHERE tenant_id = ? 
				AND timestamp >= ? 
				AND timestamp < ?
			)
			(
				SELECT 'severity' as category, severity as key, COUNT(*) as count 
				FROM time_filtered_logs 
				GROUP BY severity
			)
			UNION ALL
			(
				SELECT 'action' as category, action as key, COUNT(*) as count 
				FROM time_filtered_logs 
				GROUP BY action
			)
			UNION ALL
			(
				SELECT 'resource_type' as category, resource_type as key, COUNT(*) as count 
				FROM time_filtered_logs 
				WHERE resource_type != ''
				GROUP BY resource_type
			)`
		if err := db.Raw(query, filter.TenantID, filter.StartTime, filter.EndTime).
			Scan(&results).Error; err != nil {
			return nil, fmt.Errorf("failed to get counts: %w", err)
		}
	}

	// Process results into appropriate maps
	for _, r := range results {
		switch r.Category {
		case "severity":
			stats.SeverityCounts[domain.SeverityLevel(r.Key)] = r.Count
		case "action":
			stats.ActionCounts[domain.ActionType(r.Key)] = r.Count
		case "resource_type":
			stats.ResourceCounts[r.Key] = r.Count
		}
	}

	// Get total count using the same strategy
	if duration <= 24*time.Hour {
		if err := db.Raw(`
			SELECT COUNT(*) FROM audit_logs_hourly_stats
			WHERE tenant_id = ? AND bucket >= ? AND bucket < ?`,
			filter.TenantID, filter.StartTime, filter.EndTime).
			Count(&stats.TotalLogs).Error; err != nil {
			return nil, fmt.Errorf("failed to get total count: %w", err)
		}
	} else {
		if err := db.Raw(`
			SELECT COUNT(*) FROM audit_logs
			WHERE tenant_id = ? AND timestamp >= ? AND timestamp < ?`,
			filter.TenantID, filter.StartTime, filter.EndTime).
			Count(&stats.TotalLogs).Error; err != nil {
			return nil, fmt.Errorf("failed to get total count: %w", err)
		}
	}

	return stats, nil
}

func (r *AuditLogRepository) GetRecentLogs(ctx context.Context, tenantID string, since time.Time) ([]domain.AuditLog, error) {
	var logs []domain.AuditLog

	// Use reader database for read operations
	err := r.readerDB.WithContext(ctx).
		Where("tenant_id = ? AND timestamp >= ?", tenantID, since).
		Order("timestamp DESC").
		Limit(100). // Limit to prevent too many logs from being sent
		Find(&logs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get recent logs: %w", err)
	}

	return logs, nil
}
