// Package config provides configuration management for the aws-monitor application.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

// Duration is a custom type for handling time.Duration in YAML
type Duration time.Duration

// UnmarshalYAML implements yaml.Unmarshaler for Duration
func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}

	duration, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration format: %w", err)
	}

	*d = Duration(duration)
	return nil
}

// MarshalYAML implements yaml.Marshaler for Duration
func (d Duration) MarshalYAML() (interface{}, error) {
	return time.Duration(d).String(), nil
}

// String returns the string representation of Duration
func (d Duration) String() string {
	return time.Duration(d).String()
}

// Config represents the complete application configuration
type Config struct {
	EnabledRegions []string      `yaml:"enabled_regions" validate:"required,min=1"`
	AWS            AWSConfig     `yaml:"aws" validate:"required"`
	OTEL           OTELConfig    `yaml:"otel" validate:"required"`
	Metrics        MetricsConfig `yaml:"metrics" validate:"required"`
	Global         GlobalConfig  `yaml:"global"`
}

// AWSConfig holds AWS-specific configuration
type AWSConfig struct {
	AccessKeyID     string   `yaml:"access_key_id" validate:"required"`
	SecretAccessKey string   `yaml:"secret_access_key" validate:"required"`
	DefaultRegion   string   `yaml:"default_region" validate:"required"`
	MaxRetries      int      `yaml:"max_retries" validate:"min=1,max=10"`
	Timeout         Duration `yaml:"timeout"`
}

// OTELConfig holds OpenTelemetry configuration
type OTELConfig struct {
	CollectorEndpoint string            `yaml:"collector_endpoint" validate:"required,url"`
	ServiceName       string            `yaml:"service_name" validate:"required"`
	Headers           map[string]string `yaml:"headers"`
	Insecure          bool              `yaml:"insecure"`
	BatchTimeout      Duration          `yaml:"batch_timeout"`
	BatchSize         int               `yaml:"batch_size" validate:"min=1,max=10000"`
}

// MetricsConfig holds configuration for all metric collectors
type MetricsConfig struct {
	EC2    CollectorConfig `yaml:"ec2"`
	RDS    CollectorConfig `yaml:"rds"`
	S3     CollectorConfig `yaml:"s3"`
	Lambda CollectorConfig `yaml:"lambda"`
	EBS    CollectorConfig `yaml:"ebs"`
	ELB    CollectorConfig `yaml:"elb"`
	VPC    CollectorConfig `yaml:"vpc"`
}

// CollectorConfig holds configuration for individual collectors
type CollectorConfig struct {
	Enabled            bool     `yaml:"enabled"`
	CollectionInterval Duration `yaml:"collection_interval"`
}

// GlobalConfig holds global application settings
type GlobalConfig struct {
	LogLevel             string   `yaml:"log_level" validate:"oneof=debug info warn error"`
	LogFormat            string   `yaml:"log_format" validate:"oneof=json text"`
	HealthCheckPort      int      `yaml:"health_check_port" validate:"min=1,max=65535"`
	HealthCheckPath      string   `yaml:"health_check_path"`
	DefaultInterval      Duration `yaml:"default_collection_interval"`
	MaxConcurrentWorkers int      `yaml:"max_concurrent_workers" validate:"min=1,max=100"`
	WorkerTimeout        Duration `yaml:"worker_timeout"`
	MaxErrorCount        int      `yaml:"max_error_count" validate:"min=1"`
	ErrorResetInterval   Duration `yaml:"error_reset_interval"`
	MetricBufferSize     int      `yaml:"metric_buffer_size" validate:"min=1"`
	ExportTimeout        Duration `yaml:"export_timeout"`
}

// Load loads configuration from the specified file path
func Load(configPath string) (*Config, error) {
	// Try to find config file if path is empty
	if configPath == "" {
		var err error
		configPath, err = findConfigFile()
		if err != nil {
			return nil, fmt.Errorf("config file not found: %w", err)
		}
	}

	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}

	// Set defaults
	setDefaults(&config)

	// Validate configuration
	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// findConfigFile searches for config file in standard locations
