package logger

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap/zapcore"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "valid json logger",
			config: Config{
				Level:  "info",
				Format: "json",
			},
			expectError: false,
		},
		{
			name: "valid text logger",
			config: Config{
				Level:  "debug",
				Format: "text",
			},
			expectError: false,
		},
		{
			name: "invalid log level",
			config: Config{
				Level:  "invalid",
				Format: "json",
			},
			expectError: true,
		},
		{
			name: "invalid format",
			config: Config{
				Level:  "info",
				Format: "invalid",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewLogger(tt.config)

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

			if logger == nil {
				t.Errorf("Expected logger but got nil")
			}

			// Clean up
			if logger != nil {
				_ = logger.Sync()
			}
		})
	}
}

func TestLoggerWithFile(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test.log")
	errorFile := filepath.Join(tmpDir, "error.log")

	config := Config{
		Level:      "debug",
		Format:     "json",
		OutputPath: logFile,
		ErrorPath:  errorFile,
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	// Log some messages
	logger.Info("test info message", String("key", "value"))
	logger.Error("test error message", String("error_key", "error_value"))
	_ = logger.Sync()

	// Check that files were created and contain expected content
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Errorf("Log file was not created")
	}

	if _, err := os.Stat(errorFile); os.IsNotExist(err) {
		t.Errorf("Error file was not created")
	}

	// Read and verify log content
	logContent, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(logContent), "test info message") {
		t.Errorf("Log file does not contain expected info message")
	}

	errorContent, err := os.ReadFile(errorFile)
	if err != nil {
		t.Fatalf("Failed to read error file: %v", err)
	}

	if !strings.Contains(string(errorContent), "test error message") {
		t.Errorf("Error file does not contain expected error message")
	}
}

func TestGlobalLogger(t *testing.T) {
	// Reset global logger
	globalLogger = nil

	config := Config{
		Level:  "debug",
		Format: "json",
	}

	err := InitializeGlobal(config)
	if err != nil {
		t.Fatalf("Failed to initialize global logger: %v", err)
	}

	// Test global logging functions
	Debug("debug message", String("key", "debug"))
	Info("info message", String("key", "info"))
	Warn("warn message", String("key", "warn"))

	// Test getting global logger
	logger := GetGlobal()
	if logger == nil {
		t.Errorf("Global logger is nil")
	}

	// Test that GetGlobal creates fallback logger if not initialized
	globalLogger = nil
	fallbackLogger := GetGlobal()
	if fallbackLogger == nil {
		t.Errorf("Fallback logger is nil")
	}
}

func TestLoggerWithFields(t *testing.T) {
	config := Config{
		Level:  "info",
		Format: "json",
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	// Test WithFields
	componentLogger := logger.WithComponent("test-component")
	regionLogger := componentLogger.WithRegion("us-east-1")

	// Capture output would require more complex setup, so we just test that methods don't panic
	regionLogger.Info("test message with fields")

	// Test other With methods
	requestLogger := logger.WithRequestID("req-123")
	requestLogger.Info("test with request ID")

	collectorLogger := logger.WithCollector("ec2")
	collectorLogger.Info("test with collector")

	errorLogger := logger.WithError(errors.New("test error"))
	errorLogger.Warn("test with error context")
}

func TestStructuredLoggingMethods(t *testing.T) {
	config := Config{
		Level:  "debug",
		Format: "json",
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	// Test structured logging methods
	logger.LogStartup("1.0.0", "2023-01-01", "abc123")
	logger.LogConfigLoad("/path/to/config", []string{"us-east-1", "us-west-2"})
	logger.LogCollectorStatus("ec2", true, 5*time.Minute)
	logger.LogMetricCollection("ec2", "us-east-1", 10, 2*time.Second)
	logger.LogMetricExport(50, 1*time.Second)
	logger.LogError("test operation", errors.New("test error"))
	logger.LogAWSAPICall("ec2", "DescribeInstances", "us-east-1", 500*time.Millisecond, nil)
	logger.LogAWSAPICall("ec2", "DescribeInstances", "us-east-1", 500*time.Millisecond, errors.New("API error"))
	logger.LogHealthCheck("database", true, "connection successful")
	logger.LogShutdown("SIGTERM", 5*time.Second)
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected zapcore.Level
		hasError bool
	}{
		{"debug", zapcore.DebugLevel, false},
		{"info", zapcore.InfoLevel, false},
		{"warn", zapcore.WarnLevel, false},
		{"warning", zapcore.WarnLevel, false},
		{"error", zapcore.ErrorLevel, false},
		{"fatal", zapcore.FatalLevel, false},
		{"DEBUG", zapcore.DebugLevel, false},
		{"INFO", zapcore.InfoLevel, false},
		{"invalid", zapcore.InfoLevel, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			level, err := parseLogLevel(tt.input)

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

			if level != tt.expected {
				t.Errorf("Expected level %v, got %v for input %s", tt.expected, level, tt.input)
			}
		})
	}
}

func TestFieldHelpers(t *testing.T) {
	// Test that field helper functions don't panic
	fields := []Field{
		String("string_key", "value"),
		Int("int_key", 42),
		Int64("int64_key", 42),
		Float64("float_key", 3.14),
		Bool("bool_key", true),
		Duration("duration_key", time.Second),
		Err(errors.New("test error")),
		Any("any_key", map[string]string{"nested": "value"}),
		Strings("strings_key", []string{"a", "b", "c"}),
		Time("time_key", time.Now()),
	}

	config := Config{
		Level:  "debug",
		Format: "json",
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	// This should not panic
	logger.Info("test message with all field types", fields...)
}

func TestJSONOutput(t *testing.T) {
	// Create a logger that writes to buffer (this is complex with zap, so we'll test the config)
	config := Config{
		Level:  "info",
		Format: "json",
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	// Test that logger was created with JSON format
	if logger.config.Format != "json" {
		t.Errorf("Expected JSON format, got %s", logger.config.Format)
	}
}

func TestTextOutput(t *testing.T) {
	config := Config{
		Level:  "info",
		Format: "text",
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	// Test that logger was created with text format
	if logger.config.Format != "text" {
		t.Errorf("Expected text format, got %s", logger.config.Format)
	}
}

func TestLogLevels(t *testing.T) {
	config := Config{
		Level:  "warn",
		Format: "json",
	}

	logger, err := NewLogger(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	// These should not panic, even though debug/info might not be output
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")
}

// Benchmark tests
func BenchmarkLogger(b *testing.B) {
	config := Config{
		Level:  "info",
		Format: "json",
	}

	logger, err := NewLogger(config)
	if err != nil {
		b.Fatalf("Failed to create logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message",
			String("key1", "value1"),
			Int("key2", i),
			Bool("key3", true),
		)
	}
}

func BenchmarkGlobalLogger(b *testing.B) {
	config := Config{
		Level:  "info",
		Format: "json",
	}

	err := InitializeGlobal(config)
	if err != nil {
		b.Fatalf("Failed to initialize global logger: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info("benchmark message",
			String("key1", "value1"),
			Int("key2", i),
			Bool("key3", true),
		)
	}
}
