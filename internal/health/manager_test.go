package health

import (
	"context"
	"testing"
	"time"

	"aws-monitoring/pkg/logger"
)

// mockChecker implements the Checker interface for testing
type mockChecker struct {
	name   string
	result CheckResult
}

func (m *mockChecker) Name() string {
	return m.name
}

func (m *mockChecker) Check(_ context.Context) CheckResult {
	return m.result
}

func newMockChecker(name string, status Status, message string) *mockChecker {
	return &mockChecker{
		name: name,
		result: CheckResult{
			Name:        name,
			Status:      status,
			Message:     message,
			LastChecked: time.Now(),
			Duration:    10 * time.Millisecond,
		},
	}
}

func TestNewManager(t *testing.T) {
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	manager := NewManager("test-service", "1.0.0", log)
	
	if manager == nil {
		t.Fatal("Expected non-nil manager")
	}
	
	if manager.service != "test-service" {
		t.Errorf("Expected service to be 'test-service', got %s", manager.service)
	}
	
	if manager.version != "1.0.0" {
		t.Errorf("Expected version to be '1.0.0', got %s", manager.version)
	}
	
	if len(manager.checkers) != 0 {
		t.Error("Expected empty checkers map")
	}
	
	if len(manager.results) != 0 {
		t.Error("Expected empty results map")
	}
}

func TestManagerRegisterChecker(t *testing.T) {
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	manager := NewManager("test-service", "1.0.0", log)
	checker := newMockChecker("test-checker", StatusHealthy, "Test healthy")
	
	manager.RegisterChecker(checker)
	
	if len(manager.checkers) != 1 {
		t.Errorf("Expected 1 checker, got %d", len(manager.checkers))
	}
	
	if manager.checkers["test-checker"] != checker {
		t.Error("Checker not properly registered")
	}
}

func TestManagerUnregisterChecker(t *testing.T) {
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	manager := NewManager("test-service", "1.0.0", log)
	checker := newMockChecker("test-checker", StatusHealthy, "Test healthy")
	
	manager.RegisterChecker(checker)
	manager.UnregisterChecker("test-checker")
	
	if len(manager.checkers) != 0 {
		t.Errorf("Expected 0 checkers after unregister, got %d", len(manager.checkers))
	}
	
	// Test unregistering non-existent checker
	manager.UnregisterChecker("non-existent")
	// Should not panic
}

func TestManagerRunChecks(t *testing.T) {
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	manager := NewManager("test-service", "1.0.0", log)
	
	// Register multiple checkers
	checker1 := newMockChecker("checker1", StatusHealthy, "Healthy 1")
	checker2 := newMockChecker("checker2", StatusDegraded, "Degraded 2")
	
	manager.RegisterChecker(checker1)
	manager.RegisterChecker(checker2)
	
	ctx := context.Background()
	manager.RunChecks(ctx)
	
	// Give some time for concurrent checks to complete
	time.Sleep(100 * time.Millisecond)
	
	if len(manager.results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(manager.results))
	}
	
	result1 := manager.results["checker1"]
	if result1.Status != StatusHealthy {
		t.Errorf("Expected checker1 to be healthy, got %s", result1.Status)
	}
	
	result2 := manager.results["checker2"]
	if result2.Status != StatusDegraded {
		t.Errorf("Expected checker2 to be degraded, got %s", result2.Status)
	}
}

func TestManagerGetHealth(t *testing.T) {
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	manager := NewManager("test-service", "1.0.0", log)
	
	// Test with no checkers
	health := manager.GetHealth()
	if health.Status != StatusUnknown {
		t.Errorf("Expected unknown status with no checkers, got %s", health.Status)
	}
	
	// Add checkers and run checks
	checker1 := newMockChecker("checker1", StatusHealthy, "Healthy 1")
	checker2 := newMockChecker("checker2", StatusHealthy, "Healthy 2")
	
	manager.RegisterChecker(checker1)
	manager.RegisterChecker(checker2)
	manager.RunChecks(context.Background())
	
	time.Sleep(100 * time.Millisecond)
	
	health = manager.GetHealth()
	if health.Status != StatusHealthy {
		t.Errorf("Expected healthy status with all healthy checkers, got %s", health.Status)
	}
	
	if health.ServiceName != "test-service" {
		t.Errorf("Expected service name 'test-service', got %s", health.ServiceName)
	}
	
	if health.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got %s", health.Version)
	}
	
	if len(health.Checks) != 2 {
		t.Errorf("Expected 2 checks in health response, got %d", len(health.Checks))
	}
}

func TestAggregateStatus(t *testing.T) {
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	manager := NewManager("test-service", "1.0.0", log)
	
	tests := []struct {
		name           string
		checks         map[string]CheckResult
		expectedStatus Status
	}{
		{
			name:           "no checks",
			checks:         map[string]CheckResult{},
			expectedStatus: StatusUnknown,
		},
		{
			name: "all healthy",
			checks: map[string]CheckResult{
				"check1": {Status: StatusHealthy},
				"check2": {Status: StatusHealthy},
			},
			expectedStatus: StatusHealthy,
		},
		{
			name: "one unhealthy",
			checks: map[string]CheckResult{
				"check1": {Status: StatusHealthy},
				"check2": {Status: StatusUnhealthy},
			},
			expectedStatus: StatusUnhealthy,
		},
		{
			name: "one degraded",
			checks: map[string]CheckResult{
				"check1": {Status: StatusHealthy},
				"check2": {Status: StatusDegraded},
			},
			expectedStatus: StatusDegraded,
		},
		{
			name: "all unknown",
			checks: map[string]CheckResult{
				"check1": {Status: StatusUnknown},
				"check2": {Status: StatusUnknown},
			},
			expectedStatus: StatusUnknown,
		},
		{
			name: "mixed with unknown",
			checks: map[string]CheckResult{
				"check1": {Status: StatusHealthy},
				"check2": {Status: StatusUnknown},
			},
			expectedStatus: StatusDegraded,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, _ := manager.aggregateStatus(tt.checks)
			if status != tt.expectedStatus {
				t.Errorf("Expected status %s, got %s", tt.expectedStatus, status)
			}
		})
	}
}

func TestManagerStartStop(t *testing.T) {
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	manager := NewManager("test-service", "1.0.0", log)
	
	if manager.IsRunning() {
		t.Error("Expected manager to not be running initially")
	}
	
	manager.Start(100 * time.Millisecond)
	
	// Give it a moment to start
	time.Sleep(50 * time.Millisecond)
	
	if !manager.IsRunning() {
		t.Error("Expected manager to be running after start")
	}
	
	manager.Stop()
	
	// Give it a moment to stop
	time.Sleep(50 * time.Millisecond)
	
	if manager.IsRunning() {
		t.Error("Expected manager to not be running after stop")
	}
}