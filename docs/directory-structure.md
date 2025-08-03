# Directory Structure and Required Files

This document outlines the complete directory structure and all required files for the AWS monitoring application.

## Project Directory Structure

```
aws-monitoring/
├── cmd/
│   └── aws-monitor/
│       └── main.go                    # Application entry point
├── internal/
│   ├── config/
│   │   ├── config.go                  # Configuration management
│   │   └── config_test.go             # Configuration tests
│   ├── aws/
│   │   ├── client.go                  # AWS client factory
│   │   ├── client_test.go             # AWS client tests
│   │   └── session.go                 # AWS session management
│   ├── collectors/
│   │   ├── base.go                    # Base collector interface and common logic
│   │   ├── base_test.go               # Base collector tests
│   │   ├── ec2.go                     # EC2 resource collector
│   │   ├── ec2_test.go                # EC2 collector tests
│   │   ├── rds.go                     # RDS resource collector
│   │   ├── rds_test.go                # RDS collector tests
│   │   ├── s3.go                      # S3 resource collector
│   │   ├── s3_test.go                 # S3 collector tests
│   │   ├── lambda.go                  # Lambda resource collector
│   │   ├── lambda_test.go             # Lambda collector tests
│   │   ├── ebs.go                     # EBS resource collector
│   │   ├── ebs_test.go                # EBS collector tests
│   │   ├── elb.go                     # ELB resource collector
│   │   ├── elb_test.go                # ELB collector tests
│   │   ├── vpc.go                     # VPC resource collector
│   │   ├── vpc_test.go                # VPC collector tests
│   │   └── registry.go                # Collector registry
│   ├── metrics/
│   │   ├── exporter.go                # OpenTelemetry metric exporter
│   │   ├── exporter_test.go           # Exporter tests
│   │   ├── provider.go                # Metric provider setup
│   │   └── utils.go                   # Metric creation utilities
│   ├── scheduler/
│   │   ├── scheduler.go               # Collection scheduler
│   │   ├── scheduler_test.go          # Scheduler tests
│   │   └── worker.go                  # Worker pool implementation
│   └── health/
│       ├── handler.go                 # Health check HTTP handler
│       ├── handler_test.go            # Health handler tests
│       └── checker.go                 # Health status checker
├── pkg/
│   ├── logger/
│   │   ├── logger.go                  # Structured logging setup
│   │   └── logger_test.go             # Logger tests
│   └── errors/
│       ├── errors.go                  # Custom error types
│       └── errors_test.go             # Error handling tests
├── configs/
│   ├── config.yaml                    # Default configuration file
│   └── config.example.yaml            # Example configuration
├── deployments/
│   ├── docker/
│   │   ├── Dockerfile                 # Application Dockerfile
│   │   └── docker-compose.yml         # Docker compose setup
│   └── k8s/
│       ├── deployment.yaml            # Kubernetes deployment
│       ├── service.yaml               # Kubernetes service
│       └── configmap.yaml             # Kubernetes configmap
├── scripts/
│   ├── build.sh                       # Build script
│   ├── test.sh                        # Test execution script
│   ├── lint.sh                        # Linting script
│   └── docker-build.sh                # Docker build script
├── test/
│   ├── integration/
│   │   ├── aws_test.go                # AWS integration tests
│   │   ├── otel_test.go               # OpenTelemetry integration tests
│   │   └── e2e_test.go                # End-to-end tests
│   ├── fixtures/
│   │   ├── aws_responses.json         # Mock AWS API responses
│   │   └── test_config.yaml           # Test configuration
│   └── mocks/
│       ├── aws_mock.go                # AWS service mocks
│       └── otel_mock.go               # OpenTelemetry mocks
├── docs/
│   ├── implementation-tasks.md        # Task breakdown (already created)
│   ├── directory-structure.md         # This file
│   ├── architecture.md                # System architecture documentation
│   ├── interfaces.md                  # Interface and class design
│   └── configuration.md               # Configuration documentation
├── .github/
│   └── workflows/
│       ├── ci.yml                     # Continuous integration
│       ├── security.yml               # Security scanning
│       └── release.yml                # Release automation
├── go.mod                             # Go module definition
├── go.sum                             # Go module checksums
├── .golangci.yml                      # Linting configuration
├── .gitignore                         # Git ignore patterns
├── .dockerignore                      # Docker ignore patterns
├── Makefile                           # Build automation
└── README.md                          # Project documentation
```

## Required Files by Category

### Core Application Files
- `cmd/aws-monitor/main.go` - Application entry point and initialization
- `go.mod` - Go module definition with dependencies
- `go.sum` - Dependency checksums

### Configuration Management
- `internal/config/config.go` - Configuration struct and loading logic
- `configs/config.yaml` - Default configuration file
- `configs/config.example.yaml` - Example with all options

### AWS Integration
- `internal/aws/client.go` - AWS service client factory
- `internal/aws/session.go` - AWS session and credential management

### Metric Collectors
- `internal/collectors/base.go` - Common collector interface and base functionality
- `internal/collectors/{service}.go` - Individual service collectors (EC2, RDS, etc.)
- `internal/collectors/registry.go` - Collector registration and management

### OpenTelemetry Integration
- `internal/metrics/provider.go` - OTEL metric provider setup
- `internal/metrics/exporter.go` - OTEL collector exporter
- `internal/metrics/utils.go` - Metric creation utilities

### Scheduling and Health
- `internal/scheduler/scheduler.go` - Metric collection scheduling
- `internal/health/handler.go` - HTTP health check endpoint

### Infrastructure Files
- `deployments/docker/Dockerfile` - Container image definition
- `deployments/docker/docker-compose.yml` - Local development setup
- `.github/workflows/ci.yml` - CI/CD pipeline
- `.golangci.yml` - Code quality and linting rules

### Testing Infrastructure
- `test/integration/` - Integration test suite
- `test/mocks/` - Mock implementations for testing
- All `*_test.go` files - Unit tests for each component

### Build and Development
- `Makefile` - Build automation and common tasks
- `scripts/*.sh` - Development and deployment scripts
- `.gitignore` - Version control exclusions
- `.dockerignore` - Docker build exclusions

## File Dependencies

### Critical Path Files (Must implement first)
1. `go.mod` - Project initialization
2. `internal/config/config.go` - Configuration management
3. `internal/aws/client.go` - AWS connectivity
4. `internal/collectors/base.go` - Collector foundation
5. `internal/metrics/provider.go` - OTEL setup

### Secondary Files (Can implement in parallel)
- Individual collector implementations
- Test files
- Health check system
- Scheduler implementation

### Optional Files (Nice to have)
- Kubernetes deployments
- Advanced monitoring and alerting
- Performance optimization utilities