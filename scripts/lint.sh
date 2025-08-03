#!/bin/bash

# Linting script for AWS Monitor
set -e

echo "🔍 Running Go formatting check..."
if [ "$(gofmt -s -l . | wc -l)" -gt 0 ]; then
    echo "❌ Go files are not formatted. Please run 'go fmt ./...' or 'make fmt'"
    gofmt -s -l .
    exit 1
fi
echo "✅ Go formatting check passed"

echo "🔍 Running Go vet..."
go vet ./...
echo "✅ Go vet passed"

echo "🔍 Running golangci-lint..."
golangci-lint run --timeout=10m
echo "✅ golangci-lint passed"


echo "🎉 All linting checks passed!"