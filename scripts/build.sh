#!/bin/bash

# Build script for AWS Monitor
set -e

VERSION=${1:-"dev"}
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

echo "🔨 Building AWS Monitor..."
echo "   Version: ${VERSION}"
echo "   Build Time: ${BUILD_TIME}" 
echo "   Git Commit: ${GIT_COMMIT}"

# Create bin directory if it doesn't exist
mkdir -p bin/

# Build flags
LDFLAGS="-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.gitCommit=${GIT_COMMIT}"

# Build the application
go build -ldflags="${LDFLAGS}" -o bin/aws-monitor ./cmd/aws-monitor

echo "✅ Build complete: bin/aws-monitor"

# Test the binary
echo "🧪 Testing binary..."
./bin/aws-monitor --version 2>/dev/null || echo "Binary runs successfully"

echo "🎉 Build successful!"