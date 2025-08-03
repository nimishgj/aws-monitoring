package health

import (
	"testing"
	"time"
)

func TestDefaultCheckerConfig(t *testing.T) {
	config := DefaultCheckerConfig()
	
	if !config.Enabled {
		t.Error("Expected default config to be enabled")
	}
	
	if config.Interval != 30*time.Second {
		t.Errorf("Expected default interval to be 30s, got %v", config.Interval)
	}
	
	if config.Timeout != 10*time.Second {
		t.Errorf("Expected default timeout to be 10s, got %v", config.Timeout)
	}
	
	if config.Retries != 2 {
		t.Errorf("Expected default retries to be 2, got %d", config.Retries)
	}
	
	if config.RetryDelay != time.Second {
		t.Errorf("Expected default retry delay to be 1s, got %v", config.RetryDelay)
	}
}

func TestStatusConstants(t *testing.T) {
	// Test that status constants are properly defined
	statuses := []Status{StatusHealthy, StatusUnhealthy, StatusDegraded, StatusUnknown}
	expectedValues := []string{"healthy", "unhealthy", "degraded", "unknown"}
	
	for i, status := range statuses {
		if string(status) != expectedValues[i] {
			t.Errorf("Expected status %d to be %s, got %s", i, expectedValues[i], string(status))
		}
	}
}

func TestCheckResult(t *testing.T) {
	now := time.Now()
	duration := 100 * time.Millisecond
	
	result := CheckResult{
		Name:        "test-check",
		Status:      StatusHealthy,
		Message:     "All good",
		LastChecked: now,
		Duration:    duration,
		Error:       "",
		Metadata:    map[string]interface{}{"test": "value"},
	}
	
	if result.Name != "test-check" {
		t.Errorf("Expected name to be 'test-check', got %s", result.Name)
	}
	
	if result.Status != StatusHealthy {
		t.Errorf("Expected status to be healthy, got %s", result.Status)
	}
	
	if result.Message != "All good" {
		t.Errorf("Expected message to be 'All good', got %s", result.Message)
	}
	
	if !result.LastChecked.Equal(now) {
		t.Errorf("Expected last checked to be %v, got %v", now, result.LastChecked)
	}
	
	if result.Duration != duration {
		t.Errorf("Expected duration to be %v, got %v", duration, result.Duration)
	}
	
	if result.Metadata["test"] != "value" {
		t.Errorf("Expected metadata test value to be 'value', got %v", result.Metadata["test"])
	}
}

func TestOverallHealth(t *testing.T) {
	now := time.Now()
	uptime := 5 * time.Minute
	
	health := OverallHealth{
		Status:      StatusHealthy,
		Timestamp:   now,
		Uptime:      uptime,
		Version:     "1.0.0",
		ServiceName: "test-service",
		Checks:      make(map[string]CheckResult),
		Summary:     "All systems operational",
	}
	
	if health.Status != StatusHealthy {
		t.Errorf("Expected status to be healthy, got %s", health.Status)
	}
	
	if !health.Timestamp.Equal(now) {
		t.Errorf("Expected timestamp to be %v, got %v", now, health.Timestamp)
	}
	
	if health.Uptime != uptime {
		t.Errorf("Expected uptime to be %v, got %v", uptime, health.Uptime)
	}
	
	if health.Version != "1.0.0" {
		t.Errorf("Expected version to be '1.0.0', got %s", health.Version)
	}
	
	if health.ServiceName != "test-service" {
		t.Errorf("Expected service name to be 'test-service', got %s", health.ServiceName)
	}
	
	if health.Summary != "All systems operational" {
		t.Errorf("Expected summary to be 'All systems operational', got %s", health.Summary)
	}
}