# Audit Log API

## Overview

A comprehensive audit logging API system designed to track and manage user actions across different applications with multi-tenant support. This system handles high-volume logging, provides advanced search and filtering capabilities, and ensures data integrity and security.

The Audit Log API provides:
- **High-Performance Logging**: Handle 1000+ log entries per second with sub-100ms response times
- **Multi-Tenant Architecture**: Complete data isolation between tenants with per-tenant rate limiting
- **Real-Time Streaming**: WebSocket-based live log monitoring
- **Advanced Search**: Full-text search and filtering capabilities via OpenSearch
- **Data Lifecycle Management**: Automated archival, cleanup, and configurable retention policies
- **Enterprise Security**: JWT authentication, role-based access control, input validation, and rate limiting
- **Export Capabilities**: JSON and CSV export with comprehensive field coverage
- **Performance Testing**: Built-in load testing and benchmarking tools

## Prerequisites

Before running the application locally, ensure you have:

- **Go 1.21+** installed
- **Docker** and **Docker Compose** installed
- **Task** utility (for running Taskfile commands) - [Install here](https://taskfile.dev/)

## Tech Stack

### Core Technologies
- **Language**: Go 1.23+
- **Web Framework**: Gin (HTTP router and middleware)
- **API Documentation**: Swagger/OpenAPI 3.0

### Data Storage
- **Primary Database**: PostgreSQL 15+ with TimescaleDB extension (optimized for time-series data)
- **Search Engine**: OpenSearch (advanced search and full-text search)
- **Cache & PubSub**: Redis (rate limiting, real-time messaging, caching)
- **Archive Storage**: AWS S3 (long-term log storage with configurable retention policies)

### Queue & Workers
- **Queue System**: AWS SQS (background task processing)
- **Worker Services**: 
  - Index Worker (OpenSearch indexing)
  - Archive Worker (S3 archival with retention policies)
  - Cleanup Worker (data retention and lifecycle management)

### Infrastructure & Security
- **Containerization**: Docker & Docker Compose
- **Local Development**: LocalStack (AWS services simulation)
- **Database Migration**: sql-migrate
- **Configuration**: Environment-based configuration
- **Security Middleware**: Input validation, rate limiting, SQL injection protection
- **Performance Testing**: Built-in benchmarks and load testing tools

## Installation & Setup

### 1. Clone the Repository

```bash
git clone <repository-url>
cd audit-log-api
```

### 2. Start Infrastructure Services

Start all required services using Docker Compose:

```bash
# Start PostgreSQL, OpenSearch, Redis, and LocalStack
task docker-up

# Wait for services to be ready
```

### 3. Initialize AWS Resources

Set up SQS queues and S3 buckets in LocalStack:

```bash
# Initialize SQS queues and S3 buckets
task init-localstack
```

### 5. Database Setup

Run database migrations:

```bash
# Create database schema and seed initial data
task migrate-up
```

### 6. Build the Application

```bash
# Build all services
task build-all
```

## Running the Application

### Start All Services

```bash
# Start the main API server
task run-api

# In separate terminals, start the workers:
task run-index-worker    # OpenSearch indexing
task run-archive-worker  # S3 archival
task run-cleanup-worker  # Data cleanup
```

### Verify Installation

1. **API Health Check**:
   ```bash
   curl http://localhost:10000/health
   ```

2. **API Documentation**:
   Open http://localhost:10000/swagger/index.html in your browser

3. **Generate Test Token**:
   ```bash
   task generate-token
   ```

4. **Test API Endpoints**:
   Import this [Postman collection](test/data/AuditLogAPI.postman_collection.json) for testing

## Performance Testing

The API includes comprehensive performance testing capabilities to ensure it meets the 1000+ requests/second requirement:

### Benchmarks

```bash
# Run performance benchmarks
task test-performance
```

This will run Go benchmarks that test:
- Create audit log performance
- List audit logs performance  
- High concurrency scenarios
- Memory usage under sustained load

### Load Testing

```bash
# Run comprehensive load testing
task load-test
```

The load testing script provides:
- Basic load test (1000 requests)
- High load test (5000 requests) 
- Stress test (10000 requests)
- Search performance validation
- Throughput and latency metrics
- Requirements compliance checking

### Performance Metrics

The system is designed to meet these performance targets:
- **Throughput**: 1000+ requests per second
- **Latency**: Sub-100ms response times for search queries
- **Concurrency**: Handle 200+ concurrent connections
- **Rate Limiting**: Per-tenant (1000 req/min) and global (10k req/min per IP)

## Security Features

### Multi-Layer Security Architecture

The API implements comprehensive security measures:

```
Request → Suspicious Pattern Detection → Input Sanitization → 
Request Size Validation → Content-Type Validation → 
Global Rate Limiting → JWT Auth → Tenant Rate Limiting → 
Role-based Access → Handler
```

### Security Middleware

1. **Input Validation & Sanitization**
   - SQL injection protection
   - XSS prevention
   - Path traversal protection
   - Control character filtering

2. **Rate Limiting**
   - Per-tenant rate limiting (configurable, default: 1000 req/min)
   - Global IP-based rate limiting (default: 10k req/min)
   - Redis-backed with proper headers
   - Fail-open strategy for resilience

3. **Request Validation**
   - Content-Type validation
   - Request size limits (default: 10MB)
   - Suspicious pattern detection
   - Malicious payload blocking

4. **Authentication & Authorization**
   - JWT-based authentication
   - Role-based access control (Admin, User, Auditor)
   - Multi-tenant isolation
   - Session management

## Configuration

### Environment Variables

Key configuration options can be set via environment variables:

```bash
# Server Configuration
SERVER_PORT=10000                    # API server port
APP_ENV=development                  # Environment (development/production)

# JWT Configuration  
JWT_SECRET_KEY=your-secret-key       # JWT signing secret
JWT_EXPIRATION_HOURS=24             # Token expiration time

# Rate Limiting
DEFAULT_RATE_LIMIT=1000             # Default per-tenant rate limit (req/min)
GLOBAL_RATE_LIMIT=10000             # Global rate limit per IP (req/min)

# Database URLs
DATABASE_WRITER_URL=postgres://...   # Primary database connection
DATABASE_READER_URL=postgres://...   # Read replica connection

# Redis Configuration
REDIS_URL=redis://localhost:6379    # Redis connection string

# AWS Configuration (LocalStack for development)
AWS_ENDPOINT=http://localhost:4566  # LocalStack endpoint
AWS_REGION=us-east-1                # AWS region
S3_BUCKET=audit-logs                # S3 bucket for archives
SQS_QUEUE_URL=http://localhost:4566/... # SQS queue URL
```

### Security Best Practices

For production deployment:

1. **Use strong JWT secrets** (256-bit random keys)
2. **Enable HTTPS** for all API endpoints
3. **Configure proper CORS** settings
4. **Set appropriate rate limits** based on your traffic patterns
5. **Use database connection pooling** for optimal performance
6. **Enable request logging** for security monitoring
7. **Regularly rotate JWT secrets** and API keys

## Available Tasks

You can see all available tasks by running:

```bash
task --list
```

### Convenience Tasks

The Taskfile includes several convenience tasks for common development workflows:

```bash
# Complete project setup (dependencies + docker + migrations)
task setup

# Start development environment (docker + API server)
task dev

# Build everything and run tests
task all

# Performance testing
task test-performance      # Run benchmarks
task load-test            # Run load testing script
```

### Key Features of Task vs Make

- **Smart Caching**: Task automatically detects when files haven't changed and skips unnecessary rebuilds
- **Parallel Execution**: Tasks run in parallel when possible for faster builds
- **Cross-Platform**: Works consistently across Linux, macOS, and Windows
- **Better Dependency Management**: More intuitive task dependencies and execution order

## Project Structure

```
audit-log-api/
├── api/                   # OpenAPI/Swagger specs, protocol definitions
├── build/                 # Packaging and Continuous Integration
│   ├── ci/               # CI configurations
│   ├── docker/           # Docker files
│   └── package/          # OS package configurations
├── cmd/                   # Application entry points
│   ├── api/              # Main API server
│   ├── archive_worker/   # S3 archive worker
│   ├── cleanup_worker/   # Data cleanup worker
│   └── index_worker/     # OpenSearch index worker
├── configs/               # Configuration file templates
├── deployments/           # IaaS, PaaS, system and container orchestration
├── docs/                  # Design and user documents
├── internal/              # Private application and library code
│   ├── api/              # HTTP handlers and routes
│   ├── config/           # Configuration management
│   ├── domain/           # Domain models (audit logs, retention policies)
│   ├── middleware/       # HTTP middleware (auth, rate limiting, validation)
│   ├── repository/       # Data access layer
│   ├── service/          # Business logic
│   └── worker/           # Background workers
├── pkg/                   # Library code that's ok to use by external applications
├── scripts/               # Build, install, analysis scripts
└── test/                  # Additional external test apps and test data
    ├── data/             # Test data files (Postman collections, etc.)
    ├── integration/      # Integration tests and performance benchmarks
    └── e2e/              # End-to-end tests
```

## Documentation

For detailed information about the system, refer to:

- **[docs/architecture.md](docs/architecture.md)** - Enhanced system architecture, security flows, and data lifecycle
- **[docs/database.md](docs/database.md)** - Database design with retention policies and performance optimizations
- **[docs/queue-architecture.md](docs/queue-architecture.md)** - Multi-queue SQS architecture and background processing
- **[api/README.md](api/README.md)** - API specifications and client generation
- **API Documentation**: http://localhost:10000/swagger/index.html (when running)

## Features

### ✅ **Core Functionality**
- **Multi-tenant Architecture** with complete data isolation
- **High-Performance API** (1000+ requests/second validated)
- **Real-time WebSocket Streaming** for live log monitoring
- **Advanced Search** with OpenSearch integration
- **Export Capabilities** (JSON/CSV with all fields)

### ✅ **Security & Performance**
- **Multi-layer Security Middleware** (validation, sanitization, rate limiting)
- **JWT Authentication** with role-based access control (Admin, User, Auditor)
- **Rate Limiting** (per-tenant + global with Redis backend)
- **Input Validation** (SQL injection, XSS, path traversal protection)
- **Performance Testing** (benchmarks + load testing scripts)

### ✅ **Data Management**
- **Configurable Retention Policies** (90-day, compliance, high-volume)
- **Automated Data Lifecycle** (archival, cleanup, retention)
- **TimescaleDB Optimization** for time-series data
- **Database Read/Write Separation** for optimal performance

### ✅ **Infrastructure & DevOps**
- **AWS Integration** (SQS, S3) with LocalStack support
- **Background Workers** for async processing
- **Comprehensive API Documentation** with OpenAPI/Swagger
- **Task-based Build System** with smart caching
- **Docker Containerization** for easy deployment

### ✅ **Quality Assurance**
- **Performance Benchmarks** meeting 1000+ req/s requirement
- **Load Testing Scripts** with compliance validation
- **Comprehensive Test Coverage** with mocks
- **Production-Ready** architecture and error handling
