package api

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/kingrain94/audit-log-api/internal/api/dto"
	"github.com/kingrain94/audit-log-api/internal/domain"
	contextutils "github.com/kingrain94/audit-log-api/internal/utils"
	"github.com/kingrain94/audit-log-api/pkg/utils"
)

//go:generate mockery --name AuditLogService --output ../mocks
type AuditLogService interface {
	Create(ctx context.Context, req dto.CreateAuditLogRequest) error
	BulkCreate(ctx context.Context, reqs []dto.CreateAuditLogRequest) error
	GetByID(ctx context.Context, id string) (*dto.AuditLogResponse, error)
	List(ctx context.Context, filter *domain.AuditLogFilter, usePagination bool) ([]dto.AuditLogResponse, error)
	GetStats(ctx context.Context, filter *domain.AuditLogFilter) (*dto.GetAuditLogStatsResponse, error)
	GetStatsV2(ctx context.Context, filter *domain.AuditLogFilter) (*dto.GetAuditLogStatsResponse, error)
	ScheduleArchive(ctx context.Context, tenantID string, beforeDate time.Time) error
}

type AuditLogHandler struct {
	*BaseHandler
	service AuditLogService
}

func NewAuditLogHandler(service AuditLogService) *AuditLogHandler {
	return &AuditLogHandler{service: service}
}

// CreateLog Create a new audit log entry
// @Summary Create audit log
// @Description Create a new audit log entry
// @Tags    audit_logs
// @Accept  json
// @Produce json
// @Param   body body dto.CreateAuditLogRequest true "Audit log object"
// @Success 201
// @Failure 400 {object} dto.Error
// @Failure 401 {object} dto.Error
// @Failure 500 {object} dto.Error
// @Router  /logs [post]
func (h *AuditLogHandler) CreateLog(c *gin.Context) {
	var log dto.CreateAuditLogRequest
	if err := c.ShouldBindJSON(&log); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error{Error: err.Error()})
		return
	}

	if err := h.service.Create(h.RequestCtx(c), log); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Log created successfully"})
}

// BulkCreateLogs Create multiple audit log entries
// @Summary Bulk create audit logs
// @Description Create multiple audit log entries in a single request
// @Tags    audit_logs
// @Accept  json
// @Produce json
// @Param   body body []dto.CreateAuditLogRequest true "Array of audit log objects"
// @Success 201
// @Failure 400 {object} dto.Error
// @Failure 401 {object} dto.Error
// @Failure 500 {object} dto.Error
// @Router  /logs/bulk [post]
func (h *AuditLogHandler) BulkCreateLogs(c *gin.Context) {
	var logs []dto.CreateAuditLogRequest
	if err := c.ShouldBindJSON(&logs); err != nil {
		c.JSON(http.StatusBadRequest, dto.Error{Error: err.Error()})
		return
	}

	if err := h.service.BulkCreate(h.RequestCtx(c), logs); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Logs created successfully"})
}

// GetLog Get a specific audit log by ID
// @Summary Get audit log
// @Description Get an audit log entry by its ID
// @Tags    audit_logs
// @Produce json
// @Param   id path string true "Log ID"
// @Success 200 {object} dto.AuditLogResponse
// @Failure 401 {object} dto.Error
// @Failure 404 {object} dto.Error
// @Failure 500 {object} dto.Error
// @Router  /logs/{id} [get]
func (h *AuditLogHandler) GetLog(c *gin.Context) {
	id := c.Param("id")

	log, err := h.service.GetByID(h.RequestCtx(c), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error{Error: err.Error()})
		return
	}
	if log == nil {
		c.JSON(http.StatusNotFound, dto.Error{Error: "Log not found"})
		return
	}

	c.JSON(http.StatusOK, log)
}

