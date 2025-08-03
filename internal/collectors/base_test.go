package collectors

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"aws-monitoring/internal/aws"
	"aws-monitoring/internal/config"
	"aws-monitoring/pkg/errors"
	"aws-monitoring/pkg/logger"
)

// mockAWSProvider for testing
type mockAWSProvider struct{}

func (m *mockAWSProvider) GetEC2Client(_ string) (aws.EC2Client, error) {
	return &mockCollectorEC2Client{}, nil
}

func (m *mockAWSProvider) Close() error {
	return nil
}

type mockCollectorEC2Client struct{}

func (m *mockCollectorEC2Client) DescribeInstances(_ context.Context, _ *ec2.DescribeInstancesInput, _ ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	return &ec2.DescribeInstancesOutput{}, nil
}

func (m *mockCollectorEC2Client) DescribeInstanceStatus(_ context.Context, _ *ec2.DescribeInstanceStatusInput, _ ...func(*ec2.Options)) (*ec2.DescribeInstanceStatusOutput, error) {
	return &ec2.DescribeInstanceStatusOutput{}, nil
}

func TestNewBaseCollector(t *testing.T) {
	cfg := &config.Config{
		EnabledRegions: []string{"us-east-1", "us-west-2"},
	}
	
	collectorConfig := DefaultCollectorConfig()
	awsProvider := &mockAWSProvider{}
	
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	
	bc := NewBaseCollector("test-collector", "Test collector", cfg, collectorConfig, awsProvider, log)
	
	if bc.Name() != "test-collector" {
		t.Errorf("Expected name 'test-collector', got %s", bc.Name())
	}
	
	if bc.Description() != "Test collector" {
		t.Errorf("Expected description 'Test collector', got %s", bc.Description())
	}
	
	if bc.status != StatusStopped {
		t.Errorf("Expected initial status stopped, got %s", bc.status)
	}
}

func TestBaseCollectorStartStop(t *testing.T) {
	cfg := &config.Config{
		EnabledRegions: []string{"us-east-1"},
	}
	
	collectorConfig := DefaultCollectorConfig()
	awsProvider := &mockAWSProvider{}
	
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	
	bc := NewBaseCollector("test-collector", "Test collector", cfg, collectorConfig, awsProvider, log)
	
	ctx := context.Background()
	
	// Test start
	err = bc.Start(ctx)
	if err != nil {
		t.Errorf("Expected no error on start, got: %v", err)
	}
	
	if bc.status != StatusRunning {
		t.Errorf("Expected status running after start, got %s", bc.status)
	}
	
	// Test start when already running
	err = bc.Start(ctx)
	if err != nil {
		t.Errorf("Expected no error when starting already running collector, got: %v", err)
	}
	
	// Test stop
	err = bc.Stop(ctx)
	if err != nil {
		t.Errorf("Expected no error on stop, got: %v", err)
	}
	
	if bc.status != StatusStopped {
		t.Errorf("Expected status stopped after stop, got %s", bc.status)
	}
	
	// Test stop when already stopped
	err = bc.Stop(ctx)
	if err != nil {
		t.Errorf("Expected no error when stopping already stopped collector, got: %v", err)
	}
}

func TestBaseCollectorValidateConfig(t *testing.T) {
	cfg := &config.Config{
		EnabledRegions: []string{"us-east-1"},
	}
	
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	
	awsProvider := &mockAWSProvider{}
	
	tests := []struct {
		name           string
		config         CollectorConfig
		expectError    bool
		expectedErrCode string
	}{
		{
			name:        "valid config",
			config:      DefaultCollectorConfig(),
			expectError: false,
		},
		{
			name: "invalid interval",
			config: CollectorConfig{
				Enabled:    true,
				Interval:   0,
				Timeout:    30 * time.Second,
				Retries:    3,
				RetryDelay: time.Second,
			},
			expectError:     true,
			expectedErrCode: "INVALID_INTERVAL",
		},
		{
			name: "invalid timeout",
			config: CollectorConfig{
				Enabled:    true,
				Interval:   5 * time.Minute,
				Timeout:    0,
				Retries:    3,
				RetryDelay: time.Second,
			},
			expectError:     true,
			expectedErrCode: "INVALID_TIMEOUT",
		},
		{
			name: "invalid retries",
			config: CollectorConfig{
				Enabled:    true,
				Interval:   5 * time.Minute,
				Timeout:    30 * time.Second,
				Retries:    -1,
				RetryDelay: time.Second,
			},
			expectError:     true,
			expectedErrCode: "INVALID_RETRIES",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bc := NewBaseCollector("test", "test", cfg, tt.config, awsProvider, log)
			err := bc.validateConfig()
			
			if tt.expectError {
				if err == nil {
					t.Error("Expected validation error, got nil")
				} else if err.Code != tt.expectedErrCode {
					t.Errorf("Expected error code %s, got %s", tt.expectedErrCode, err.Code)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no validation error, got: %v", err)
				}
			}
		})
	}
}

