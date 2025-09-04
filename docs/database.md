# Database Design & Schema

This document describes the **enhanced database schema and design decisions** for the Audit Log API project.  
The database is implemented using **PostgreSQL with TimescaleDB extension**, optimized for **multi-tenant, time-series audit log data** with **configurable retention policies**.

## Key Enhancements

- **Retention Policy System**: Configurable data lifecycle management
- **Performance Optimization**: Enhanced indexing for 1000+ req/s throughput  
- **Multi-tenant Isolation**: Complete data separation with tenant-specific policies
- **TimescaleDB Features**: Hypertables, compression, and continuous aggregates

---

## Core Schema Overview

### `tenants` table
Manages tenant metadata for multi-tenant support.

| Column        | Type         | Description                         |
|----------------|--------------|-------------------------------------|
| `id`           | UUID         | Primary key, auto-generated         |
| `name`         | TEXT         | Tenant name                         |
| `rate_limit`   | INTEGER      | Requests per second allowed         |
| `created_at`   | TIMESTAMPTZ  | Row creation timestamp              |
| `updated_at`   | TIMESTAMPTZ  | Row update timestamp                |

---

### `audit_logs` table
Stores audit log entries, optimized for time-series workloads using TimescaleDB hypertable.

| Column          | Type         | Description                          |
|------------------|--------------|--------------------------------------|
| `id`            | UUID         | Primary key (with `timestamp`)      |
| `tenant_id`     | UUID         | References `tenants(id)`            |
| `message`       | TEXT         | Human-readable log message          |
| `user_id`       | TEXT         | ID of the user performing action    |
| `session_id`    | TEXT         | Session identifier                  |
| `ip_address`    | TEXT         | IP address of the client            |
| `user_agent`    | TEXT         | User agent string                   |
| `action`        | TEXT         | Action type (CREATE, UPDATE, etc.) |
| `resource_type` | TEXT         | Resource type affected              |
| `resource_id`   | TEXT         | Resource ID affected                |
| `severity`      | TEXT         | Severity level (INFO, ERROR, etc.) |
| `before_state`  | JSONB        | State of resource before change     |
| `after_state`   | JSONB        | State of resource after change      |
| `metadata`      | JSONB        | Additional structured metadata      |
| `timestamp`     | TIMESTAMPTZ  | Logical event timestamp             |
| `created_at`    | TIMESTAMPTZ  | Row creation timestamp              |
| `updated_at`    | TIMESTAMPTZ  | Row update timestamp                |

Primary Key: (`id`, `timestamp`) — supports time-series optimization and uniqueness.

---

## Multi-Tenancy

- Every `audit_logs` row is associated with a `tenant_id`, ensuring tenant isolation.  
- Foreign key with `ON DELETE CASCADE` ensures cleanup when a tenant is removed.

---

## Indexing Strategy

To optimize query performance:
- BRIN index on `timestamp` for efficient range queries.
- Partial index on `resource_type` where not null.
- Aggregation-friendly indexes on (`tenant_id`, `timestamp`, `action`), (`tenant_id`, `timestamp`, `severity`).

---

## TimescaleDB Features

### Hypertable
- `audit_logs` is converted into a **hypertable**, partitioned by `timestamp` in **1-day chunks**.

### Compression
- Chunks older than **7 days** are automatically compressed.
- Segments by `tenant_id, action, severity, resource_type` for efficient storage and decompression.

---

## Continuous Aggregates

### `audit_logs_hourly_stats`
A materialized view using TimescaleDB continuous aggregates:
- Aggregates log counts hourly per tenant, action, severity, and resource type.
- Covers up to 1 month of data.
- Automatically refreshed every hour.

This enables fast dashboard queries without scanning raw logs.

---

## Enhanced Schema: Retention Policy System

### `retention_policies` table
Manages configurable data retention rules per tenant.

| Column        | Type         | Description                              |
|---------------|--------------|------------------------------------------|
| `id`          | UUID         | Primary key, auto-generated             |
| `tenant_id`   | UUID         | References `tenants(id)` ON DELETE CASCADE |
| `name`        | TEXT         | Policy name (unique per tenant)         |
| `description` | TEXT         | Human-readable policy description        |
| `rules`       | JSONB        | Array of retention rules (see below)    |
| `enabled`     | BOOLEAN      | Whether policy is active                 |
| `created_at`  | TIMESTAMPTZ  | Row creation timestamp                   |
| `updated_at`  | TIMESTAMPTZ  | Row update timestamp (auto-updated)     |

