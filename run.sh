#!/bin/bash

# oReader v2.0.0 Backend Server (API-only mode)
# Usage: ./run.sh

set -e

# ---- Environment Variables ----
export DATABASE_URL=oreader.db
export JWT_SECRET_KEY=dev-secret-key-min-32-chars-change-in-prod
export ENV=development
export PORT=8080

# Paper Import Configuration
export PAPER_GRPC_ADDR=localhost:50051
export PAPER_UPLOAD_DIR=uploads/papers
export PAPER_MAX_UPLOAD_SIZE=52428800

# Load additional config from .env (OAuth keys, etc.)
if [ -f .env ]; then
    set -a
    source .env
    set +a
fi

echo "🚀 Starting oReader Backend (API-only)..."
echo "  Port: $PORT"
echo "  Env:  $ENV"
echo ""

# Run with noembed tag to skip frontend embedding
go run -tags noembed ./cmd/server