// ListLogs Get a list of audit logs with filtering
// @Summary List audit logs
// @Description Get a list of audit logs with filtering options
// @Tags    audit_logs
// @Produce json
// @Param   page query int false "Page number"
// @Param   page_size query int false "Page size"
// @Param   user_id query string false "Filter by user ID"
// @Param   action query string false "Filter by action"
// @Param   resource_type query string false "Filter by resource type"
// @Param   severity query string false "Filter by severity"
// @Param   start_time query string true "Filter by start time (RFC3339 or YYYY-MM-DD)" example:"2024-03-20T00:00:00Z"
// @Param   end_time query string true "Filter by end time (RFC3339 or YYYY-MM-DD)" example:"2024-03-20T23:59:59Z"
// @Success 200 {array} dto.AuditLogResponse
// @Failure 401 {object} dto.Error
// @Failure 500 {object} dto.Error
// @Router  /logs [get]
func (h *AuditLogHandler) ListLogs(c *gin.Context) {
	filter, err := getFilterFromQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error{Error: err.Error()})
		return
	}

	logs, err := h.service.List(h.RequestCtx(c), filter, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, logs)
}

// ExportLogs Export audit logs in JSON or CSV format
// @Summary Export audit logs
// @Description Export audit logs with filtering options in JSON or CSV format
// @Tags    audit_logs
// @Produce json,text/csv
// @Param   format query string false "Export format (json or csv)" default(json)
// @Param   user_id query string false "Filter by user ID"
// @Param   action query string false "Filter by action"
// @Param   resource_type query string false "Filter by resource type"
// @Param   severity query string false "Filter by severity"
// @Param   start_time query string true "Filter by start time (RFC3339 or YYYY-MM-DD)" example:"2024-03-20T00:00:00Z"
// @Param   end_time query string true "Filter by end time (RFC3339 or YYYY-MM-DD)" example:"2024-03-20T23:59:59Z"
// @Success 200 {file} file
// @Failure 400 {object} dto.Error
// @Failure 401 {object} dto.Error
// @Failure 500 {object} dto.Error
// @Router  /logs/export [get]
func (h *AuditLogHandler) ExportLogs(c *gin.Context) {
	format := c.DefaultQuery("format", "json")
	if format != "json" && format != "csv" {
		c.JSON(http.StatusBadRequest, dto.Error{Error: "Invalid format. Must be 'json' or 'csv'"})
		return
	}

	filter, err := getFilterFromQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error{Error: err.Error()})
		return
	}

	logs, err := h.service.List(h.RequestCtx(c), filter, false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error{Error: err.Error()})
		return
	}

	switch format {
	case "json":
		c.Header("Content-Disposition", "attachment; filename=audit_logs.json")
		c.JSON(http.StatusOK, logs)
	case "csv":
		c.Header("Content-Disposition", "attachment; filename=audit_logs.csv")
		c.Header("Content-Type", "text/csv")

		writer := csv.NewWriter(c.Writer)
		defer writer.Flush()

		// Write CSV header
		header := []string{
			"ID", "TenantID", "UserID", "SessionID", "Action",
			"ResourceType", "ResourceID", "IPAddress", "UserAgent",
			"Severity", "Message", "BeforeState", "AfterState",
			"Metadata", "Timestamp",
		}
		if err := writer.Write(header); err != nil {
			c.JSON(http.StatusInternalServerError, dto.Error{Error: "Failed to write CSV header"})
			return
		}

		// Write each log entry as CSV
		for _, log := range logs {
			// Convert JSON fields to strings
			beforeState := ""
			if log.BeforeState != nil {
				beforeState = string(log.BeforeState)
			}
			afterState := ""
			if log.AfterState != nil {
				afterState = string(log.AfterState)
			}
			metadata := ""
			if log.Metadata != nil {
				metadata = string(log.Metadata)
			}

			record := []string{
				log.ID,
				log.TenantID,
				log.UserID,
				log.SessionID,
				log.Action,
				log.ResourceType,
				log.ResourceID,
				log.IPAddress,
				log.UserAgent,
				log.Severity,
				log.Message,
				beforeState,
				afterState,
				metadata,
				log.Timestamp.Format(time.RFC3339),
			}

			if err := writer.Write(record); err != nil {
				c.JSON(http.StatusInternalServerError, dto.Error{Error: "Failed to write CSV record"})
				return
			}
		}
	}
}

