#!/bin/bash
# Build all Sentinel components

set -e

echo "Building Sentinel Fleet Monitoring System..."

# Build Agent
echo ""
echo "=== Building Agent ==="
cd agent
make clean
make all
cd ..

# Build Backend
echo ""
echo "=== Building Backend ==="
cd sentinel/backend
make clean
make build
cd ../..

# Build Frontend
echo ""
echo "=== Building Frontend ==="
cd sentinel/frontend
npm install
npm run build
cd ../..

echo ""
echo "=== Build Complete ==="
echo "Agent binary: agent/build/sentinel-agent"
echo "Backend binary: sentinel/backend/build/sentinel"
echo "Frontend build: sentinel/frontend/build/"
