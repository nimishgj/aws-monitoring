package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"aws-monitoring/pkg/logger"
)

func TestNewServer(t *testing.T) {
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	manager := NewManager("test-service", "1.0.0", log)
	server := NewServer(manager, 8080, log)
	
	if server == nil {
		t.Fatal("Expected non-nil server")
	}
	
	if server.manager != manager {
		t.Error("Manager not properly set")
	}
	
	if server.port != 8080 {
		t.Errorf("Expected port 8080, got %d", server.port)
	}
}

func TestHealthEndpoint(t *testing.T) {
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	manager := NewManager("test-service", "1.0.0", log)
	server := NewServer(manager, 8080, log)
	
	// Add a healthy checker
	checker := newMockChecker("test-checker", StatusHealthy, "All good")
	manager.RegisterChecker(checker)
	manager.RunChecks(context.Background())
	time.Sleep(50 * time.Millisecond)
	
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	
	server.handleHealth(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	
	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got %v", response["status"])
	}
	
	if response["service"] != "test-service" {
		t.Errorf("Expected service 'test-service', got %v", response["service"])
	}
}

func TestHealthEndpointMethodNotAllowed(t *testing.T) {
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	manager := NewManager("test-service", "1.0.0", log)
	server := NewServer(manager, 8080, log)
	
	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	w := httptest.NewRecorder()
	
	server.handleHealth(w, req)
	
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestLivenessEndpoint(t *testing.T) {
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	manager := NewManager("test-service", "1.0.0", log)
	server := NewServer(manager, 8080, log)
	
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()
	
	server.handleLiveness(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	
	if response["status"] != "alive" {
		t.Errorf("Expected status 'alive', got %v", response["status"])
	}
}

func TestReadinessEndpoint(t *testing.T) {
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	manager := NewManager("test-service", "1.0.0", log)
	server := NewServer(manager, 8080, log)
	
	tests := []struct {
		name           string
		checkerStatus  Status
		expectedCode   int
		expectedReady  string
	}{
		{
			name:          "healthy",
			checkerStatus: StatusHealthy,
			expectedCode:  http.StatusOK,
			expectedReady: "ready",
		},
		{
			name:          "degraded",
			checkerStatus: StatusDegraded,
			expectedCode:  http.StatusOK,
			expectedReady: "ready",
		},
		{
			name:          "unhealthy",
			checkerStatus: StatusUnhealthy,
			expectedCode:  http.StatusServiceUnavailable,
			expectedReady: "not_ready",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear any existing checkers
			manager = NewManager("test-service", "1.0.0", log)
			server.manager = manager
			
			checker := newMockChecker("test-checker", tt.checkerStatus, "Test message")
			manager.RegisterChecker(checker)
			manager.RunChecks(context.Background())
			time.Sleep(50 * time.Millisecond)
			
			req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
			w := httptest.NewRecorder()
			
			server.handleReadiness(w, req)
			
			if w.Code != tt.expectedCode {
				t.Errorf("Expected status %d, got %d", tt.expectedCode, w.Code)
			}
			
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}
			
			if response["status"] != tt.expectedReady {
				t.Errorf("Expected status '%s', got %v", tt.expectedReady, response["status"])
			}
		})
	}
}

func TestDetailedHealthEndpoint(t *testing.T) {
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	manager := NewManager("test-service", "1.0.0", log)
	server := NewServer(manager, 8080, log)
	
	// Add multiple checkers
	checker1 := newMockChecker("checker1", StatusHealthy, "All good")
	checker2 := newMockChecker("checker2", StatusDegraded, "Some issues")
	
	manager.RegisterChecker(checker1)
	manager.RegisterChecker(checker2)
	manager.RunChecks(context.Background())
	time.Sleep(50 * time.Millisecond)
	
	req := httptest.NewRequest(http.MethodGet, "/health/detailed", nil)
	w := httptest.NewRecorder()
	
	server.handleDetailedHealth(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	
	var response OverallHealth
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	
	if response.Status != StatusDegraded {
		t.Errorf("Expected status 'degraded', got %s", response.Status)
	}
	
	if len(response.Checks) != 2 {
		t.Errorf("Expected 2 checks, got %d", len(response.Checks))
	}
	
	if response.ServiceName != "test-service" {
		t.Errorf("Expected service 'test-service', got %s", response.ServiceName)
	}
}

func TestStatusToHTTPCode(t *testing.T) {
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, err := logger.NewLogger(loggerConfig)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	manager := NewManager("test-service", "1.0.0", log)
	server := NewServer(manager, 8080, log)
	
	tests := []struct {
		status   Status
		expected int
	}{
		{StatusHealthy, http.StatusOK},
		{StatusDegraded, http.StatusOK},
		{StatusUnhealthy, http.StatusServiceUnavailable},
		{StatusUnknown, http.StatusServiceUnavailable},
		{Status("invalid"), http.StatusInternalServerError},
	}
	
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			code := server.statusToHTTPCode(tt.status)
			if code != tt.expected {
				t.Errorf("Expected HTTP code %d for status %s, got %d", tt.expected, tt.status, code)
			}
		})
	}
}