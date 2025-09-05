package domain

import (
	"encoding/json"
	"time"
)

type SeverityLevel string

const (
	SeverityInfo     SeverityLevel = "INFO"
	SeverityWarning  SeverityLevel = "WARNING"
	SeverityError    SeverityLevel = "ERROR"
	SeverityCritical SeverityLevel = "CRITICAL"
)

type ActionType string

const (
	ActionCreate ActionType = "CREATE"
	ActionUpdate ActionType = "UPDATE"
	ActionDelete ActionType = "DELETE"
	ActionView   ActionType = "VIEW"
)

type AuditLog struct {
	ID           string          `gorm:"primaryKey;type:uuid" json:"id"`
	TenantID     string          `gorm:"type:uuid;not null" json:"tenant_id"`
	UserID       string          `gorm:"type:uuid" json:"user_id"`
	SessionID    string          `gorm:"type:text" json:"session_id"`
	IPAddress    string          `gorm:"type:text" json:"ip_address"`
	UserAgent    string          `gorm:"type:text" json:"user_agent"`
	Action       string          `gorm:"type:text;not null" json:"action"`
	ResourceType string          `gorm:"type:text" json:"resource_type"`
	ResourceID   string          `gorm:"type:text" json:"resource_id"`
	Message      string          `gorm:"type:text" json:"message"`
	Severity     string          `gorm:"type:text;not null;default:'INFO'" json:"severity"`
	BeforeState  json.RawMessage `gorm:"type:jsonb" json:"before_state,omitempty"`
	AfterState   json.RawMessage `gorm:"type:jsonb" json:"after_state,omitempty"`
	Metadata     json.RawMessage `gorm:"type:jsonb" json:"metadata,omitempty"`
	Timestamp    time.Time       `gorm:"type:timestamp with time zone;not null;default:CURRENT_TIMESTAMP" json:"timestamp"`
	CreatedAt    time.Time       `gorm:"type:timestamp with time zone;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt    time.Time       `gorm:"type:timestamp with time zone;default:CURRENT_TIMESTAMP" json:"updated_at"`
	Tenant       *Tenant         `gorm:"foreignKey:TenantID" json:"-"`
	User         *User           `gorm:"foreignKey:UserID" json:"-"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

type AuditLogFilter struct {
	TenantID     string    `json:"tenant_id"`
	UserID       string    `json:"user_id"`
	SessionID    string    `json:"session_id"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	Message      string    `json:"message"`
	Severity     string    `json:"severity"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Page         int       `json:"page"`
	PageSize     int       `json:"page_size"`
	Limit        int       `json:"limit"`
	Offset       int       `json:"offset"`
}

type AuditLogStats struct {
	TotalLogs      int64                   `json:"total_logs"`
	ActionCounts   map[ActionType]int64    `json:"action_counts"`
	SeverityCounts map[SeverityLevel]int64 `json:"severity_counts"`
	ResourceCounts map[string]int64        `json:"resource_counts"`
}
