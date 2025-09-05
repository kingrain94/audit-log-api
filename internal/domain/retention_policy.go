package domain

import (
	"encoding/json"
	"time"
)

// RetentionPolicy defines data retention rules for audit logs
type RetentionPolicy struct {
	ID          string          `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	TenantID    string          `gorm:"type:uuid;not null" json:"tenant_id"`
	Name        string          `gorm:"type:text;not null" json:"name"`
	Description string          `gorm:"type:text" json:"description"`
	Rules       []RetentionRule `gorm:"type:jsonb" json:"rules"`
	Enabled     bool            `gorm:"not null;default:true" json:"enabled"`
	CreatedAt   time.Time       `gorm:"type:timestamp with time zone;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time       `gorm:"type:timestamp with time zone;default:CURRENT_TIMESTAMP" json:"updated_at"`
	Tenant      *Tenant         `gorm:"foreignKey:TenantID" json:"-"`
}

func (RetentionPolicy) TableName() string {
	return "retention_policies"
}

// RetentionRule defines a specific retention rule
type RetentionRule struct {
	// Conditions for applying this rule
	Conditions RetentionConditions `json:"conditions"`

	// Actions to take when conditions are met
	Actions RetentionActions `json:"actions"`

	// Priority (higher numbers processed first)
	Priority int `json:"priority"`

	// Rule name for identification
	Name string `json:"name"`
}

// RetentionConditions define when a retention rule should be applied
type RetentionConditions struct {
	// Age-based conditions
	OlderThan *time.Duration `json:"older_than,omitempty"` // e.g., "90 days"

	// Severity-based conditions
	Severities []string `json:"severities,omitempty"` // e.g., ["INFO", "WARNING"]

	// Action-based conditions
	Actions []string `json:"actions,omitempty"` // e.g., ["VIEW", "CREATE"]

	// Resource-based conditions
	ResourceTypes []string `json:"resource_types,omitempty"` // e.g., ["user", "order"]

	// Size-based conditions (for large datasets)
	MaxRecords *int64 `json:"max_records,omitempty"` // Keep only the most recent N records
}

// RetentionActions define what to do with matching audit logs
type RetentionActions struct {
	// Archive to S3 before deletion
	Archive bool `json:"archive"`

	// Delete from primary storage
	Delete bool `json:"delete"`

	// Compress before archiving
	Compress bool `json:"compress"`

	// Notification settings
	NotifyOnCompletion bool `json:"notify_on_completion"`

	// Custom metadata for archived data
	ArchiveMetadata map[string]interface{} `json:"archive_metadata,omitempty"`
}

// RetentionPolicyFilter for querying retention policies
type RetentionPolicyFilter struct {
	TenantID string `json:"tenant_id"`
	Enabled  *bool  `json:"enabled"`
	Name     string `json:"name"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Limit    int    `json:"limit"`
	Offset   int    `json:"offset"`
}

// RetentionJob represents a scheduled retention job
type RetentionJob struct {
	ID               string             `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	TenantID         string             `gorm:"type:uuid;not null" json:"tenant_id"`
	PolicyID         string             `gorm:"type:uuid;not null" json:"policy_id"`
	Status           RetentionJobStatus `gorm:"type:text;not null;default:'pending'" json:"status"`
	StartTime        *time.Time         `gorm:"type:timestamp with time zone" json:"start_time,omitempty"`
	EndTime          *time.Time         `gorm:"type:timestamp with time zone" json:"end_time,omitempty"`
	ProcessedRecords int64              `gorm:"not null;default:0" json:"processed_records"`
	ArchivedRecords  int64              `gorm:"not null;default:0" json:"archived_records"`
	DeletedRecords   int64              `gorm:"not null;default:0" json:"deleted_records"`
	ErrorMessage     string             `gorm:"type:text" json:"error_message,omitempty"`
	Metadata         json.RawMessage    `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt        time.Time          `gorm:"type:timestamp with time zone;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt        time.Time          `gorm:"type:timestamp with time zone;default:CURRENT_TIMESTAMP" json:"updated_at"`
	Policy           *RetentionPolicy   `gorm:"foreignKey:PolicyID" json:"policy,omitempty"`
	Tenant           *Tenant            `gorm:"foreignKey:TenantID" json:"-"`
}

