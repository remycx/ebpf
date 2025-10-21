#!/bin/bash
# Run all Sentinel components in development mode

set -e

echo "Starting Sentinel Fleet Monitoring System..."

# Trap to cleanup on exit
cleanup() {
    echo ""
    echo "Stopping all components..."
    pkill -P $$ || true
    exit 0
}
trap cleanup EXIT INT TERM

# Start Backend
echo "Starting Sentinel Backend..."
cd sentinel/backend
go run . -config config.yaml &
BACKEND_PID=$!
cd ../..

# Wait for backend to start
sleep 2

# Start Frontend
echo "Starting Sentinel Frontend..."
cd sentinel/frontend
npm start &
FRONTEND_PID=$!
cd ../..

echo ""
echo "=== Sentinel is running ==="
echo "Backend: http://localhost:8080"
echo "Frontend: http://localhost:3000"
echo "Agent WebSocket: ws://localhost:8080/api/v1/events"
echo ""
echo "Press Ctrl+C to stop all components"

# Wait for any process to exit
wait
