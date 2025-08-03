.PHONY: help build test lint clean docker-build docker-run fmt vet sec deps

# Default target
help: ## Show this help message
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development
build: ## Build the application
	@echo "Building aws-monitor..."
	@go build -v -o bin/aws-monitor ./cmd/aws-monitor

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out

lint: ## Run linter
	@echo "Running golangci-lint..."
	@golangci-lint run --timeout=10m

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@goimports -w .

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

##@ Security
sec: ## Run security checks
	@echo "Running security checks..."
	@gosec ./...

##@ Docker
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t aws-monitor:latest .

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	@docker run --rm -p 8080:8080 aws-monitor:latest

docker-compose-up: ## Start services with docker-compose
	@echo "Starting services with docker-compose..."
	@docker-compose up -d

docker-compose-down: ## Stop services with docker-compose
	@echo "Stopping services with docker-compose..."
	@docker-compose down

##@ Cleanup
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@rm -f aws-monitor

clean-docker: ## Clean Docker images and containers
	@echo "Cleaning Docker images and containers..."
	@docker-compose down -v --remove-orphans
	@docker system prune -f

##@ Tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest

##@ Validation
validate-config: ## Validate configuration file
	@echo "Validating configuration..."
	@./scripts/validate-config.sh

check: lint vet test ## Run all checks (lint, vet, test)

ci: deps check build validate-config ## Run CI pipeline locally

##@ Release
release-build: ## Build release binaries for multiple platforms
	@echo "Building release binaries..."
	@mkdir -p bin/
	@GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/aws-monitor-linux-amd64 ./cmd/aws-monitor
	@GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o bin/aws-monitor-linux-arm64 ./cmd/aws-monitor
	@GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o bin/aws-monitor-darwin-amd64 ./cmd/aws-monitor
	@GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o bin/aws-monitor-darwin-arm64 ./cmd/aws-monitor
	@GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o bin/aws-monitor-windows-amd64.exe ./cmd/aws-monitor