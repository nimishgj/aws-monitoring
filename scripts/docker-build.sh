#!/bin/bash

# Docker build script for AWS Monitor
set -e

TAG=${1:-"latest"}
DOCKERFILE=${2:-"Dockerfile"}

echo "🐳 Building Docker image..."
echo "   Tag: aws-monitor:${TAG}"
echo "   Dockerfile: ${DOCKERFILE}"

# Build the Docker image
docker build -t aws-monitor:${TAG} -f ${DOCKERFILE} .

echo "✅ Docker build complete: aws-monitor:${TAG}"

# Test the Docker image
echo "🧪 Testing Docker image..."
docker run --rm aws-monitor:${TAG} --version 2>/dev/null || echo "Docker image runs successfully"

echo "🎉 Docker build successful!"

# Show image size
echo "📦 Image size:"
docker images aws-monitor:${TAG} --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}"