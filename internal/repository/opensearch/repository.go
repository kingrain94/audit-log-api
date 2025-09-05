package opensearch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"

	"github.com/kingrain94/audit-log-api/internal/config"
	"github.com/kingrain94/audit-log-api/internal/domain"
	"github.com/kingrain94/audit-log-api/internal/utils"
)

type Repository interface {
	// Index indexes a single audit log
	Index(ctx context.Context, log *domain.AuditLog) error
	// BulkIndex indexes multiple audit logs
	BulkIndex(ctx context.Context, logs []domain.AuditLog) error
	// Search searches audit logs with the given filter
	Search(ctx context.Context, filter *domain.AuditLogFilter) ([]domain.AuditLog, error)
	// CreateIndex creates an index for a tenant if it doesn't exist
	CreateIndex(ctx context.Context, tenantID string, t time.Time) error
	// DeleteIndex deletes an index for a tenant
	DeleteIndex(ctx context.Context, tenantID string) error
	// Delete deletes a single audit log by ID
	Delete(ctx context.Context, tenantID, logID string) error
}

type repository struct {
	client *opensearch.Client
	config *config.OpenSearchConfig
}

func NewRepository(client *opensearch.Client, config *config.OpenSearchConfig) Repository {
	return &repository{
		client: client,
		config: config,
	}
}

func (r *repository) Index(ctx context.Context, log *domain.AuditLog) error {
	// Use log timestamp for index name, fallback to current time if not set
	indexTime := time.Now()
	if !log.Timestamp.IsZero() {
		indexTime = log.Timestamp
	}
	indexName := r.config.GetIndexName(log.TenantID, indexTime)

	// Ensure index exists
	if err := r.CreateIndex(ctx, log.TenantID, indexTime); err != nil {
		return fmt.Errorf("failed to ensure index exists: %w", err)
	}

	// Convert log to JSON
	data, err := json.Marshal(log)
	if err != nil {
		return fmt.Errorf("failed to marshal log: %w", err)
	}

	// Create index request
	req := opensearchapi.IndexRequest{
		Index:      indexName,
		DocumentID: log.ID,
		Body:       strings.NewReader(string(data)),
	}

	res, err := req.Do(ctx, r.client)
	if err != nil {
		return fmt.Errorf("failed to index document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error indexing document: %s", res.String())
	}

	return nil
}

func (r *repository) BulkIndex(ctx context.Context, logs []domain.AuditLog) error {
	if len(logs) == 0 {
		return nil
	}

	// Group logs by tenant and date
	logGroups := make(map[string][]domain.AuditLog)
	for _, log := range logs {
		indexTime := time.Now()
		if !log.Timestamp.IsZero() {
			indexTime = log.Timestamp
		}
		indexName := r.config.GetIndexName(log.TenantID, indexTime)
		logGroups[indexName] = append(logGroups[indexName], log)
	}

	// Process each group separately
	for indexName, groupLogs := range logGroups {
		if err := r.bulkIndexGroup(ctx, indexName, groupLogs); err != nil {
			return fmt.Errorf("failed to bulk index group for index %s: %w", indexName, err)
		}
	}

	return nil
}

