#!/bin/bash

# oReader v2.0.0 Development Server Starter (API-only mode)

echo "🚀 Starting oReader v2.0.0 (Development Mode)..."

# Set environment variables
export DATABASE_URL=oreader.db
export JWT_SECRET_KEY=dev-secret-key-min-32-chars-change-in-prod
export ENV=development
export PORT=8080

echo "✓ Environment: $ENV, Port: $PORT"
echo "📝 API-only mode - run 'make frontend-dev' in another terminal for frontend"
echo ""

# Run with noembed tag to skip frontend embedding
go run -tags noembed ./cmd/server