func (RetentionJob) TableName() string {
	return "retention_jobs"
}

// RetentionJobStatus represents the status of a retention job
type RetentionJobStatus string

const (
	RetentionJobPending   RetentionJobStatus = "pending"
	RetentionJobRunning   RetentionJobStatus = "running"
	RetentionJobCompleted RetentionJobStatus = "completed"
	RetentionJobFailed    RetentionJobStatus = "failed"
	RetentionJobCancelled RetentionJobStatus = "cancelled"
)

// Default retention policies for different use cases
func GetDefaultRetentionPolicies() []RetentionPolicy {
	return []RetentionPolicy{
		{
			Name:        "Standard 90-Day Retention",
			Description: "Archive logs older than 90 days, keep high-severity logs for 1 year",
			Rules: []RetentionRule{
				{
					Name:     "Archive old INFO logs",
					Priority: 1,
					Conditions: RetentionConditions{
						OlderThan:  durationPtr(90 * 24 * time.Hour), // 90 days
						Severities: []string{"INFO"},
					},
					Actions: RetentionActions{
						Archive:            true,
						Delete:             true,
						Compress:           true,
						NotifyOnCompletion: false,
					},
				},
				{
					Name:     "Archive old WARNING logs",
					Priority: 2,
					Conditions: RetentionConditions{
						OlderThan:  durationPtr(180 * 24 * time.Hour), // 180 days
						Severities: []string{"WARNING"},
					},
					Actions: RetentionActions{
						Archive:            true,
						Delete:             true,
						Compress:           true,
						NotifyOnCompletion: false,
					},
				},
				{
					Name:     "Keep ERROR and CRITICAL logs longer",
					Priority: 3,
					Conditions: RetentionConditions{
						OlderThan:  durationPtr(365 * 24 * time.Hour), // 1 year
						Severities: []string{"ERROR", "CRITICAL"},
					},
					Actions: RetentionActions{
						Archive:            true,
						Delete:             true,
						Compress:           true,
						NotifyOnCompletion: true,
						ArchiveMetadata: map[string]interface{}{
							"retention_reason": "high_severity",
							"compliance":       "required",
						},
					},
				},
			},
			Enabled: true,
		},
		{
			Name:        "Compliance 7-Year Retention",
			Description: "Long-term retention for compliance requirements",
			Rules: []RetentionRule{
				{
					Name:     "Long-term archive for compliance",
					Priority: 1,
					Conditions: RetentionConditions{
						OlderThan: durationPtr(30 * 24 * time.Hour), // 30 days
					},
					Actions: RetentionActions{
						Archive:            true,
						Delete:             false, // Keep in primary storage for 30 days
						Compress:           true,
						NotifyOnCompletion: false,
						ArchiveMetadata: map[string]interface{}{
							"retention_period": "7_years",
							"compliance_type":  "financial",
						},
					},
				},
				{
					Name:     "Delete after 7 years",
					Priority: 2,
					Conditions: RetentionConditions{
						OlderThan: durationPtr(7 * 365 * 24 * time.Hour), // 7 years
					},
					Actions: RetentionActions{
						Archive:            false,
						Delete:             true,
						Compress:           false,
						NotifyOnCompletion: true,
					},
				},
			},
			Enabled: false, // Disabled by default, enable for compliance-required tenants
		},
		{
			Name:        "High-Volume Data Management",
			Description: "Manage high-volume audit logs with size-based retention",
			Rules: []RetentionRule{
				{
					Name:     "Keep only recent records for high-volume resources",
					Priority: 1,
					Conditions: RetentionConditions{
						ResourceTypes: []string{"api_request", "page_view"},
						MaxRecords:    int64Ptr(1000000), // Keep only 1M most recent records
					},
					Actions: RetentionActions{
						Archive:            true,
						Delete:             true,
						Compress:           true,
						NotifyOnCompletion: false,
					},
				},
			},
			Enabled: false, // Enable for high-volume tenants
		},
	}
}

// Helper functions
func durationPtr(d time.Duration) *time.Duration {
	return &d
}

func int64Ptr(i int64) *int64 {
	return &i
}
