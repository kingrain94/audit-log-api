-- +migrate Up
-- Enable TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Create tenants table for multi-tenant support
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    rate_limit INTEGER NOT NULL DEFAULT 1000,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create audit_logs table optimized for time-series data
CREATE TABLE IF NOT EXISTS audit_logs (
    -- Primary identification
    id UUID DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    message TEXT,

    -- User information
    user_id TEXT,
    session_id TEXT,
    ip_address TEXT,
    user_agent TEXT,
    
    -- Action details
    action TEXT NOT NULL,
    resource_type TEXT,
    resource_id TEXT,
    severity TEXT NOT NULL DEFAULT 'INFO',
    
    -- State changes
    before_state JSONB,
    after_state JSONB,
    
    -- Additional data
    metadata JSONB,
    
    -- Timestamps
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Primary key using both id and timestamp for TimescaleDB
    CONSTRAINT pk_audit_logs PRIMARY KEY (id, timestamp)
);

-- Create indexes for audit_logs
CREATE INDEX idx_audit_logs_brin_timestamp ON audit_logs USING BRIN (timestamp);

-- Add specialized indexes for aggregation queries
CREATE INDEX idx_audit_logs_stats_action ON audit_logs(tenant_id, timestamp, action) INCLUDE (id);
CREATE INDEX idx_audit_logs_stats_severity ON audit_logs(tenant_id, timestamp, severity) INCLUDE (id);
CREATE INDEX idx_audit_logs_stats_resource ON audit_logs(tenant_id, timestamp, resource_type) INCLUDE (id) WHERE resource_type IS NOT NULL;

-- Convert audit_logs to hypertable
SELECT create_hypertable('audit_logs', 'timestamp', chunk_time_interval => INTERVAL '1 day', if_not_exists => TRUE);

-- Add compression policy for chunks older than 7 days
ALTER TABLE audit_logs SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'tenant_id,action,severity,resource_type'
);

SELECT add_compression_policy('audit_logs', INTERVAL '7 days');

-- Create continuous aggregates for real-time stats
CREATE MATERIALIZED VIEW audit_logs_hourly_stats
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 hour', timestamp) AS bucket,
    tenant_id,
    action,
    severity,
    resource_type,
    COUNT(*) as count
FROM audit_logs
GROUP BY bucket, tenant_id, action, severity, resource_type
WITH NO DATA;

SELECT add_continuous_aggregate_policy('audit_logs_hourly_stats',
    start_offset => INTERVAL '1 month',
    end_offset => INTERVAL '1 hour',
    schedule_interval => INTERVAL '1 hour');


-- +migrate Down

-- Remove continuous aggregate policy
SELECT remove_continuous_aggregate_policy('audit_logs_hourly_stats', if_exists => TRUE);

-- Drop continuous aggregate view
DROP MATERIALIZED VIEW IF EXISTS audit_logs_hourly_stats;

-- Remove compression policy
SELECT remove_compression_policy('audit_logs', if_exists => TRUE);

-- Drop tables in correct order due to foreign key constraints
DROP TABLE IF EXISTS audit_logs CASCADE;
DROP TABLE IF EXISTS tenants CASCADE;

-- Drop TimescaleDB extension
DROP EXTENSION IF EXISTS timescaledb;
