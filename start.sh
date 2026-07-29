#!/bin/bash

echo "🚀 Starting FinTrack Server..."
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null
then
    echo "❌ Go is not installed. Please install Go 1.21 or higher."
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | awk '{print $3}')
echo "✅ Go version: $GO_VERSION"
echo ""

# Download dependencies if needed
if [ ! -f "go.sum" ]; then
    echo "📦 Downloading dependencies..."
    go mod tidy
    echo ""
fi

# Run the application
echo "🌟 Server will start at http://localhost:8080"
echo "📱 Pages available:"
echo "   - Login:      http://localhost:8080/login"
echo "   - Dashboard:  http://localhost:8080/"
echo "   - Statistics: http://localhost:8080/stats"
echo "   - Targets:    http://localhost:8080/targets"
echo "   - Profile:    http://localhost:8080/profile"
echo ""
echo "Press Ctrl+C to stop the server"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

go run main.go
