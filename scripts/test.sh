#!/bin/bash

# Test script for AWS Monitor
set -e

echo "🧪 Running Go tests..."

# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...

# Generate coverage report
echo "📊 Generating coverage report..."
go tool cover -func=coverage.out

# Check coverage threshold (90%)
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print substr($3, 1, length($3)-1)}')
THRESHOLD=90

echo "📈 Coverage: ${COVERAGE}%"

if (( $(echo "$COVERAGE < $THRESHOLD" | bc -l) )); then
    echo "❌ Coverage ${COVERAGE}% is below threshold ${THRESHOLD}%"
    exit 1
fi

echo "✅ Coverage ${COVERAGE}% meets threshold ${THRESHOLD}%"

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html
echo "📄 HTML coverage report generated: coverage.html"

echo "🎉 All tests passed!"