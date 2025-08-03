package logger_test

import (
	"errors"
	"time"

	"aws-monitoring/pkg/logger"
)

func ExampleNewLogger() {
	config := logger.Config{
		Level:  "info",
		Format: "json",
	}

	log, err := logger.NewLogger(config)
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	log.Info("Application started successfully",
		logger.String("version", "1.0.0"),
		logger.Int("port", 8080),
	)
}

func ExampleLogger_WithComponent() {
	config := logger.Config{
		Level:  "info",
		Format: "json",
	}

	log, _ := logger.NewLogger(config)
	defer log.Sync()

	// Create component-specific logger
	dbLogger := log.WithComponent("database")
	dbLogger.Info("Database connection established")

	apiLogger := log.WithComponent("api")
	apiLogger.Info("API server started")
}

func ExampleLogger_LogAWSAPICall() {
	config := logger.Config{
		Level:  "debug",
		Format: "json",
	}

	log, _ := logger.NewLogger(config)
	defer log.Sync()

	// Log successful AWS API call
	log.LogAWSAPICall("ec2", "DescribeInstances", "us-east-1", 500*time.Millisecond, nil)

	// Log failed AWS API call
	log.LogAWSAPICall("ec2", "DescribeInstances", "us-east-1", 2*time.Second, errors.New("rate limit exceeded"))
}

func ExampleLogger_LogMetricCollection() {
	config := logger.Config{
		Level:  "info",
		Format: "json",
	}

	log, _ := logger.NewLogger(config)
	defer log.Sync()

	// Log metric collection event
	log.LogMetricCollection("ec2", "us-east-1", 42, 1500*time.Millisecond)
}

func ExampleGlobalLogging() {
	config := logger.Config{
		Level:  "info",
		Format: "text",
	}

	// Initialize global logger
	logger.InitializeGlobal(config)
	defer logger.Sync()

	// Use global logging functions
	logger.Info("Application starting",
		logger.String("component", "main"),
		logger.String("version", "1.0.0"),
	)

	logger.Debug("Debug information",
		logger.Any("config", map[string]interface{}{
			"debug": true,
			"port":  8080,
		}),
	)

	logger.Warn("Warning message",
		logger.String("reason", "deprecated API usage"),
	)
}

func ExampleLogger_WithFields() {
	config := logger.Config{
		Level:  "info",
		Format: "json",
	}

	log, _ := logger.NewLogger(config)
	defer log.Sync()

	// Create logger with context fields
	requestLogger := log.WithFields(
		logger.String("request_id", "req-12345"),
		logger.String("user_id", "user-789"),
		logger.String("operation", "create_resource"),
	)

	requestLogger.Info("Processing request")
	requestLogger.Info("Request completed successfully",
		logger.Duration("processing_time", 250*time.Millisecond),
	)
}