func findConfigFile() (string, error) {
	possiblePaths := []string{
		"./config.yaml",
		"./configs/config.yaml",
		"/etc/aws-monitor/config.yaml",
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("no config file found in standard locations: %v", possiblePaths)
}

// setDefaults sets default values for configuration fields
func setDefaults(config *Config) {
	// AWS defaults
	if config.AWS.MaxRetries == 0 {
		config.AWS.MaxRetries = 3
	}
	if config.AWS.Timeout == 0 {
		config.AWS.Timeout = Duration(30 * time.Second)
	}

	// OTEL defaults
	if config.OTEL.BatchTimeout == 0 {
		config.OTEL.BatchTimeout = Duration(5 * time.Second)
	}
	if config.OTEL.BatchSize == 0 {
		config.OTEL.BatchSize = 512
	}
	if config.OTEL.Headers == nil {
		config.OTEL.Headers = make(map[string]string)
	}

	// Global defaults
	if config.Global.LogLevel == "" {
		config.Global.LogLevel = "info"
	}
	if config.Global.LogFormat == "" {
		config.Global.LogFormat = "json"
	}
	if config.Global.HealthCheckPort == 0 {
		config.Global.HealthCheckPort = 8080
	}
	if config.Global.HealthCheckPath == "" {
		config.Global.HealthCheckPath = "/health"
	}
	if config.Global.DefaultInterval == 0 {
		config.Global.DefaultInterval = Duration(300 * time.Second) // 5 minutes
	}
	if config.Global.MaxConcurrentWorkers == 0 {
		config.Global.MaxConcurrentWorkers = 10
	}
	if config.Global.WorkerTimeout == 0 {
		config.Global.WorkerTimeout = Duration(60 * time.Second)
	}
	if config.Global.MaxErrorCount == 0 {
		config.Global.MaxErrorCount = 5
	}
	if config.Global.ErrorResetInterval == 0 {
		config.Global.ErrorResetInterval = Duration(300 * time.Second) // 5 minutes
	}
	if config.Global.MetricBufferSize == 0 {
		config.Global.MetricBufferSize = 1000
	}
	if config.Global.ExportTimeout == 0 {
		config.Global.ExportTimeout = Duration(30 * time.Second)
	}

	// Set default collection intervals for collectors
	defaultInterval := config.Global.DefaultInterval
	setCollectorDefaults(&config.Metrics.EC2, defaultInterval)
	setCollectorDefaults(&config.Metrics.RDS, defaultInterval)
	setCollectorDefaults(&config.Metrics.S3, Duration(600*time.Second)) // 10 minutes for S3
	setCollectorDefaults(&config.Metrics.Lambda, defaultInterval)
	setCollectorDefaults(&config.Metrics.EBS, defaultInterval)
	setCollectorDefaults(&config.Metrics.ELB, defaultInterval)
	setCollectorDefaults(&config.Metrics.VPC, Duration(600*time.Second)) // 10 minutes for VPC
}

// setCollectorDefaults sets default values for a collector
func setCollectorDefaults(collector *CollectorConfig, defaultInterval Duration) {
	if collector.CollectionInterval == 0 {
		collector.CollectionInterval = defaultInterval
	}
}

// validate validates the configuration using struct tags
func validate(config *Config) error {
	validator := validator.New()

	// Register custom validations
	registerCustomValidations(validator)

	if err := validator.Struct(config); err != nil {
		return formatValidationError(err)
	}

	// Custom validation logic
	return validateCustomRules(config)
}

// registerCustomValidations registers custom validation rules
func registerCustomValidations(_ *validator.Validate) {
	// Add custom validation for duration fields if needed
}

// validateCustomRules performs custom validation logic
func validateCustomRules(config *Config) error {
	// Validate enabled regions
	if len(config.EnabledRegions) == 0 {
		return fmt.Errorf("at least one region must be enabled")
	}

	// Validate AWS region format (basic check)
	for _, region := range config.EnabledRegions {
		if len(region) < 3 {
			return fmt.Errorf("invalid AWS region format: %s", region)
		}
	}

	// Validate default region is in enabled regions
	found := false
	for _, region := range config.EnabledRegions {
		if region == config.AWS.DefaultRegion {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("default region %s must be in enabled regions", config.AWS.DefaultRegion)
	}

	return nil
}

// formatValidationError formats validation errors into user-friendly messages
func formatValidationError(err error) error {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		var messages []string
		for _, fieldError := range validationErrors {
			message := fmt.Sprintf("field '%s' %s", fieldError.Field(), getValidationMessage(fieldError))
			messages = append(messages, message)
		}
		return fmt.Errorf("validation failed: %v", messages)
	}
	return err
}

// getValidationMessage returns a user-friendly validation message
func getValidationMessage(fieldError validator.FieldError) string {
	switch fieldError.Tag() {
	case "required":
		return "is required"
	case "min":
		return fmt.Sprintf("must be at least %s", fieldError.Param())
	case "max":
		return fmt.Sprintf("must be at most %s", fieldError.Param())
	case "url":
		return "must be a valid URL"
	case "oneof":
		return fmt.Sprintf("must be one of: %s", fieldError.Param())
	default:
		return fmt.Sprintf("failed validation: %s", fieldError.Tag())
	}
}

// GetCollectorConfig returns the configuration for a specific collector
func (c *Config) GetCollectorConfig(collectorName string) (CollectorConfig, error) {
	switch collectorName {
	case "ec2":
		return c.Metrics.EC2, nil
	case "rds":
		return c.Metrics.RDS, nil
	case "s3":
		return c.Metrics.S3, nil
	case "lambda":
		return c.Metrics.Lambda, nil
	case "ebs":
		return c.Metrics.EBS, nil
	case "elb":
		return c.Metrics.ELB, nil
	case "vpc":
		return c.Metrics.VPC, nil
	default:
		return CollectorConfig{}, fmt.Errorf("unknown collector: %s", collectorName)
	}
}

// Save saves the configuration to a file
func (c *Config) Save(configPath string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", dir, err)
	}

	// Write file with secure permissions
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", configPath, err)
	}

	return nil
}
