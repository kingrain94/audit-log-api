# System Architecture & Flow Diagrams

This document provides comprehensive visual representations of the enhanced audit log system's architecture, request flow, and data lifecycle management.

## Overview

The audit log system is designed to handle high-volume logging (1000+ req/s) with real-time processing, advanced search capabilities, comprehensive security, and automated data lifecycle management. The system has been enhanced with:

- **Multi-layer Security**: Rate limiting, input validation, suspicious pattern detection
- **Performance Optimization**: Benchmarking, load testing, smart caching
- **Enhanced Data Management**: Configurable retention policies, automated archival
- **Production-Ready Architecture**: Following golang-standards project layout

These diagrams illustrate the complete flow from API request to long-term storage with all security and performance enhancements.

---

## 1. System Architecture Overview

This diagram shows the complete system architecture and how all components interact with each other.

```mermaid
graph TB
    subgraph "Client Layer"
        Client[Client Application]
        Browser[Web Browser]
        Mobile[Mobile App]
    end
    
    subgraph "Security & Gateway Layer"
        LB[Load Balancer]
        SuspiciousBlock[Suspicious Pattern Detection]
        InputSanitize[Input Sanitization]
        RequestValidation[Request Size & Content-Type Validation]
        GlobalRateLimit[Global Rate Limiting]
        Auth[JWT Authentication]
        TenantRateLimit[Per-Tenant Rate Limiting]
        RoleCheck[Role-based Access Control]
    end
    
    subgraph "API Layer"
        AuditAPI[Audit Log API<br/>POST /api/v1/logs]
        BulkAPI[Bulk API<br/>POST /api/v1/logs/bulk]
        SearchAPI[Search API<br/>GET /api/v1/logs]
        ExportAPI[Export API<br/>GET /api/v1/logs/export<br/>(JSON/CSV)]
        StreamAPI[WebSocket Stream<br/>WS /api/v1/logs/stream]
        TenantAPI[Tenant Management<br/>POST/GET /api/v1/tenants]
    end
    
    subgraph "Service Layer"
        AuditService[Audit Log Service]
        TenantService[Tenant Service]
        ValidationSvc[Validation Service]
    end
    
    subgraph "Repository Layer"
        CompositeRepo[Composite Repository]
        PostgresRepo[PostgreSQL Repository]
        OpenSearchRepo[OpenSearch Repository]
    end
    
    subgraph "Message Queue"
        SQS[AWS SQS]
        IndexQueue[Index Queue]
    end
    
    subgraph "Background Workers"
        IndexWorker[Index Worker<br/>OpenSearch Indexing]
        ArchiveWorker[Archive Worker<br/>S3 Archival with Retention Policies]
        CleanupWorker[Cleanup Worker<br/>Data Lifecycle Management]
    end
    
    subgraph "Data Storage"
        PostgresW[(PostgreSQL Writer<br/>Primary Database + TimescaleDB)]
        PostgresR[(PostgreSQL Reader<br/>Read Replica)]
        OpenSearch[(OpenSearch<br/>Search & Analytics)]
        Redis[(Redis<br/>Rate Limiting + PubSub + Cache)]
        S3[(AWS S3<br/>Long-term Archive Storage)]
    end
    
    subgraph "Real-time Features"
        WebSocketHub[WebSocket Hub]
        PubSub[Redis PubSub]
        LiveStream[Live Log Stream]
    end
    
    %% Client Flow
    Client --> LB
    Browser --> LB
    Mobile --> LB
    
    %% Security & Authentication Flow
    LB --> SuspiciousBlock
    SuspiciousBlock --> InputSanitize
    InputSanitize --> RequestValidation
    RequestValidation --> GlobalRateLimit
    GlobalRateLimit --> Auth
    Auth --> TenantRateLimit
    TenantRateLimit --> RoleCheck
    
    %% API Routing
    RoleCheck --> AuditAPI
    RoleCheck --> BulkAPI
    RoleCheck --> SearchAPI
    RoleCheck --> ExportAPI
    RoleCheck --> StreamAPI
    RoleCheck --> TenantAPI
    
    %% Service Layer Flow
    AuditAPI --> AuditService
    BulkAPI --> AuditService
    SearchAPI --> AuditService
    ExportAPI --> AuditService
    StreamAPI --> WebSocketHub
    TenantAPI --> TenantService
    
    AuditService --> TenantService
    AuditService --> ValidationSvc
    
    %% Repository Flow
    AuditService --> CompositeRepo
    CompositeRepo --> PostgresRepo
    CompositeRepo --> OpenSearchRepo
    
    %% Database Writes
    PostgresRepo --> PostgresW
    PostgresRepo --> PostgresR
    OpenSearchRepo --> OpenSearch
    
    %% Queue Processing
    AuditService --> SQS
    SQS --> IndexQueue
    
    %% Worker Processing
    IndexQueue --> IndexWorker
    SQS --> ArchiveWorker
    SQS --> CleanupWorker
    
    IndexWorker --> OpenSearch
    ArchiveWorker --> S3
    CleanupWorker --> PostgresW
    
    %% Real-time Streaming
    AuditService --> PubSub
    PubSub --> WebSocketHub
    WebSocketHub --> LiveStream
    WebSocketHub --> Redis
    
    %% Styling
    classDef clientClass fill:#e1f5fe
    classDef apiClass fill:#f3e5f5
    classDef serviceClass fill:#e8f5e8
    classDef storageClass fill:#fff3e0
    classDef queueClass fill:#fce4ec
    classDef workerClass fill:#f1f8e9
    
    class Client,Browser,Mobile clientClass
    class AuditAPI,BulkAPI,SearchAPI,StreamAPI apiClass
    class AuditService,TenantService,ValidationSvc serviceClass
    class PostgresW,PostgresR,OpenSearch,Redis storageClass
    class SQS,IndexQueue queueClass
    class IndexWorker workerClass
```

