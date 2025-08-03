#!/bin/bash

# Configuration validation script for AWS Monitor
set -e

CONFIG_FILE=${1:-"configs/config.yaml"}

# If default config doesn't exist, try example config
if [ ! -f "${CONFIG_FILE}" ] && [ "${CONFIG_FILE}" = "configs/config.yaml" ]; then
    if [ -f "configs/config.example.yaml" ]; then
        CONFIG_FILE="configs/config.example.yaml"
        echo "📝 Using example config for validation: ${CONFIG_FILE}"
    fi
fi

echo "🔍 Validating configuration file: ${CONFIG_FILE}"

# Check if config file exists
if [ ! -f "${CONFIG_FILE}" ]; then
    echo "❌ Configuration file not found: ${CONFIG_FILE}"
    exit 1
fi

# Build the application if binary doesn't exist
if [ ! -f "aws-monitor" ] && [ ! -f "bin/aws-monitor" ]; then
    echo "🔨 Building application..."
    go build -o aws-monitor ./cmd/aws-monitor
fi

# Use existing binary or build one
BINARY="./aws-monitor"
if [ -f "bin/aws-monitor" ]; then
    BINARY="./bin/aws-monitor"
fi

# Validate configuration
echo "🧪 Running configuration validation..."
if ${BINARY} --validate --config "${CONFIG_FILE}"; then
    echo "✅ Configuration validation successful"
else
    echo "❌ Configuration validation failed"
    exit 1
fi

# Check YAML syntax
echo "🔍 Checking YAML syntax..."
if command -v yamllint &> /dev/null; then
    if yamllint "${CONFIG_FILE}" 2>/dev/null; then
        echo "✅ YAML syntax check passed"
    else
        echo "⚠️  yamllint found style issues, but YAML is syntactically valid"
    fi
elif command -v yq &> /dev/null; then
    if yq eval '.' "${CONFIG_FILE}" >/dev/null 2>&1; then
        echo "✅ YAML syntax validated with yq"
    else
        echo "❌ YAML syntax validation failed"
        exit 1
    fi
else
    echo "⚠️  No YAML validator found, skipping syntax check"
fi

echo "🎉 All configuration checks passed!"