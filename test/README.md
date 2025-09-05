# Test Directory

This directory contains additional external test apps and test data for the Audit Log API.

## Directory Structure

```
test/
├── data/           # Test data files
├── integration/    # Integration tests
└── e2e/           # End-to-end tests
```

## Test Data

The `data/` directory contains:
- `AuditLogAPI.postman_collection.json` - Postman collection for API testing
- Sample JSON payloads for testing
- Mock data for load testing

## Integration Tests

The `integration/` directory contains:
- `performance_test.go` - Performance benchmarks and load tests
- Database integration tests
- External service integration tests

## End-to-End Tests

The `e2e/` directory is for full end-to-end tests that test the complete system.

## Running Tests

### Unit Tests (in source code)
```bash
# Run all unit tests
task test

# Run tests with coverage
go test -v -cover ./...
```

### Performance Tests
```bash
# Run performance benchmarks
task test-performance

# Run load tests
task load-test
```

### Integration Tests
```bash
# Run integration tests
go test -v ./test/integration/...
```

### Using Test Data

#### Postman Collection
Import `test/data/AuditLogAPI.postman_collection.json` into Postman to test API endpoints interactively.

#### Load Testing Data
The load testing scripts in `/scripts/` use test data from this directory.

## Test Environment

### Prerequisites
- Docker and Docker Compose (for test infrastructure)
- Go 1.21+ (for running tests)
- Task utility (for task automation)

### Setup Test Environment
```bash
# Start test infrastructure
cd deployments && docker-compose up -d

# Run database migrations
task migrate-up

# Generate test JWT token
task generate-token
```

### Cleanup
```bash
# Stop test infrastructure
cd deployments && docker-compose down -v

# Clean build artifacts
task clean
```

## Writing Tests

### Test Naming Convention
- Unit tests: `*_test.go` (alongside source code)
- Integration tests: `test/integration/*_test.go`
- End-to-end tests: `test/e2e/*_test.go`

### Test Data Guidelines
- Use realistic but anonymized data
- Include edge cases and error conditions
- Keep test data minimal but comprehensive
- Document test scenarios clearly