### Architecture Components

- **Client Layer**: Various client applications (web, mobile, API clients)
- **Security & Gateway Layer**: Multi-layer security with pattern detection, input sanitization, rate limiting, authentication, and authorization
- **API Layer**: RESTful endpoints, WebSocket streaming, and export capabilities (JSON/CSV)
- **Service Layer**: Business logic, tenant management, and validation services
- **Repository Layer**: Data access abstraction with composite pattern
- **Message Queue**: Multi-queue SQS architecture for indexing, archival, and cleanup operations
- **Background Workers**: Specialized workers for OpenSearch indexing, S3 archival with retention policies, and data lifecycle management
- **Data Storage**: PostgreSQL with TimescaleDB, OpenSearch, Redis (rate limiting + PubSub), and AWS S3 for long-term archival
- **Real-time Features**: Live streaming and notifications with WebSocket support

---

## 2. Create Log Flow Sequence

This sequence diagram shows the step-by-step process when creating an audit log entry.

```mermaid
sequenceDiagram
    participant Client as Client App
    participant LB as Load Balancer
    participant Security as Security Middleware
    participant RateLimit as Rate Limiting
    participant Auth as Auth Middleware
    participant API as Audit Log API
    participant Service as Audit Service
    participant Repo as Repository
    participant PG as PostgreSQL
    participant SQS as AWS SQS
    participant Redis as Redis PubSub
    participant WS as WebSocket Hub
    participant Worker as Background Workers
    participant OS as OpenSearch
    
    Note over Client,OS: Enhanced Audit Log Creation Flow with Security
    
    %% Request Flow with Security Layers
    Client->>+LB: POST /api/v1/logs<br/>{audit_log_data}
    LB->>+Security: Forward Request
    Security->>Security: Block Suspicious Patterns<br/>(SQL injection, XSS, etc.)
    Security->>Security: Sanitize Input<br/>(Remove control chars)
    Security->>Security: Validate Request Size<br/>& Content-Type
    Security->>+RateLimit: Security Checks Passed
    RateLimit->>RateLimit: Check Global Rate Limit<br/>(10k req/min per IP)
    RateLimit->>+Auth: Rate Limit OK
    Auth->>Auth: Validate JWT Token
    Auth->>Auth: Check Tenant Access
    Auth->>+RateLimit: JWT Valid
    RateLimit->>RateLimit: Check Tenant Rate Limit<br/>(1k req/min per tenant)
    RateLimit->>+API: All Checks Passed
    
    %% Validation & Processing
    API->>API: Validate Request Schema
    API->>+Service: CreateLog(auditLogData)
    Service->>Service: Generate Log ID
    Service->>Service: Add Metadata<br/>(timestamp, IP, etc.)
    Service->>Service: Validate Tenant Permissions
    
    %% Database Storage
    Service->>+Repo: SaveAuditLog(log)
    Repo->>+PG: INSERT INTO audit_logs
    PG-->>-Repo: Success
    Repo-->>-Service: Log Saved
    
    %% Queue Background Tasks
    Service->>+SQS: Queue Index Task
    SQS-->>-Service: Task Queued
    
    %% Real-time Broadcasting
    Service->>+Redis: Publish Log Event
    Redis->>+WS: Broadcast to Subscribers
    WS->>Client: Real-time Log Update<br/>(WebSocket)
    
    %% Response
    Service-->>-API: Log Created Successfully
    API-->>-RateLimit: 201 Created
    RateLimit-->>-Auth: Response
    Auth-->>-RateLimit: Response  
    RateLimit-->>-Security: Response
    Security-->>-LB: Response
    LB-->>-Client: 201 Created<br/>{log_id, timestamp}<br/>Headers: X-RateLimit-*
    
    Note over Worker,OS: Background Processing
    
    %% Background Workers (Async)
    SQS->>+Worker: Index Task Message
    Worker->>+OS: Index Log for Search
    OS-->>-Worker: Indexed
    Worker-->>-SQS: Task Complete
    
    Note over Client,OS: Complete Flow - Log persisted, indexed, and available for real-time streaming
```

