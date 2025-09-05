# Build and Packaging

This directory contains packaging and Continuous Integration configurations.

## Directory Structure

```
build/
├── ci/          # CI/CD configurations
├── docker/      # Docker-related files
└── package/     # OS package configurations
```

## Docker

The `docker/` directory contains Docker-related files:

- `Dockerfile.db` - Database container configuration
- `init.sh` - Database initialization script

### Building Docker Images

```bash
# Build database image
docker build -f build/docker/Dockerfile.db -t audit-log-db .

# Or use docker-compose (from deployments/)
cd deployments && docker-compose build
```

## CI/CD

Place your CI/CD configuration files in the `ci/` directory:

- GitHub Actions: `.github/workflows/` (in project root)
- Travis CI: `ci/.travis.yml`
- CircleCI: `ci/.circleci/config.yml`
- Jenkins: `ci/Jenkinsfile`

## Package

The `package/` directory is for OS package configurations:

- Debian packages: `package/deb/`
- RPM packages: `package/rpm/`
- Container images: `package/container/`

## Examples

### GitHub Actions Workflow

```yaml
# .github/workflows/ci.yml
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v3
        with:
          go-version: '1.21'
      - run: task test
      - run: task build-all
```

### Docker Multi-stage Build

```dockerfile
# build/package/Dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN task build-all

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/bin/ ./
CMD ["./audit-log-api"]
```
