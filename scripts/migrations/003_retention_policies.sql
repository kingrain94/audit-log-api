-- +migrate Up
-- Create retention_policies table
CREATE TABLE IF NOT EXISTS retention_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    rules JSONB NOT NULL DEFAULT '[]',
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Ensure unique policy names per tenant
    UNIQUE(tenant_id, name)
);

-- Create retention_jobs table
CREATE TABLE IF NOT EXISTS retention_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    policy_id UUID NOT NULL REFERENCES retention_policies(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled')),
    start_time TIMESTAMP WITH TIME ZONE,
    end_time TIMESTAMP WITH TIME ZONE,
    processed_records BIGINT NOT NULL DEFAULT 0,
    archived_records BIGINT NOT NULL DEFAULT 0,
    deleted_records BIGINT NOT NULL DEFAULT 0,
    error_message TEXT,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for retention_policies
CREATE INDEX idx_retention_policies_tenant_enabled ON retention_policies(tenant_id, enabled);
CREATE INDEX idx_retention_policies_tenant_name ON retention_policies(tenant_id, name);

-- Create indexes for retention_jobs
CREATE INDEX idx_retention_jobs_tenant_status ON retention_jobs(tenant_id, status);
CREATE INDEX idx_retention_jobs_policy_status ON retention_jobs(policy_id, status);
CREATE INDEX idx_retention_jobs_created_at ON retention_jobs(created_at);
CREATE INDEX idx_retention_jobs_status_created ON retention_jobs(status, created_at);

-- Add trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply triggers
CREATE TRIGGER update_retention_policies_updated_at 
    BEFORE UPDATE ON retention_policies 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_retention_jobs_updated_at 
    BEFORE UPDATE ON retention_jobs 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- +migrate Down
DROP TRIGGER IF EXISTS update_retention_jobs_updated_at ON retention_jobs;
DROP TRIGGER IF EXISTS update_retention_policies_updated_at ON retention_policies;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP INDEX IF EXISTS idx_retention_jobs_status_created;
DROP INDEX IF EXISTS idx_retention_jobs_created_at;
DROP INDEX IF EXISTS idx_retention_jobs_policy_status;
DROP INDEX IF EXISTS idx_retention_jobs_tenant_status;

DROP INDEX IF EXISTS idx_retention_policies_tenant_name;
DROP INDEX IF EXISTS idx_retention_policies_tenant_enabled;

DROP TABLE IF EXISTS retention_jobs;
DROP TABLE IF EXISTS retention_policies;
