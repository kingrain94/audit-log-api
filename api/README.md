# API Specifications

This directory contains OpenAPI/Swagger specifications and protocol definitions for the Audit Log API.

## Files

- `openapi.yaml` - OpenAPI 3.0 specification in YAML format
- `openapi.json` - OpenAPI 3.0 specification in JSON format  
- `swagger.yaml` - Swagger specification in YAML format
- `swagger.json` - Swagger specification in JSON format

## Usage

### Viewing API Documentation

When the API server is running, you can view the interactive API documentation at:
- http://localhost:10000/swagger/index.html

### Generating Client SDKs

You can use these specifications to generate client SDKs in various languages:

```bash
# Generate Go client
swagger generate client -f api/openapi.yaml -A audit-log-api

# Generate TypeScript client  
openapi-generator-cli generate -i api/openapi.yaml -g typescript-axios -o clients/typescript

# Generate Python client
openapi-generator-cli generate -i api/openapi.yaml -g python -o clients/python
```

### Validation

To validate the API specifications:

```bash
# Validate OpenAPI spec
swagger validate api/openapi.yaml

# Or using openapi-generator
openapi-generator-cli validate -i api/openapi.yaml
```