func TestBaseCollectorNoRegions(t *testing.T) {
	cfg := &config.Config{
		EnabledRegions: []string{}, // No regions
	}
	
	collectorConfig := DefaultCollectorConfig()
	awsProvider := &mockAWSProvider{}
	
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	
	bc := NewBaseCollector("test", "test", cfg, collectorConfig, awsProvider, log)
	
	ctx := context.Background()
	err = bc.Start(ctx)
	
	if err == nil {
		t.Error("Expected error when starting with no regions")
	}
	
	if bc.status != StatusError {
		t.Errorf("Expected status error, got %s", bc.status)
	}
}

func TestBaseCollectorCreateMetric(t *testing.T) {
	cfg := &config.Config{
		EnabledRegions: []string{"us-east-1"},
	}
	
	collectorConfig := DefaultCollectorConfig()
	collectorConfig.CustomTags = map[string]string{
		"environment": "test",
		"team":        "platform",
	}
	
	awsProvider := &mockAWSProvider{}
	
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	
	bc := NewBaseCollector("test-collector", "test", cfg, collectorConfig, awsProvider, log)
	
	// Test basic metric creation
	metric := bc.CreateMetric("test_metric", 42.5, "Count", map[string]string{
		"region":   "us-east-1",
		"instance": "i-1234567890abcdef0",
	})
	
	if metric.Name != "test_metric" {
		t.Errorf("Expected name 'test_metric', got %s", metric.Name)
	}
	
	if metric.Value != 42.5 {
		t.Errorf("Expected value 42.5, got %f", metric.Value)
	}
	
	if metric.Unit != "Count" {
		t.Errorf("Expected unit 'Count', got %s", metric.Unit)
	}
	
	// Check that common labels are added
	expectedLabels := map[string]string{
		"collector":   "test-collector",
		"service":     "aws-monitor",
		"environment": "test",
		"team":        "platform",
		"region":      "us-east-1",
		"instance":    "i-1234567890abcdef0",
	}
	
	for key, expectedValue := range expectedLabels {
		if actualValue, exists := metric.Labels[key]; !exists {
			t.Errorf("Expected label %s to exist", key)
		} else if actualValue != expectedValue {
			t.Errorf("Expected label %s to be %s, got %s", key, expectedValue, actualValue)
		}
	}
	
	// Test metric with description
	metricWithDesc := bc.CreateMetricWithDescription("test_metric_desc", 100, "Bytes", "Test metric with description", nil)
	if metricWithDesc.Description != "Test metric with description" {
		t.Errorf("Expected description 'Test metric with description', got %s", metricWithDesc.Description)
	}
}

