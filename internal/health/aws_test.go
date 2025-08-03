package health

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2"

	"aws-monitoring/internal/aws"
	"aws-monitoring/internal/config"
	"aws-monitoring/pkg/logger"
)

// mockClientProvider implements aws.ClientProvider for testing
type mockClientProvider struct {
	shouldFail bool
	clients    map[string]*mockHealthEC2Client
}

type mockHealthEC2Client struct {
	shouldFail bool
}

func (m *mockHealthEC2Client) DescribeInstances(_ context.Context, _ *ec2.DescribeInstancesInput, _ ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	if m.shouldFail {
		return nil, errors.New("mock AWS error")
	}
	return &ec2.DescribeInstancesOutput{}, nil
}

func (m *mockHealthEC2Client) DescribeInstanceStatus(_ context.Context, _ *ec2.DescribeInstanceStatusInput, _ ...func(*ec2.Options)) (*ec2.DescribeInstanceStatusOutput, error) {
	if m.shouldFail {
		return nil, errors.New("mock AWS error")
	}
	return &ec2.DescribeInstanceStatusOutput{}, nil
}

func (m *mockClientProvider) GetEC2Client(region string) (aws.EC2Client, error) {
	if m.shouldFail {
		return nil, errors.New("failed to create client")
	}
	
	if m.clients == nil {
		m.clients = make(map[string]*mockHealthEC2Client)
	}
	
	if client, exists := m.clients[region]; exists {
		return client, nil
	}
	
	client := &mockHealthEC2Client{shouldFail: false}
	m.clients[region] = client
	return client, nil
}

func (m *mockClientProvider) Close() error {
	return nil
}

func TestNewAWSChecker(t *testing.T) {
	cfg := &config.Config{
		EnabledRegions: []string{"us-east-1", "us-west-2"},
	}
	
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	mockProvider := &mockClientProvider{}
	checker := NewAWSChecker(mockProvider, cfg, log)
	
	if checker == nil {
		t.Fatal("Expected non-nil AWS checker")
	}
	
	if checker.Name() != "aws_connectivity" {
		t.Errorf("Expected name 'aws_connectivity', got %s", checker.Name())
	}
}

func TestAWSCheckerCheckNoRegions(t *testing.T) {
	cfg := &config.Config{
		EnabledRegions: []string{}, // No regions enabled
	}
	
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	mockProvider := &mockClientProvider{}
	checker := NewAWSChecker(mockProvider, cfg, log)
	
	ctx := context.Background()
	result := checker.Check(ctx)
	
	if result.Status != StatusDegraded {
		t.Errorf("Expected status degraded with no regions, got %s", result.Status)
	}
	
	if result.Message != "No AWS regions enabled" {
		t.Errorf("Expected 'No AWS regions enabled' message, got %s", result.Message)
	}
}

func TestAWSCheckerCheckAllHealthy(t *testing.T) {
	cfg := &config.Config{
		EnabledRegions: []string{"us-east-1", "us-west-2"},
	}
	
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	mockProvider := &mockClientProvider{shouldFail: false}
	checker := NewAWSChecker(mockProvider, cfg, log)
	
	ctx := context.Background()
	result := checker.Check(ctx)
	
	if result.Status != StatusHealthy {
		t.Errorf("Expected status healthy with all regions accessible, got %s", result.Status)
	}
	
	if result.Metadata["healthy_regions"] != 2 {
		t.Errorf("Expected 2 healthy regions, got %v", result.Metadata["healthy_regions"])
	}
	
	if result.Metadata["total_regions"] != 2 {
		t.Errorf("Expected 2 total regions, got %v", result.Metadata["total_regions"])
	}
}

func TestAWSCheckerCheckPartialFailure(t *testing.T) {
	cfg := &config.Config{
		EnabledRegions: []string{"us-east-1", "us-west-2"},
	}
	
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Create a mock provider that will fail for one region
	mockProvider := &mockClientProvider{
		clients: map[string]*mockHealthEC2Client{
			"us-east-1": {shouldFail: false},
			"us-west-2": {shouldFail: true},
		},
	}
	
	checker := NewAWSChecker(mockProvider, cfg, log)
	
	ctx := context.Background()
	result := checker.Check(ctx)
	
	if result.Status != StatusDegraded {
		t.Errorf("Expected status degraded with partial failure, got %s", result.Status)
	}
	
	if result.Metadata["healthy_regions"] != 1 {
		t.Errorf("Expected 1 healthy region, got %v", result.Metadata["healthy_regions"])
	}
}

