package config

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/opensearch-project/opensearch-go/v2"
)

type OpenSearchConfig struct {
	Host     string
	Port     string
	Username string
	Password string
}

func DefaultOpenSearchConfig() *OpenSearchConfig {
	return &OpenSearchConfig{
		Host:     getEnvOrDefault("OPENSEARCH_HOST", "localhost"),
		Port:     getEnvOrDefault("OPENSEARCH_PORT", "9200"),
		Username: getEnvOrDefault("OPENSEARCH_USERNAME", ""),
		Password: getEnvOrDefault("OPENSEARCH_PASSWORD", ""),
	}
}

func (c *OpenSearchConfig) GetClient() (*opensearch.Client, error) {
	config := opensearch.Config{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		Addresses: []string{
			fmt.Sprintf("http://%s:%s", c.Host, c.Port),
		},
	}

	if c.Username != "" && c.Password != "" {
		config.Username = c.Username
		config.Password = c.Password
	}

	return opensearch.NewClient(config)
}

// GetIndexName returns the index name for a given tenant and time
// Format: audit_logs_<tenant_id>_YYYY_MM_DD
func (c *OpenSearchConfig) GetIndexName(tenantID string, t time.Time) string {
	return fmt.Sprintf("audit_logs_%s_%s", tenantID, t.Format("2006_01_02"))
}

// GetIndexPattern returns a pattern matching all indices for a tenant
// Format: audit_logs_<tenant_id>_*
func (c *OpenSearchConfig) GetIndexPattern(tenantID string) string {
	return fmt.Sprintf("audit_logs_%s_*", tenantID)
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
