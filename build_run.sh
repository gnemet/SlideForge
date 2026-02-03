#!/bin/bash

# SlideForge Build and Run Script

# Create necessary directories
mkdir -p data/templates data/uploads data/thumbnails

# Check if DB exists, if not create a warning (assume user handles DB for now)
# psql -lqt | cut -d \| -f 1 | grep -qw slideforge || createdb slideforge

# Run migrations (Optional: use a migration tool)
# psql slideforge < database/migrations/001_init.sql

echo "Building SlideForge..."
go build -o bin/slideforge cmd/server/main.go

echo "Starting SlideForge on http://localhost:8080"
./bin/slideforge
