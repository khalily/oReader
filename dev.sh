#!/bin/bash

# oReader v2.0.0 Development Environment
# One-click: starts backend + frontend + gRPC converter service
# Usage: ./dev.sh
# Press Ctrl+C to stop all services

set -e

# Cleanup: kill all background processes on exit
trap 'echo ""; echo "Stopping all services..."; kill $(jobs -p) 2>/dev/null || true; exit 0' EXIT INT TERM

# ---- Environment Variables ----
export DATABASE_URL=oreader.db
export JWT_SECRET_KEY=dev-secret-key-min-32-chars-change-in-prod
export ENV=development
export PORT=8080
export FRONTEND_URL=http://10.37.126.68:5173
export LOG_LEVEL=debug

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

echo "🚀 Starting oReader Development Environment..."
echo "  Backend:    http://localhost:$PORT"
echo "  Frontend:   http://localhost:5173"
echo "  Converter:  gRPC localhost:50051"
echo ""
echo "Press Ctrl+C to stop all services"
echo ""

# Start converter gRPC service
(cd converter && pip install -r requirements.txt > /dev/null && python server.py) &

# Start backend API server
go run -tags noembed ./cmd/server &

# Start frontend dev server
(cd web && npm run dev -- --host) &

# Wait for any process to exit
wait
