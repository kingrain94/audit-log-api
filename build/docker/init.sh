#!/bin/bash
set -e

# Create extensions if they don't exist
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;
EOSQL

# Create audit_log database if it doesn't exist
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    SELECT 'CREATE DATABASE audit_log'
    WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'audit_log')\gexec
EOSQL

# Connect to audit_log database and create extensions
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "audit_log" <<-EOSQL
    CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;
EOSQL

# Run migrations if they exist
for file in /docker-entrypoint-initdb.d/migrations/*.up.sql; do
    if [ -f "$file" ]; then
        echo "Running migration: $file"
        psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "audit_log" -f "$file"
    fi
done 