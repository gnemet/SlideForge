#!/bin/bash
# Apply SlideForge OLTP Schema
# Consistent with Jiramntr database management style

set -e

# Load environment variables
if [ -f ../../.env ]; then
    set -a; source ../../.env; set +a
    echo "Loaded .env from project root"
elif [ -f .env ]; then
    set -a; source .env; set +a
    echo "Loaded local .env"
fi

PG_CONN="postgresql://${PG_USER}:${PG_PASSWORD}@${PG_HOST}:${PG_PORT}/${PG_DB}?sslmode=${PG_SSLMODE}"

echo "Applying DDL to ${PG_DB}..."
psql "${PG_CONN}" -f ddl_oltp.sql

echo "Database initialization complete."