// GetStats Get audit log statistics
// @Summary Get log statistics
// @Description Get statistics about audit logs including counts by action, severity, and resource
// @Tags    audit_logs
// @Produce json
// @Param   start_time query string true "Filter by start time (RFC3339 or YYYY-MM-DD)" example:"2024-03-20T00:00:00Z"
// @Param   end_time query string true "Filter by end time (RFC3339 or YYYY-MM-DD)" example:"2024-03-20T23:59:59Z"
// @Success 200 {object} dto.GetAuditLogStatsResponse
// @Failure 401 {object} dto.Error
// @Failure 500 {object} dto.Error
// @Router  /logs/stats [get]
func (h *AuditLogHandler) GetStats(c *gin.Context) {
	filter, err := getFilterFromQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error{Error: err.Error()})
		return
	}

	stats, err := h.service.GetStatsV2(h.RequestCtx(c), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func getFilterFromQuery(c *gin.Context) (*domain.AuditLogFilter, error) {
	tenantID := c.GetString(string(contextutils.TenantIDKey))
	if tenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}

	filter := &domain.AuditLogFilter{
		TenantID:     tenantID,
		UserID:       c.Query("user_id"),
		Action:       c.Query("action"),
		ResourceType: c.Query("resource_type"),
		Severity:     c.Query("severity"),
		SessionID:    c.Query("session_id"),
		IPAddress:    c.Query("ip_address"),
		UserAgent:    c.Query("user_agent"),
		Message:      c.Query("message"),
	}

	// Parse pagination
	if page := c.Query("page"); page != "" {
		if pageNum, err := strconv.Atoi(page); err == nil {
			filter.Page = pageNum
		}
	}
	if pageSize := c.Query("page_size"); pageSize != "" {
		if size, err := strconv.Atoi(pageSize); err == nil {
			filter.PageSize = size
		}
	}

	// Parse time filters
	if startTime := c.Query("start_time"); startTime != "" {
		t, err := utils.ParseUserTime(startTime, false)
		if err != nil {
			return nil, err
		}
		filter.StartTime = t
	} else {
		return nil, fmt.Errorf("start_time is required")
	}
	if endTime := c.Query("end_time"); endTime != "" {
		t, err := utils.ParseUserTime(endTime, true)
		if err != nil {
			return nil, err
		}
		filter.EndTime = t
	} else {
		return nil, fmt.Errorf("end_time is required")
	}
	if filter.StartTime.After(filter.EndTime) {
		return nil, fmt.Errorf("start_time must be before end_time")
	}

	return filter, nil
}

// Cleanup Schedule cleanup operation for audit logs
// @Summary Schedule cleanup operation
// @Description Enqueues an archive job message to SQS for logs before the specified date
// @Tags audit-logs
// @Accept json
// @Produce json
// @Param before_date query string true "Cleanup logs before this date (ISO 8601 or YYYY-MM-DD)"
// @Success 202 {object} map[string]interface{} "Cleanup operation scheduled"
// @Failure 400 {object} dto.Error
// @Failure 401 {object} dto.Error
// @Failure 500 {object} dto.Error
// @Security ApiKeyAuth
// @Router /api/v1/logs/cleanup [delete]
func (h *AuditLogHandler) Cleanup(c *gin.Context) {
	tenantID := c.GetString(string(contextutils.TenantIDKey))
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, dto.Error{Error: "No tenant ID found"})
		return
	}

	// Parse before_date from query parameter
	beforeDateStr := c.Query("before_date")
	if beforeDateStr == "" {
		c.JSON(http.StatusBadRequest, dto.Error{Error: "before_date parameter is required"})
		return
	}

	beforeDate, err := utils.ParseUserTime(beforeDateStr, true)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.Error{Error: "Invalid before_date format: " + err.Error()})
		return
	}

	// Validate that the date is not in the future
	if beforeDate.After(time.Now()) {
		c.JSON(http.StatusBadRequest, dto.Error{Error: "before_date cannot be in the future"})
		return
	}

	// Enqueue archive message to SQS
	if err := h.service.ScheduleArchive(c.Request.Context(), tenantID, beforeDate); err != nil {
		c.JSON(http.StatusInternalServerError, dto.Error{Error: "Failed to schedule cleanup: " + err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":     "Cleanup operation scheduled successfully",
		"tenant_id":   tenantID,
		"before_date": beforeDate.Format(time.RFC3339),
	})
}
