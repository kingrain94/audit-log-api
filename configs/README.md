# Configuration Files

This directory contains configuration file templates and default configurations for the Audit Log API.

## Files

- `env.example` - Example environment variables file
- `dbconfig.yml` - Database migration configuration

## Usage

### Environment Variables

1. Copy the example environment file:
   ```bash
   cp configs/env.example .env
   ```

2. Edit the `.env` file with your specific configuration values:
   ```bash
   # Update with your actual values
   JWT_SECRET_KEY=your-actual-secret-key
   DATABASE_WRITER_URL=your-database-url
   ```

### Database Configuration

The `dbconfig.yml` file is used by sql-migrate for database migrations:

```bash
# Run migrations
sql-migrate up -config=configs/dbconfig.yml

# Rollback migrations  
sql-migrate down -config=configs/dbconfig.yml
```

## Configuration Categories

### Server Settings
- `SERVER_PORT`: API server port (default: 10000)
- `APP_ENV`: Environment (development/production)

### Security
- `JWT_SECRET_KEY`: JWT signing secret (use strong random key in production)
- `JWT_EXPIRATION_HOURS`: Token expiration time (default: 24 hours)

### Rate Limiting
- `DEFAULT_RATE_LIMIT`: Per-tenant rate limit (requests per minute)
- `GLOBAL_RATE_LIMIT`: Global rate limit per IP (requests per minute)

### Database
- `DATABASE_WRITER_URL`: Primary database connection string
- `DATABASE_READER_URL`: Read replica connection string

### External Services
- Redis, AWS (S3, SQS), OpenSearch connection settings

## Security Notes

- Never commit actual secrets to version control
- Use strong, randomly generated JWT secrets in production
- Rotate secrets regularly
- Use environment-specific configuration files
