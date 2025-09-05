package dto

import (
	"github.com/kingrain94/audit-log-api/internal/domain"
)

// ToAuditLog converts a CreateAuditLogRequest DTO to an AuditLog domain model
func (r *CreateAuditLogRequest) ToAuditLog() *domain.AuditLog {
	return &domain.AuditLog{
		TenantID:     r.TenantID,
		UserID:       r.UserID,
		SessionID:    r.SessionID,
		IPAddress:    r.IPAddress,
		UserAgent:    r.UserAgent,
		Action:       r.Action,
		ResourceType: r.ResourceType,
		ResourceID:   r.ResourceID,
		Severity:     r.Severity,
		Message:      r.Message,
		BeforeState:  r.BeforeState,
		AfterState:   r.AfterState,
		Metadata:     r.Metadata,
		Timestamp:    r.Timestamp,
	}
}

// FromAuditLog converts an AuditLog domain model to an AuditLogResponse DTO
func FromAuditLog(log *domain.AuditLog) *AuditLogResponse {
	return &AuditLogResponse{
		ID:           log.ID,
		TenantID:     log.TenantID,
		UserID:       log.UserID,
		SessionID:    log.SessionID,
		IPAddress:    log.IPAddress,
		UserAgent:    log.UserAgent,
		Action:       log.Action,
		ResourceType: log.ResourceType,
		ResourceID:   log.ResourceID,
		Severity:     log.Severity,
		Message:      log.Message,
		BeforeState:  log.BeforeState,
		AfterState:   log.AfterState,
		Metadata:     log.Metadata,
		Timestamp:    log.Timestamp,
	}
}

func FromAuditLogs(logs []domain.AuditLog) []AuditLogResponse {
	responses := make([]AuditLogResponse, len(logs))
	for i, log := range logs {
		responses[i] = *FromAuditLog(&log)
	}
	return responses
}
