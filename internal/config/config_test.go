package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		expectError bool
		validate    func(*Config) bool
	}{
		{
			name: "valid minimal config",
			configYAML: `
enabled_regions:
  - us-east-1
aws:
  access_key_id: "test-key"
  secret_access_key: "test-secret"
  default_region: us-east-1
otel:
  collector_endpoint: "http://localhost:4317"
  service_name: "aws-monitor"
metrics:
  ec2:
    enabled: true
`,
			expectError: false,
			validate: func(c *Config) bool {
				return len(c.EnabledRegions) == 1 &&
					c.EnabledRegions[0] == "us-east-1" &&
					c.AWS.AccessKeyID == "test-key" &&
					c.AWS.DefaultRegion == "us-east-1" &&
					c.OTEL.ServiceName == "aws-monitor"
			},
		},
		{
			name: "valid complete config",
			configYAML: `
enabled_regions:
  - us-east-1
  - us-west-2
aws:
  access_key_id: "test-key"
  secret_access_key: "test-secret"
  default_region: us-east-1
  max_retries: 5
  timeout: 60s
otel:
  collector_endpoint: "https://otel.example.com:4317"
  service_name: "aws-monitor-prod"
  headers:
    Authorization: "Bearer token"
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
    enabled: false
    collection_interval: 600s
global:
  log_level: "warn"
  log_format: "json"
  health_check_port: 8080
  max_concurrent_workers: 20
`,
			expectError: false,
			validate: func(c *Config) bool {
				return len(c.EnabledRegions) == 2 &&
					c.AWS.MaxRetries == 5 &&
					time.Duration(c.AWS.Timeout) == 60*time.Second &&
					c.OTEL.BatchSize == 1000 &&
					c.Global.LogLevel == "warn" &&
					c.Global.MaxConcurrentWorkers == 20 &&
					c.Metrics.EC2.Enabled == true &&
					c.Metrics.S3.Enabled == false
			},
		},
		{
			name: "missing required fields",
			configYAML: `
enabled_regions: []
aws:
  access_key_id: ""
otel:
  service_name: ""
`,
			expectError: true,
		},
		{
			name: "invalid region format",
			configYAML: `
enabled_regions:
  - ""
aws:
  access_key_id: "test-key"
  secret_access_key: "test-secret"
  default_region: us-east-1
otel:
  collector_endpoint: "http://localhost:4317"
  service_name: "aws-monitor"
`,
			expectError: true,
		},
		{
			name: "default region not in enabled regions",
			configYAML: `
enabled_regions:
  - us-west-1
aws:
  access_key_id: "test-key"
  secret_access_key: "test-secret"
  default_region: us-east-1
otel:
  collector_endpoint: "http://localhost:4317"
  service_name: "aws-monitor"
`,
			expectError: true,
		},
		{
			name: "invalid URL",
			configYAML: `
enabled_regions:
  - us-east-1
aws:
  access_key_id: "test-key"
  secret_access_key: "test-secret"
  default_region: us-east-1
otel:
  collector_endpoint: "invalid-url"
  service_name: "aws-monitor"
`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "config.yaml")
			err := os.WriteFile(configPath, []byte(tt.configYAML), 0600)
			if err != nil {
				t.Fatalf("Failed to create test config file: %v", err)
			}

			// Load configuration
			config, err := Load(configPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.validate != nil && !tt.validate(config) {
				t.Errorf("Config validation failed")
			}
		})
	}
}

