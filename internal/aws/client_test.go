package aws

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"aws-monitoring/internal/config"
	"aws-monitoring/pkg/logger"
)

// mockEC2Client implements EC2Client for testing
type mockEC2Client struct {
	describeInstancesFunc       func(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	describeInstanceStatusFunc  func(ctx context.Context, params *ec2.DescribeInstanceStatusInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceStatusOutput, error)
}

func (m *mockEC2Client) DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	if m.describeInstancesFunc != nil {
		return m.describeInstancesFunc(ctx, params, optFns...)
	}
	return &ec2.DescribeInstancesOutput{}, nil
}

func (m *mockEC2Client) DescribeInstanceStatus(ctx context.Context, params *ec2.DescribeInstanceStatusInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceStatusOutput, error) {
	if m.describeInstanceStatusFunc != nil {
		return m.describeInstanceStatusFunc(ctx, params, optFns...)
	}
	return &ec2.DescribeInstanceStatusOutput{}, nil
}

func TestNewClientProvider(t *testing.T) {
	cfg := &config.Config{
		AWS: config.AWSConfig{
			AccessKeyID:     "test-key",
			SecretAccessKey: "test-secret",
			DefaultRegion:   "us-east-1",
			MaxRetries:      3,
			Timeout:         config.Duration(30 * time.Second),
		},
	}

	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	provider := NewClientProvider(cfg, log)
	if provider == nil {
		t.Fatal("Expected non-nil client provider")
	}

	// Test that we can cast to our implementation
	cp, ok := provider.(*clientProvider)
	if !ok {
		t.Fatal("Expected clientProvider implementation")
	}

	if cp.config != cfg {
		t.Error("Config not properly set")
	}

	if cp.logger == nil {
		t.Error("Logger not properly set")
	}

	if cp.awsConfigs == nil {
		t.Error("AWS configs map not initialized")
	}
}

func TestClientProvider_Close(t *testing.T) {
	cfg := &config.Config{
		AWS: config.AWSConfig{
			DefaultRegion: "us-east-1",
			MaxRetries:    3,
			Timeout:       config.Duration(30 * time.Second),
		},
	}

	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	provider := NewClientProvider(cfg, log)
	
	err = provider.Close()
	if err != nil {
		t.Errorf("Expected no error on close, got: %v", err)
	}

	// Verify internal state is cleaned up
	cp := provider.(*clientProvider)
	if len(cp.awsConfigs) != 0 {
		t.Error("Expected AWS configs to be cleared after close")
	}
}

func TestClientProvider_GetEC2Client_WithCredentials(t *testing.T) {
	cfg := &config.Config{
		AWS: config.AWSConfig{
			AccessKeyID:     "test-access-key",
			SecretAccessKey: "test-secret-key",
			DefaultRegion:   "us-east-1",
			MaxRetries:      5,
			Timeout:         config.Duration(45 * time.Second),
		},
	}

	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	provider := NewClientProvider(cfg, log)
	
	// This will fail in actual AWS calls but should succeed in creating the client
	client, err := provider.GetEC2Client("us-west-2")
	if err != nil {
		t.Errorf("Expected no error getting EC2 client, got: %v", err)
	}

	if client == nil {
		t.Fatal("Expected non-nil EC2 client")
	}
}

func TestClientProvider_GetEC2Client_WithoutCredentials(t *testing.T) {
	cfg := &config.Config{
		AWS: config.AWSConfig{
			// No explicit credentials - should use default credential chain
			DefaultRegion: "us-east-1",
			MaxRetries:    3,
			Timeout:       config.Duration(30 * time.Second),
		},
	}

	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	provider := NewClientProvider(cfg, log)
	
	// This should still create a client even without explicit credentials
	client, err := provider.GetEC2Client("eu-west-1")
	if err != nil {
		t.Errorf("Expected no error getting EC2 client, got: %v", err)
	}

	if client == nil {
		t.Fatal("Expected non-nil EC2 client")
	}
}

func TestClientProvider_ConfigCaching(t *testing.T) {
	cfg := &config.Config{
		AWS: config.AWSConfig{
			AccessKeyID:     "test-key",
			SecretAccessKey: "test-secret",
			DefaultRegion:   "us-east-1",
			MaxRetries:      3,
			Timeout:         config.Duration(30 * time.Second),
		},
	}

	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	provider := NewClientProvider(cfg, log)
	cp := provider.(*clientProvider)

	region := "us-west-1"

	// First call should create and cache the config
	_, err = provider.GetEC2Client(region)
	if err != nil {
		t.Errorf("First call failed: %v", err)
	}

	if len(cp.awsConfigs) != 1 {
		t.Errorf("Expected 1 cached config, got %d", len(cp.awsConfigs))
	}

	// Second call should use cached config
	_, err = provider.GetEC2Client(region)
	if err != nil {
		t.Errorf("Second call failed: %v", err)
	}

	if len(cp.awsConfigs) != 1 {
		t.Errorf("Expected 1 cached config after second call, got %d", len(cp.awsConfigs))
	}

	// Different region should create new config
	_, err = provider.GetEC2Client("eu-central-1")
	if err != nil {
		t.Errorf("Third call with different region failed: %v", err)
	}

	if len(cp.awsConfigs) != 2 {
		t.Errorf("Expected 2 cached configs for different regions, got %d", len(cp.awsConfigs))
	}
}

func TestEC2ClientInterface(t *testing.T) {
	// Test that our mock implements the interface
	var client EC2Client = &mockEC2Client{}

	ctx := context.Background()

	// Test DescribeInstances
	_, err := client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
	if err != nil {
		t.Errorf("DescribeInstances failed: %v", err)
	}

	// Test DescribeInstanceStatus
	_, err = client.DescribeInstanceStatus(ctx, &ec2.DescribeInstanceStatusInput{})
	if err != nil {
		t.Errorf("DescribeInstanceStatus failed: %v", err)
	}
}

func TestClientProvider_MultipleRegions(t *testing.T) {
	cfg := &config.Config{
		EnabledRegions: []string{"us-east-1", "us-west-2", "eu-west-1"},
		AWS: config.AWSConfig{
			AccessKeyID:     "test-key",
			SecretAccessKey: "test-secret",
			DefaultRegion:   "us-east-1",
			MaxRetries:      3,
			Timeout:         config.Duration(30 * time.Second),
		},
	}

	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	provider := NewClientProvider(cfg, log)

	// Test creating clients for multiple regions
	for _, region := range cfg.EnabledRegions {
		client, err := provider.GetEC2Client(region)
		if err != nil {
			t.Errorf("Failed to get EC2 client for region %s: %v", region, err)
		}
		if client == nil {
			t.Errorf("Got nil client for region %s", region)
		}
	}

	// Verify all regions are cached
	cp := provider.(*clientProvider)
	if len(cp.awsConfigs) != len(cfg.EnabledRegions) {
		t.Errorf("Expected %d cached configs, got %d", len(cfg.EnabledRegions), len(cp.awsConfigs))
	}
}