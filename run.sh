#!/bin/bash

# oReader v2.0.0 Development Server Starter

echo "🚀 Starting oReader v2.0.0..."

# Set environment variables
export DATABASE_URL=oreader.db
export JWT_SECRET_KEY=dev-secret-key-min-32-chars-change-in-prod
export ENV=development
export PORT=8080

echo "✓ Environment variables set"
echo "  DATABASE_URL=$DATABASE_URL"
echo "  ENV=$ENV"
echo "  PORT=$PORT"
echo ""

# Run the server
make run