func (r *repository) bulkIndexGroup(ctx context.Context, indexName string, logs []domain.AuditLog) error {
	// Ensure index exists (using first log's tenant and timestamp)
	if len(logs) > 0 {
		indexTime := time.Now()
		if !logs[0].Timestamp.IsZero() {
			indexTime = logs[0].Timestamp
		}
		if err := r.CreateIndex(ctx, logs[0].TenantID, indexTime); err != nil {
			return fmt.Errorf("failed to ensure index exists: %w", err)
		}
	}

	// Build bulk request body
	var bulkBody strings.Builder
	for _, log := range logs {
		action := map[string]any{
			"index": map[string]any{
				"_index": indexName,
				"_id":    log.ID,
			},
		}
		actionLine, err := json.Marshal(action)
		if err != nil {
			return fmt.Errorf("failed to marshal action: %w", err)
		}
		bulkBody.Write(actionLine)
		bulkBody.WriteString("\n")

		// Add document line
		docLine, err := json.Marshal(log)
		if err != nil {
			return fmt.Errorf("failed to marshal document: %w", err)
		}
		bulkBody.Write(docLine)
		bulkBody.WriteString("\n")
	}

	// Send bulk request
	req := opensearchapi.BulkRequest{
		Body: strings.NewReader(bulkBody.String()),
	}

	res, err := req.Do(ctx, r.client)
	if err != nil {
		return fmt.Errorf("failed to execute bulk request: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("bulk request failed: %s", res.String())
	}

	return nil
}

func (r *repository) Search(ctx context.Context, filter *domain.AuditLogFilter) ([]domain.AuditLog, error) {
	// Get tenant ID from context
	tenantID, err := utils.GetTenantIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant ID from context: %w", err)
	}

	// Build search query
	query := r.buildSearchQuery(filter)

	// Convert query to JSON
	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	// Create search request using tenant's index pattern
	req := opensearchapi.SearchRequest{
		Index: []string{r.config.GetIndexPattern(tenantID)},
		Body:  strings.NewReader(string(queryJSON)),
	}

	// Execute search
	res, err := req.Do(ctx, r.client)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		if res.StatusCode == 404 {
			return []domain.AuditLog{}, nil
		}
		return nil, fmt.Errorf("search request failed: %s", res.String())
	}

	// Parse response
	var searchResult struct {
		Hits struct {
			Hits []struct {
				Source domain.AuditLog `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&searchResult); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract logs from response
	var logs []domain.AuditLog
	for _, hit := range searchResult.Hits.Hits {
		logs = append(logs, hit.Source)
	}

	return logs, nil
}

// buildSearchQuery constructs the OpenSearch query based on the filter
func (r *repository) buildSearchQuery(filter *domain.AuditLogFilter) map[string]any {
	must := make([]map[string]any, 0)

	// Add exact match filters (keyword fields)
	exactMatches := map[string]string{
		"user_id":       filter.UserID,
		"action":        filter.Action,
		"resource_type": filter.ResourceType,
		"severity":      filter.Severity,
		"session_id":    filter.SessionID,
	}
	for field, value := range exactMatches {
		if value != "" {
			must = append(must, createTermQuery(field, value))
		}
	}

	// Add full-text search filters (text fields)
	textMatches := map[string]string{
		"user_agent": filter.UserAgent,
		"message":    filter.Message,
	}
	for field, value := range textMatches {
		if value != "" {
			must = append(must, createMatchQuery(field, value))
		}
	}

	// Add IP address filter (special handling for IP type)
	if filter.IPAddress != "" {
		must = append(must, createTermQuery("ip_address", filter.IPAddress))
	}

	// Add time range filter
	if !filter.StartTime.IsZero() || !filter.EndTime.IsZero() {
		must = append(must, createTimeRangeQuery(filter.StartTime, filter.EndTime))
	}

	// Construct the final query
	query := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": must,
			},
		},
	}

	// Add pagination
	if filter.Page > 0 && filter.PageSize > 0 {
		query["from"] = (filter.Page - 1) * filter.PageSize
		query["size"] = filter.PageSize
	}

	// Add sorting (most recent first)
	query["sort"] = []map[string]any{
		{
			"timestamp": map[string]any{
				"order": "desc",
			},
		},
	}

	return query
}

// Helper functions to create specific query types
func createTermQuery(field, value string) map[string]any {
	return map[string]any{
		"term": map[string]any{
			field: value,
		},
	}
}

func createMatchQuery(field, value string) map[string]any {
	return map[string]any{
		"match": map[string]any{
			field: value,
		},
	}
}

func createTimeRangeQuery(startTime, endTime time.Time) map[string]any {
	timeRange := make(map[string]any)
	if !startTime.IsZero() {
		timeRange["gte"] = startTime
	}
	if !endTime.IsZero() {
		timeRange["lte"] = endTime
	}
	return map[string]any{
		"range": map[string]any{
			"timestamp": timeRange,
		},
	}
}

// getIndexMapping returns the mapping for audit log index with optimized settings
func (r *repository) getIndexMapping() string {
	return `{
		"mappings": {
			"properties": {
				"id": { "type": "keyword" },
				"tenant_id": { "type": "keyword" },
				"user_id": { "type": "keyword" },
				"session_id": { "type": "keyword" },
				"action": { "type": "keyword" },
				"resource_type": { "type": "keyword" },
				"resource_id": { "type": "keyword" },
				"message": { "type": "text" },
				"metadata": { 
					"type": "object",
					"dynamic": true
				},
				"before_state": {
					"type": "object",
					"dynamic": true
				},
				"after_state": {
					"type": "object",
					"dynamic": true
				},
				"severity": { "type": "keyword" },
				"timestamp": { "type": "date" },
				"ip_address": { "type": "ip" },
				"user_agent": { "type": "text" }
			}
		},
		"settings": {
			"index": {
				"number_of_shards": 1,
				"number_of_replicas": 1,
				"refresh_interval": "1s",
				"mapping": {
					"total_fields": {
						"limit": 2000
					}
				}
			}
		}
	}`
}

func (r *repository) CreateIndex(ctx context.Context, tenantID string, t time.Time) error {
	indexName := r.config.GetIndexName(tenantID, t)

	// Check if index exists
	exists := opensearchapi.IndicesExistsRequest{
		Index: []string{indexName},
	}
	res, err := exists.Do(ctx, r.client)
	if err != nil {
		return fmt.Errorf("failed to check index existence: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 200 {
		return nil // Index already exists
	}

	// Create index with mapping and settings
	create := opensearchapi.IndicesCreateRequest{
		Index: indexName,
		Body:  strings.NewReader(r.getIndexMapping()),
	}

	res, err = create.Do(ctx, r.client)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error creating index: %s", res.String())
	}

	return nil
}

func (r *repository) DeleteIndex(ctx context.Context, tenantID string) error {
	indexName := r.config.GetIndexName(tenantID, time.Now()) // Assuming current time for deletion

	delete := opensearchapi.IndicesDeleteRequest{
		Index: []string{indexName},
	}

	res, err := delete.Do(ctx, r.client)
	if err != nil {
		return fmt.Errorf("failed to delete index: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error deleting index: %s", res.String())
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, tenantID, logID string) error {
	indexName := r.config.GetIndexName(tenantID, time.Now()) // Assuming current time for deletion

	req := opensearchapi.DeleteRequest{
		Index:      indexName,
		DocumentID: logID,
	}

	res, err := req.Do(ctx, r.client)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error deleting document: %s", res.String())
	}

	return nil
}
