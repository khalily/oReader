#!/bin/bash

# oReader v2.0.0 Full Development Environment
# Starts both backend API and frontend Vite dev server

set -e

# Cleanup function to kill all background processes
trap 'echo ""; echo "Stopping servers..."; kill $(jobs -p) 2>/dev/null || true' EXIT

# Set environment variables
export DATABASE_URL=oreader.db
export JWT_SECRET_KEY=dev-secret-key-min-32-chars-change-in-prod
export ENV=development
export PORT=8080
export FRONTEND_URL=http://10.37.126.68:5173
export LOG_LEVEL=debug
export GITHUB_CLIENT_ID=Ov23lioGHWXTYNnrwADJ
export GITHUB_CLIENT_SECRET=d333de867462b7afa421715476c02c0cafbe7606
export GITHUB_CALLBACK_HOST=10.37.126.68:5173


echo "🚀 Starting Full Development Environment..."
echo "  Backend API:  http://localhost:$PORT"
echo "  Frontend Dev: http://localhost:5173"
echo ""
echo "Press Ctrl+C to stop all servers"
echo ""

# Start backend with noembed tag
go run -tags noembed ./cmd/server &
BACKEND_PID=$!

# Give backend a moment to start
sleep 1

# Start frontend dev server
cd web && npm run dev &
FRONTEND_PID=$!
cd ..

# Wait for all background processes
wait
