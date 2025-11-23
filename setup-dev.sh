#!/bin/bash
# Setup script for local development

set -e

echo "Setting up WhatsDown for local development..."

# Build frontend
echo "Building frontend..."
cd frontend
if [ ! -d "node_modules" ]; then
    echo "Installing frontend dependencies..."
    npm install
fi
npm run build
cd ..

# Copy web directory to cmd/server/web for Go embed
echo "Setting up web files for Go embed..."
mkdir -p cmd/server/web
cp -r web/* cmd/server/web/ 2>/dev/null || true

echo "Setup complete! You can now run:"
echo "  go run cmd/server/main.go"
echo ""
echo "Or for frontend development:"
echo "  cd frontend && npm run dev"