func TestNewBasicChecker(t *testing.T) {
	checker := NewBasicChecker("test-service", "1.0.0")
	
	if checker == nil {
		t.Fatal("Expected non-nil basic checker")
	}
	
	if checker.Name() != "basic" {
		t.Errorf("Expected name 'basic', got %s", checker.Name())
	}
	
	ctx := context.Background()
	result := checker.Check(ctx)
	
	if result.Status != StatusHealthy {
		t.Errorf("Expected status healthy, got %s", result.Status)
	}
	
	if result.Metadata["service"] != "test-service" {
		t.Errorf("Expected service 'test-service', got %v", result.Metadata["service"])
	}
	
	if result.Metadata["version"] != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %v", result.Metadata["version"])
	}
}

func TestNewConfigChecker(t *testing.T) {
	cfg := &config.Config{
		EnabledRegions: []string{"us-east-1"},
		OTEL: config.OTELConfig{
			CollectorEndpoint: "http://localhost:4317",
			ServiceName:       "test-service",
		},
		Metrics: config.MetricsConfig{
			EC2: config.CollectorConfig{Enabled: true},
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

	checker := NewConfigChecker(cfg, log)
	
	if checker == nil {
		t.Fatal("Expected non-nil config checker")
	}
	
	if checker.Name() != "configuration" {
		t.Errorf("Expected name 'configuration', got %s", checker.Name())
	}
}

func TestConfigCheckerCheckValid(t *testing.T) {
	cfg := &config.Config{
		EnabledRegions: []string{"us-east-1"},
		OTEL: config.OTELConfig{
			CollectorEndpoint: "http://localhost:4317",
			ServiceName:       "test-service",
		},
		Metrics: config.MetricsConfig{
			EC2: config.CollectorConfig{Enabled: true},
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

	checker := NewConfigChecker(cfg, log)
	
	ctx := context.Background()
	result := checker.Check(ctx)
	
	if result.Status != StatusHealthy {
		t.Errorf("Expected status healthy with valid config, got %s", result.Status)
	}
	
	if result.Message != "Configuration is valid" {
		t.Errorf("Expected 'Configuration is valid' message, got %s", result.Message)
	}
}

func TestConfigCheckerCheckInvalid(t *testing.T) {
	cfg := &config.Config{
		EnabledRegions: []string{}, // No regions
		OTEL: config.OTELConfig{
			CollectorEndpoint: "", // Empty endpoint
			ServiceName:       "", // Empty service name
		},
		Metrics: config.MetricsConfig{
			// No collectors enabled
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

	checker := NewConfigChecker(cfg, log)
	
	ctx := context.Background()
	result := checker.Check(ctx)
	
	if result.Status != StatusUnhealthy {
		t.Errorf("Expected status unhealthy with invalid config, got %s", result.Status)
	}
	
	if result.Error == "" {
		t.Error("Expected error message with invalid config")
	}
}

func TestGetEnabledCollectors(t *testing.T) {
	cfg := &config.Config{
		Metrics: config.MetricsConfig{
			EC2:    config.CollectorConfig{Enabled: true},
			RDS:    config.CollectorConfig{Enabled: false},
			S3:     config.CollectorConfig{Enabled: true},
			Lambda: config.CollectorConfig{Enabled: false},
			EBS:    config.CollectorConfig{Enabled: true},
			ELB:    config.CollectorConfig{Enabled: false},
			VPC:    config.CollectorConfig{Enabled: true},
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

	checker := NewConfigChecker(cfg, log)
	enabled := checker.getEnabledCollectors()
	
	expected := []string{"ec2", "s3", "ebs", "vpc"}
	if len(enabled) != len(expected) {
		t.Errorf("Expected %d enabled collectors, got %d", len(expected), len(enabled))
	}
	
	for _, exp := range expected {
		found := false
		for _, en := range enabled {
			if en == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected collector %s to be enabled", exp)
		}
	}
}