func TestBaseCollectorCollectWithRetry(t *testing.T) {
	cfg := &config.Config{
		EnabledRegions: []string{"us-east-1"},
	}
	
	collectorConfig := DefaultCollectorConfig()
	collectorConfig.Retries = 2
	collectorConfig.RetryDelay = 10 * time.Millisecond
	
	awsProvider := &mockAWSProvider{}
	
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	
	bc := NewBaseCollector("test-collector", "test", cfg, collectorConfig, awsProvider, log)
	
	ctx := context.Background()
	
	// Test successful collection
	successFunc := func(_ context.Context, _ string) ([]MetricData, error) {
		return []MetricData{
			bc.CreateMetric("success_metric", 1, "Count", nil),
		}, nil
	}
	
	result := bc.CollectWithRetry(ctx, "us-east-1", successFunc)
	
	if result.Error != nil {
		t.Errorf("Expected no error for successful collection, got: %v", result.Error)
	}
	
	if len(result.Metrics) != 1 {
		t.Errorf("Expected 1 metric, got %d", len(result.Metrics))
	}
	
	if result.CollectorName != "test-collector" {
		t.Errorf("Expected collector name 'test-collector', got %s", result.CollectorName)
	}
	
	if result.Region != "us-east-1" {
		t.Errorf("Expected region 'us-east-1', got %s", result.Region)
	}
	
	// Test collection with retryable error
	attemptCount := 0
	retryableErrorFunc := func(_ context.Context, _ string) ([]MetricData, error) {
		attemptCount++
		if attemptCount <= 2 {
			return nil, errors.NewNetworkError("CONNECTION_ERROR", "connection failed")
		}
		return []MetricData{
			bc.CreateMetric("retry_success_metric", 2, "Count", nil),
		}, nil
	}
	
	attemptCount = 0 // Reset counter
	result = bc.CollectWithRetry(ctx, "us-east-1", retryableErrorFunc)
	
	if result.Error != nil {
		t.Errorf("Expected no error after retries, got: %v", result.Error)
	}
	
	if len(result.Metrics) != 1 {
		t.Errorf("Expected 1 metric after retries, got %d", len(result.Metrics))
	}
	
	// Test collection with non-retryable error
	nonRetryableErrorFunc := func(_ context.Context, _ string) ([]MetricData, error) {
		return nil, errors.NewPermissionError("describe", "ec2:instances")
	}
	
	result = bc.CollectWithRetry(ctx, "us-east-1", nonRetryableErrorFunc)
	
	if result.Error == nil {
		t.Error("Expected error for non-retryable failure")
	}
	
	if len(result.Metrics) != 0 {
		t.Errorf("Expected no metrics for failed collection, got %d", len(result.Metrics))
	}
}

func TestBaseCollectorInfo(t *testing.T) {
	cfg := &config.Config{
		EnabledRegions: []string{"us-east-1", "us-west-2"},
	}
	
	collectorConfig := DefaultCollectorConfig()
	awsProvider := &mockAWSProvider{}
	
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	
	bc := NewBaseCollector("test-collector", "Test collector description", cfg, collectorConfig, awsProvider, log)
	
	info := bc.Info()
	
	if info.Name != "test-collector" {
		t.Errorf("Expected name 'test-collector', got %s", info.Name)
	}
	
	if info.Description != "Test collector description" {
		t.Errorf("Expected description 'Test collector description', got %s", info.Description)
	}
	
	if info.Status != StatusStopped {
		t.Errorf("Expected status stopped, got %s", info.Status)
	}
	
	if len(info.EnabledRegions) != 2 {
		t.Errorf("Expected 2 enabled regions, got %d", len(info.EnabledRegions))
	}
	
	if info.Interval != collectorConfig.Interval {
		t.Errorf("Expected interval %v, got %v", collectorConfig.Interval, info.Interval)
	}
}

func TestBaseCollectorHealth(t *testing.T) {
	cfg := &config.Config{
		EnabledRegions: []string{"us-east-1"},
	}
	
	collectorConfig := DefaultCollectorConfig()
	awsProvider := &mockAWSProvider{}
	
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	
	bc := NewBaseCollector("test-collector", "test", cfg, collectorConfig, awsProvider, log)
	
	// Test health when stopped
	health := bc.Health()
	if health == nil {
		t.Error("Expected health error when collector is stopped")
	}
	
	// Start the collector
	ctx := context.Background()
	err = bc.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start collector: %v", err)
	}
	
	// Test health when running but no collections yet
	health = bc.Health()
	if health != nil {
		t.Errorf("Expected no health error for newly started collector, got: %v", health)
	}
	
	// Simulate a recent collection
	now := time.Now()
	bc.lastCollection = &now
	bc.successfulCollections = 1
	
	health = bc.Health()
	if health != nil {
		t.Errorf("Expected no health error with recent collection, got: %v", health)
	}
}

func TestDefaultCollectorConfig(t *testing.T) {
	config := DefaultCollectorConfig()
	
	if !config.Enabled {
		t.Error("Expected default config to be enabled")
	}
	
	if config.Interval != 5*time.Minute {
		t.Errorf("Expected default interval to be 5m, got %v", config.Interval)
	}
	
	if config.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout to be 30s, got %v", config.Timeout)
	}
	
	if config.Retries != 3 {
		t.Errorf("Expected default retries to be 3, got %d", config.Retries)
	}
	
	if config.RetryDelay != 10*time.Second {
		t.Errorf("Expected default retry delay to be 10s, got %v", config.RetryDelay)
	}
	
	if config.CustomTags == nil {
		t.Error("Expected custom tags map to be initialized")
	}
}