**Indexes:**
- `idx_retention_policies_tenant_enabled` ON `(tenant_id, enabled)`
- `idx_retention_policies_tenant_name` ON `(tenant_id, name)`

### `retention_jobs` table
Tracks execution of retention policy jobs.

| Column             | Type         | Description                           |
|--------------------|--------------|---------------------------------------|
| `id`               | UUID         | Primary key, auto-generated          |
| `tenant_id`        | UUID         | References `tenants(id)`             |
| `policy_id`        | UUID         | References `retention_policies(id)`  |
| `status`           | TEXT         | Job status (pending/running/completed/failed/cancelled) |
| `start_time`       | TIMESTAMPTZ  | Job start time                        |
| `end_time`         | TIMESTAMPTZ  | Job completion time                   |
| `processed_records`| BIGINT       | Number of records processed           |
| `archived_records` | BIGINT       | Number of records archived            |
| `deleted_records`  | BIGINT       | Number of records deleted             |
| `error_message`    | TEXT         | Error details if job failed           |
| `metadata`         | JSONB        | Additional job metadata               |
| `created_at`       | TIMESTAMPTZ  | Row creation timestamp                |
| `updated_at`       | TIMESTAMPTZ  | Row update timestamp (auto-updated)  |

**Indexes:**
- `idx_retention_jobs_tenant_status` ON `(tenant_id, status)`
- `idx_retention_jobs_policy_status` ON `(policy_id, status)`
- `idx_retention_jobs_status_created` ON `(status, created_at)`

---

## Retention Policy Rules Structure

The `rules` JSONB field contains an array of retention rules:

```json
{
  "conditions": {
    "older_than": "90 days",
    "severities": ["INFO", "WARNING"],
    "actions": ["VIEW", "CREATE"],
    "resource_types": ["user", "order"],
    "max_records": 1000000
  },
  "actions": {
    "archive": true,
    "delete": true,
    "compress": true,
    "notify_on_completion": false,
    "archive_metadata": {
      "retention_reason": "policy_compliance",
      "compliance_type": "financial"
    }
  },
  "priority": 1,
  "name": "Archive old INFO logs"
}
```

### Default Retention Policies

The system provides three default policy templates:

1. **Standard 90-Day Retention**
   - Archive INFO logs after 90 days
   - Archive WARNING logs after 180 days  
   - Keep ERROR/CRITICAL logs for 1 year

2. **Compliance 7-Year Retention**
   - Archive all logs after 30 days
   - Delete after 7 years
   - Enhanced metadata for compliance

3. **High-Volume Data Management**
   - Size-based retention (keep only 1M most recent records)
   - Optimized for high-traffic resources

---

## Performance Optimizations

### Enhanced Indexing Strategy

**Primary Indexes:**
- BRIN index on `timestamp` for efficient time-range queries
- Composite indexes for tenant-scoped queries
- Partial indexes for non-null resource types

**Specialized Indexes for Analytics:**
```sql
-- Optimized for statistics queries
CREATE INDEX idx_audit_logs_stats_action 
ON audit_logs(tenant_id, timestamp, action) INCLUDE (id);

CREATE INDEX idx_audit_logs_stats_severity 
ON audit_logs(tenant_id, timestamp, severity) INCLUDE (id);

CREATE INDEX idx_audit_logs_stats_resource 
ON audit_logs(tenant_id, timestamp, resource_type) INCLUDE (id) 
WHERE resource_type IS NOT NULL;
```

### TimescaleDB Optimizations

**Hypertable Configuration:**
- 1-day chunk intervals for optimal query performance
- Automatic compression after 7 days
- Compression segments by `(tenant_id, action, severity, resource_type)`

**Query Performance:**
- Sub-100ms response times for typical searches
- Efficient aggregation queries with continuous aggregates
- Optimized for 1000+ insertions per second

---

## Data Lifecycle Management

### Automated Processes

1. **Retention Policy Evaluation**: Daily job checks policies against data
2. **Archival Process**: Background workers move old data to S3
3. **Cleanup Process**: Remove archived data from primary storage  
4. **Compression**: TimescaleDB automatically compresses old chunks

### Monitoring & Observability

- Retention job status tracking
- Performance metrics for archival/cleanup operations
- Storage usage monitoring per tenant
- Policy effectiveness analytics

---

## Migration Strategy

Database migrations are located in `scripts/migrations/`:

- `001_init.sql` - Core tables and TimescaleDB setup
- `002_seed_data.sql` - Initial tenant and user data
- `003_retention_policies.sql` - Retention policy system

**Migration Command:**
```bash
sql-migrate up -config=configs/dbconfig.yml
```
