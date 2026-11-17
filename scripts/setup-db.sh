#!/bin/bash

set -e

echo "🔧 Setting up ASF Database..."

# Check if docker is available
if ! command -v docker &> /dev/null; then
	echo "❌ Docker is not installed. Please install Docker."
	exit 1
fi

# Start PostgreSQL
echo "📦 Starting PostgreSQL container..."
docker compose up -d postgres

# Wait for PostgreSQL to be ready
echo "⏳ Waiting for PostgreSQL to be ready..."
sleep 5

# Test connection
echo "✅ PostgreSQL is ready"

# Environment setup info
echo ""
echo "📝 Database Configuration:"
echo "   Host: localhost"
echo "   Port: 5432"
echo "   User: postgres"
echo "   Password: postgres"
echo "   Database: asf"
echo ""
echo "🔗 Connection string:"
echo "   postgresql://postgres:postgres@localhost:5432/asf"
echo ""
echo "✨ Set these environment variables to enable persistence:"
echo "   export DB_HOST=localhost"
echo "   export DB_PORT=5432"
echo "   export DB_USER=postgres"
echo "   export DB_PASSWORD=postgres"
echo "   export DB_NAME=asf"
