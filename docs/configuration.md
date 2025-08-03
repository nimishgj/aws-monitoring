# Configuration Documentation

This document describes the configuration system for the AWS monitoring application using a single config.yaml file.

## Configuration File Structure

The application uses a single YAML configuration file (`config.yaml`) with the following structure:

```yaml
# List of AWS regions to collect metrics from
enabled_regions:
  - us-east-1
  - us-west-2
  - eu-west-1
  - ap-southeast-1

# AWS service configuration
aws:
  # AWS credentials (required)
  access_key_id: "your_access_key"
  secret_access_key: "your_secret_key"
  
  # Default region for AWS operations
  default_region: us-east-1
  
  # Maximum number of retries for AWS API calls
  max_retries: 3
  
  # Timeout for AWS API calls
  timeout: 30s

# OpenTelemetry configuration
otel:
  # OpenTelemetry collector endpoint (required)
  collector_endpoint: "http://localhost:4317"
  
  # Service name for tracing and metrics (required)
  service_name: "aws-monitor"
  
  # Additional headers for OTEL collector
  headers:
    Authorization: "Bearer <token>"
    Custom-Header: "value"
  
  # Use insecure connection (for development)
  insecure: false
  
  # Batch configuration
  batch_timeout: 5s
  batch_size: 512

# Metrics collection configuration
metrics:
  ec2:
    enabled: true
    collection_interval: 300s
  
  rds:
    enabled: true
    collection_interval: 300s
  
  s3:
    enabled: false
    collection_interval: 600s
  
  lambda:
    enabled: true
    collection_interval: 300s
  
  ebs:
    enabled: true
    collection_interval: 300s
  
  elb:
    enabled: true
    collection_interval: 300s
  
  vpc:
    enabled: true
    collection_interval: 600s

# Global application settings
global:
  # Logging configuration
  log_level: "info"          # debug, info, warn, error
  log_format: "json"         # json, text
  
  # Health check HTTP server
  health_check_port: 8080
  health_check_path: "/health"
  
  # Default collection interval (used if not specified per metric)
  default_collection_interval: 300s
  
  # Worker pool configuration
  max_concurrent_workers: 10
  worker_timeout: 60s
  
  # Error handling
  max_error_count: 5         # Max consecutive errors before disabling collector
  error_reset_interval: 300s # Time to reset error count
  
  # Performance tuning
  metric_buffer_size: 1000
  export_timeout: 30s
```

## Configuration File Location

The application looks for the configuration file in the following order:

1. `./config.yaml` (current directory)
2. `./configs/config.yaml` (configs subdirectory)
3. `/etc/aws-monitor/config.yaml` (system-wide configuration)

You can also specify a custom path using the `-config` command line flag:

```bash
./aws-monitor -config /path/to/your/config.yaml
```

## Configuration Validation

The application validates configuration on startup and will fail to start if required values are missing or invalid.

### Required Configuration

1. **AWS Credentials**: Must be specified in config.yaml
2. **AWS Regions**: At least one region must be specified in enabled_regions
3. **OTEL Endpoint**: Valid OpenTelemetry collector endpoint
4. **Service Name**: Service name for telemetry identification

### Configuration Validation Rules

```go
type ValidationRule struct {
    Field       string
    Required    bool
    Validator   func(interface{}) error
}

var ConfigValidationRules = []ValidationRule{
    {
        Field:    "enabled_regions",
        Required: true,
        Validator: func(v interface{}) error {
            regions := v.([]string)
            if len(regions) == 0 {
                return errors.New("at least one region must be enabled")
            }
            return nil
        },
    },
    {
        Field:    "otel.collector_endpoint",
        Required: true,
        Validator: func(v interface{}) error {
            endpoint := v.(string)
            if _, err := url.Parse(endpoint); err != nil {
                return fmt.Errorf("invalid OTEL collector endpoint: %w", err)
            }
            return nil
        },
    },
    // Additional validation rules...
}
```

## Configuration Examples

### Minimal Configuration

