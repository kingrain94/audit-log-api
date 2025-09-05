package dto

import (
	"encoding/json"
	"time"
)

// CreateTenantResponse represents the response after creating a tenant
type CreateTenantResponse struct {
	ID        string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name      string    `json:"name" example:"My Tenant"`
	CreatedAt time.Time `json:"created_at" example:"2025-07-17T21:20:48Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2025-07-17T21:20:48Z"`
}

// AuditLogResponse represents a single audit log entry in the response
type AuditLogResponse struct {
	ID           string          `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	TenantID     string          `json:"tenant_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID       string          `json:"user_id" example:"123456"`
	SessionID    string          `json:"session_id" example:"sess_123456"`
	IPAddress    string          `json:"ip_address" example:"192.168.1.1"`
	UserAgent    string          `json:"user_agent" example:"Mozilla/5.0"`
	Action       string          `json:"action" example:"CREATE"`
	ResourceType string          `json:"resource_type" example:"user"`
	ResourceID   string          `json:"resource_id" example:"user123"`
	Severity     string          `json:"severity" example:"INFO"`
	Message      string          `json:"message" example:"User created successfully"`
	BeforeState  json.RawMessage `json:"before_state,omitempty" swaggertype:"string" example:"{\\"name\\":\\"old name\\"}"`
	AfterState   json.RawMessage `json:"after_state,omitempty" swaggertype:"string" example:"{\\"name\\":\\"new name\\"}"`
	Metadata     json.RawMessage `json:"metadata,omitempty" swaggertype:"string" example:"{\\"key\\":\\"value\\"}"`
	Timestamp    time.Time       `json:"timestamp" example:"2025-07-17T21:20:48Z"`
}

// GetAuditLogStatsResponse represents statistics about audit logs
type GetAuditLogStatsResponse struct {
	TotalLogs      int64            `json:"total_logs" example:"100"`
	ActionCounts   map[string]int64 `json:"action_counts" example:"CREATE:50,UPDATE:30,DELETE:20"`
	SeverityCounts map[string]int64 `json:"severity_counts" example:"INFO:80,WARNING:15,ERROR:5"`
	ResourceCounts map[string]int64 `json:"resource_counts" example:"user:60,order:40"`
}