func TestDefaults(t *testing.T) {
	configYAML := `
enabled_regions:
  - us-east-1
aws:
  access_key_id: "test-key"
  secret_access_key: "test-secret"
  default_region: us-east-1
otel:
  collector_endpoint: "http://localhost:4317"
  service_name: "aws-monitor"
metrics:
  ec2:
    enabled: true
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(configPath, []byte(configYAML), 0600)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	config, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test AWS defaults
	if config.AWS.MaxRetries != 3 {
		t.Errorf("Expected AWS.MaxRetries to be 3, got %d", config.AWS.MaxRetries)
	}
	if time.Duration(config.AWS.Timeout) != 30*time.Second {
		t.Errorf("Expected AWS.Timeout to be 30s, got %s", config.AWS.Timeout)
	}

	// Test OTEL defaults
	if config.OTEL.BatchSize != 512 {
		t.Errorf("Expected OTEL.BatchSize to be 512, got %d", config.OTEL.BatchSize)
	}
	if time.Duration(config.OTEL.BatchTimeout) != 5*time.Second {
		t.Errorf("Expected OTEL.BatchTimeout to be 5s, got %s", config.OTEL.BatchTimeout)
	}

	// Test Global defaults
	if config.Global.LogLevel != "info" {
		t.Errorf("Expected Global.LogLevel to be 'info', got %s", config.Global.LogLevel)
	}
	if config.Global.LogFormat != "json" {
		t.Errorf("Expected Global.LogFormat to be 'json', got %s", config.Global.LogFormat)
	}
	if config.Global.HealthCheckPort != 8080 {
		t.Errorf("Expected Global.HealthCheckPort to be 8080, got %d", config.Global.HealthCheckPort)
	}
	if config.Global.MaxConcurrentWorkers != 10 {
		t.Errorf("Expected Global.MaxConcurrentWorkers to be 10, got %d", config.Global.MaxConcurrentWorkers)
	}

	// Test collector defaults
	if time.Duration(config.Metrics.EC2.CollectionInterval) != 300*time.Second {
		t.Errorf("Expected EC2.CollectionInterval to be 300s, got %s", config.Metrics.EC2.CollectionInterval)
	}
	if time.Duration(config.Metrics.S3.CollectionInterval) != 600*time.Second {
		t.Errorf("Expected S3.CollectionInterval to be 600s, got %s", config.Metrics.S3.CollectionInterval)
	}
}

func TestGetCollectorConfig(t *testing.T) {
	config := &Config{
		Metrics: MetricsConfig{
			EC2: CollectorConfig{
				Enabled:            true,
				CollectionInterval: Duration(300 * time.Second),
			},
			RDS: CollectorConfig{
				Enabled:            false,
				CollectionInterval: Duration(600 * time.Second),
			},
		},
	}

	// Test valid collector
	ec2Config, err := config.GetCollectorConfig("ec2")
	if err != nil {
		t.Errorf("Unexpected error getting EC2 config: %v", err)
	}
	if !ec2Config.Enabled {
		t.Errorf("Expected EC2 to be enabled")
	}

	// Test invalid collector
	_, err = config.GetCollectorConfig("invalid")
	if err == nil {
		t.Errorf("Expected error for invalid collector")
	}
}

func TestSave(t *testing.T) {
	config := &Config{
		EnabledRegions: []string{"us-east-1"},
		AWS: AWSConfig{
			AccessKeyID:     "test-key",
			SecretAccessKey: "test-secret",
			DefaultRegion:   "us-east-1",
		},
		OTEL: OTELConfig{
			CollectorEndpoint: "http://localhost:4317",
			ServiceName:       "aws-monitor",
		},
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "saved-config.yaml")

	err := config.Save(configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file exists and has correct permissions
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Config file not created: %v", err)
	}

	mode := info.Mode()
	if mode != 0600 {
		t.Errorf("Expected file permissions 0600, got %o", mode)
	}

	// Load the saved config and verify
	loadedConfig, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loadedConfig.AWS.AccessKeyID != config.AWS.AccessKeyID {
		t.Errorf("Config not saved correctly")
	}
}

func TestFindConfigFile(t *testing.T) {
	// Test when config file doesn't exist
	_, err := findConfigFile()
	if err == nil {
		t.Errorf("Expected error when config file doesn't exist")
	}

	// Create a config file in current directory
	tmpConfig := "config.yaml"
	err = os.WriteFile(tmpConfig, []byte("test"), 0600)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}
	defer os.Remove(tmpConfig)

	path, err := findConfigFile()
	if err != nil {
		t.Errorf("Unexpected error finding config file: %v", err)
	}
	if path != "./config.yaml" {
		t.Errorf("Expected './config.yaml', got %s", path)
	}
}

func TestDurationUnmarshal(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		hasError bool
	}{
		{"30s", 30 * time.Second, false},
		{"5m", 5 * time.Minute, false},
		{"1h", 1 * time.Hour, false},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var d Duration
			err := d.UnmarshalYAML(&yaml.Node{Value: tt.input, Kind: yaml.ScalarNode})

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error for input %s", tt.input)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for input %s: %v", tt.input, err)
				return
			}

			if time.Duration(d) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, time.Duration(d))
			}
		})
	}
}