```yaml
enabled_regions:
  - us-east-1

aws:
  access_key_id: "your_access_key"
  secret_access_key: "your_secret_key"
  default_region: us-east-1

otel:
  collector_endpoint: "http://localhost:4317"
  service_name: "aws-monitor"

metrics:
  ec2:
    enabled: true
```

### Production Configuration

```yaml
enabled_regions:
  - us-east-1
  - us-west-2
  - eu-west-1

aws:
  access_key_id: "AKIA1234567890EXAMPLE"
  secret_access_key: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
  default_region: us-east-1
  max_retries: 5
  timeout: 60s

otel:
  collector_endpoint: "https://otel-collector.company.com:4317"
  service_name: "aws-monitor-prod"
  headers:
    Authorization: "Bearer your-auth-token"
  insecure: false
  batch_timeout: 10s
  batch_size: 1000

metrics:
  ec2:
    enabled: true
    collection_interval: 300s
  rds:
    enabled: true
    collection_interval: 300s
  s3:
    enabled: true
    collection_interval: 600s
  lambda:
    enabled: true
    collection_interval: 300s
  ebs:
    enabled: true
    collection_interval: 300s
  elb:
    enabled: true
    collection_interval: 300s
  vpc:
    enabled: true
    collection_interval: 600s

global:
  log_level: "warn"
  log_format: "json"
  health_check_port: 8080
  max_concurrent_workers: 20
  max_error_count: 3
  error_reset_interval: 600s
  metric_buffer_size: 2000
  export_timeout: 45s
```

### Development Configuration

```yaml
enabled_regions:
  - us-east-1

aws:
  access_key_id: "your_dev_access_key"
  secret_access_key: "your_dev_secret_key"
  default_region: us-east-1
  max_retries: 1
  timeout: 10s

otel:
  collector_endpoint: "http://localhost:4317"
  service_name: "aws-monitor-dev"
  insecure: true
  batch_timeout: 1s
  batch_size: 10

metrics:
  ec2:
    enabled: true
    collection_interval: 60s
  rds:
    enabled: false
  s3:
    enabled: false
  lambda:
    enabled: false
  ebs:
    enabled: false
  elb:
    enabled: false
  vpc:
    enabled: false

global:
  log_level: "debug"
  log_format: "text"
  health_check_port: 8080
  max_concurrent_workers: 2
  max_error_count: 10
  metric_buffer_size: 100
```

## Configuration Loading Priority

The application loads configuration in the following order (later values override earlier ones):

1. **Default Values**: Hard-coded defaults in the application
2. **Configuration File**: Values from config.yaml
3. **Command Line Flags**: Values from CLI arguments (if implemented)

## Dynamic Configuration

### Hot Reload Support

The application can optionally support hot-reloading of configuration:

```yaml
global:
  enable_hot_reload: true
  config_reload_interval: 60s
```

### Runtime Configuration Changes

Certain configuration values can be changed at runtime:

- **Log Level**: Can be changed via HTTP endpoint
- **Collector Enable/Disable**: Can be toggled via health check endpoint
- **Collection Intervals**: Can be adjusted via configuration reload

### Configuration API Endpoints

```bash
# Get current configuration
GET /config

# Update log level
POST /config/log-level
{
  "level": "debug"
}

# Enable/disable collector
POST /config/collectors/ec2
{
  "enabled": false
}

# Reload configuration from file
POST /config/reload
```

## Security Considerations

### Credential Management

1. **Store credentials securely in configuration files**
2. **Restrict configuration file permissions (600)**
3. **Rotate credentials regularly**
4. **Use least-privilege permissions**

### Configuration File Security

1. **Restrict file permissions** (600 or 644)
2. **Store in secure location**
3. **Avoid world-readable permissions**
4. **Use configuration management tools for production**

### Configuration File Protection

1. **Use secure file storage**
2. **Avoid logging configuration contents**
3. **Clear sensitive data from memory after use**
4. **Use encrypted storage for configuration files**

This configuration system provides simplicity and centralized management through a single configuration file.