### Enhanced Flow Breakdown

1. **Multi-Layer Security**: Suspicious pattern detection, input sanitization, request validation
2. **Rate Limiting**: Global (per-IP) and per-tenant rate limiting with Redis backend
3. **Authentication & Authorization**: JWT validation and role-based access control
4. **Request Processing**: Schema validation and business logic processing
5. **Immediate Storage**: Write to PostgreSQL with TimescaleDB for immediate availability
6. **Async Processing**: Queue background tasks for indexing, archival, and cleanup
7. **Real-time Broadcasting**: Notify connected WebSocket clients with rate limit headers
8. **Background Workers**: Process queued tasks asynchronously with specialized workers

---

## 3. Cleanup Flow Sequence

This sequence diagram shows the cleanup process triggered by a user API request to remove old audit logs.

```mermaid
sequenceDiagram
    participant Client as Client App
    participant LB as Load Balancer
    participant Auth as Auth Middleware
    participant API as Cleanup API
    participant Service as Audit Service
    participant Repo as Repository
    participant PG as PostgreSQL
    participant OS as OpenSearch
    participant S3 as AWS S3
    participant SQS as AWS SQS
    participant Worker as Cleanup Worker
    
    Note over Client,Worker: User-Initiated Cleanup Process
    
    %% API Request
    Client->>+LB: DELETE /api/v1/logs/cleanup<br/>{before_date: "2024-01-01"}
    LB->>+Auth: Forward Request
    Auth->>Auth: Validate JWT Token
    Auth->>Auth: Check User Role<br/>(requires "auditor" role)
    Auth->>+API: Authorized Request
    
    %% Validation & Processing
    API->>API: Validate Request Parameters<br/>(before_date, tenant_id)
    API->>+Service: InitiateCleanup(beforeDate, tenantId)
    Service->>Service: Validate Cleanup Parameters
    Service->>Service: Check Tenant Permissions
    
    %% Queue Cleanup Task
    Service->>+SQS: Queue Cleanup Task<br/>{<br/>  tenant_id: "tenant-123",<br/>  before_date: "2024-01-01",<br/>  user_id: "user-456"<br/>}
    SQS-->>-Service: Task Queued
    
    %% Immediate Response
    Service-->>-API: Cleanup Initiated<br/>(task_id: "cleanup-789")
    API-->>-Auth: 202 Accepted
    Auth-->>-LB: Response
    LB-->>-Client: 202 Accepted<br/>{<br/>  "message": "Cleanup initiated",<br/>  "task_id": "cleanup-789"<br/>}
    
    Note over SQS,Worker: Background Archive & Cleanup Processing
    
    %% Worker Processing
    SQS->>+Worker: Cleanup Task Message
    Worker->>Worker: Parse Cleanup Parameters
    Worker->>Worker: Validate Tenant Access
    
    %% Count Records to Archive/Delete
    Worker->>+Repo: CountLogsBeforeDate(beforeDate, tenantId)
    Repo->>+PG: SELECT COUNT(*) FROM audit_logs<br/>WHERE created_at < ? AND tenant_id = ?
    PG-->>-Repo: Record Count
    Repo-->>-Worker: Total Records: 5000
    
    %% Archive to S3 (Before Cleanup)
    Note over Worker,S3: Step 1: Archive Logs to S3
    loop Archive Processing (1000 records per batch)
        Worker->>+Repo: GetLogsBatch(beforeDate, tenantId, offset, 1000)
        Repo->>+PG: SELECT * FROM audit_logs<br/>WHERE created_at < ?<br/>AND tenant_id = ?<br/>ORDER BY created_at<br/>LIMIT 1000 OFFSET ?
        PG-->>-Repo: Log Records Batch
        Repo-->>-Worker: Logs Data (1000 records)
        
        Worker->>Worker: Format Logs as JSON<br/>Compress Data
        Worker->>+S3: PUT /audit-log-archives/<br/>tenant-123/audit_logs_before_2024-01-01_batch_1.json.gz<br/>{compressed_logs_data}
        S3-->>-Worker: Archive Success
        
        alt More Records to Archive
            Worker->>Worker: Wait 200ms<br/>(Rate Limiting)
        else All Records Archived
            Worker->>Worker: Archive Complete (5000 records)
        end
    end
    
    %% Cleanup PostgreSQL (Batch Processing)  
    Note over Worker,PG: Step 2: PostgreSQL Cleanup - Batch Processing
    loop Batch Processing (1000 records per batch)
        Worker->>+Repo: DeleteLogsBatch(beforeDate, tenantId, 1000)
        Repo->>+PG: BEGIN TRANSACTION
        Repo->>+PG: DELETE FROM audit_logs<br/>WHERE created_at < ?<br/>AND tenant_id = ?<br/>LIMIT 1000
        PG-->>-Repo: Deleted Count: 1000
        Repo->>+PG: COMMIT TRANSACTION
        PG-->>-Repo: Success
        Repo-->>-Worker: Batch Complete (1000 deleted)
        
        alt More Records Exist
            Worker->>Worker: Wait 500ms<br/>(Rate Limiting)
        else All Records Processed
            Worker->>Worker: PostgreSQL Cleanup Complete
        end
    end
    
    %% Cleanup OpenSearch
    Note over Worker,OS: Step 3: OpenSearch Cleanup
    Worker->>+OS: DELETE /audit-logs/_query<br/>{<br/>  "query": {<br/>    "bool": {<br/>      "must": [<br/>        {"range": {"timestamp": {"lt": "before_date"}}},<br/>        {"term": {"tenant_id": "tenant-123"}}<br/>      ]<br/>    }<br/>  }<br/>}
    OS-->>-Worker: Deleted Documents: 5000
    
    %% Final Summary
    Worker->>Worker: Generate Archive & Cleanup Summary<br/>(Archived: 5000, PostgreSQL: 5000, OpenSearch: 5000)
    Worker-->>-SQS: Archive & Cleanup Task Complete<br/>{<br/>  "status": "success",<br/>  "archived_count": 5000,<br/>  "deleted_count": 5000,<br/>  "duration": "2m15s"<br/>}
    
    Note over Client,Worker: Archive & Cleanup Complete - Logs safely archived to S3, then removed from active systems
```

### Archive & Cleanup Process Breakdown

1. **API Request**: User makes DELETE request to `/api/v1/logs/cleanup` with date parameter
2. **Authentication**: Validates JWT token and checks for "auditor" role permission
3. **Validation**: Validates cleanup parameters and tenant permissions
4. **Task Queuing**: Queues archive & cleanup task in SQS for background processing
5. **Immediate Response**: Returns 202 Accepted with task ID for status tracking
6. **Archive Phase**: Worker archives logs to S3 in compressed JSON format
7. **Cleanup Phase**: Removes archived records from PostgreSQL and OpenSearch
8. **Status Tracking**: Task completion status with archive and deletion counts
