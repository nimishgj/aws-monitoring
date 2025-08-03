package scheduler

import (
	"context"
	"testing"
	"time"

	"aws-monitoring/internal/collectors"
	"aws-monitoring/pkg/errors"
	"aws-monitoring/pkg/logger"
)

func TestDefaultJobExecutor(t *testing.T) {
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, _ := logger.NewLogger(loggerConfig)
	
	registry := newMockRegistry()
	executor := NewDefaultJobExecutor(registry, log)
	
	// Test with non-existent collector
	job := &ScheduledJob{
		ID:            "test-job",
		CollectorName: "non-existent",
		Region:        "us-east-1",
		Interval:      5 * time.Minute,
		NextRun:       time.Now(),
		Enabled:       true,
	}
	
	ctx := context.Background()
	result := executor.ExecuteJob(ctx, job)
	
	if result.Error == nil {
		t.Error("Expected error for non-existent collector")
	}
	
	if result.Error.Code != "COLLECTOR_NOT_FOUND" {
		t.Errorf("Expected error code COLLECTOR_NOT_FOUND, got %s", result.Error.Code)
	}
	
	// Test with existing collector
	collector := &mockCollector{
		name:        "test-collector",
		description: "Test collector",
	}
	err := registry.Register(collector)
	if err != nil {
		t.Fatalf("Failed to register collector: %v", err)
	}
	
	job.CollectorName = "test-collector"
	result = executor.ExecuteJob(ctx, job)
	
	if result.Error != nil {
		t.Errorf("Expected no error for existing collector, got: %v", result.Error)
	}
	
	if result.CollectorName != "test-collector" {
		t.Errorf("Expected collector name 'test-collector', got %s", result.CollectorName)
	}
	
	if result.Region != "us-east-1" {
		t.Errorf("Expected region 'us-east-1', got %s", result.Region)
	}
	
	if len(result.Metrics) != 1 {
		t.Errorf("Expected 1 metric, got %d", len(result.Metrics))
	}
}

func TestDefaultJobProcessor(t *testing.T) {
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, _ := logger.NewLogger(loggerConfig)
	
	processor := NewDefaultJobProcessor(log)
	ctx := context.Background()
	
	job := &ScheduledJob{
		ID:            "test-job",
		CollectorName: "test-collector",
		Region:        "us-east-1",
		Interval:      5 * time.Minute,
		NextRun:       time.Now(),
		Enabled:       true,
	}
	
	// Test ProcessResult
	result := &collectors.CollectionResult{
		CollectorName:  "test-collector",
		Region:         "us-east-1",
		CollectionTime: time.Now(),
		Metrics: []collectors.MetricData{
			{
				Name:      "test_metric",
				Value:     1.0,
				Unit:      "Count",
				Timestamp: time.Now(),
				Labels:    map[string]string{"region": "us-east-1"},
			},
		},
		Duration: 100 * time.Millisecond,
	}
	
	err := processor.ProcessResult(ctx, job, result)
	if err != nil {
		t.Errorf("Expected no error processing result, got: %v", err)
	}
	
	// Test ProcessError
	collectionError := errors.NewNetworkError("CONNECTION_ERROR", "connection failed")
	err = processor.ProcessError(ctx, job, collectionError)
	if err != nil {
		t.Errorf("Expected no error processing error, got: %v", err)
	}
}