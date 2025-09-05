package dto

import (
	"encoding/json"
	"time"
)

type CreateTenantRequest struct {
	Name string `json:"name" binding:"required"`
}

type CreateAuditLogRequest struct {
	TenantID     string          `json:"tenant_id" binding:"required" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID       string          `json:"user_id" example:"123456"`
	SessionID    string          `json:"session_id" example:"sess_123456"`
	IPAddress    string          `json:"ip_address" example:"192.168.1.1"`
	UserAgent    string          `json:"user_agent" example:"Mozilla/5.0"`
	Action       string          `json:"action" binding:"required" example:"CREATE"`
	ResourceType string          `json:"resource_type" binding:"required" example:"user"`
	ResourceID   string          `json:"resource_id" binding:"required" example:"user123"`
	Severity     string          `json:"severity" binding:"required" example:"INFO"`
	Message      string          `json:"message" binding:"required" example:"User created successfully"`
	BeforeState  json.RawMessage `json:"before_state" swaggertype:"string" example:"{\\"name\\":\\"old name\\"}"`
	AfterState   json.RawMessage `json:"after_state" swaggertype:"string" example:"{\\"name\\":\\"new name\\"}"`
	Metadata     json.RawMessage `json:"metadata" swaggertype:"string" example:"{\\"key\\":\\"value\\"}"`
	Timestamp    time.Time       `json:"timestamp" binding:"required" example:"2025-07-17T21:20:48Z"`
}
