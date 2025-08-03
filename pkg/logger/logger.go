package logger

import (
	"fmt"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps zap.Logger with additional functionality
type Logger struct {
	*zap.Logger
	config Config
}

// Config holds logger configuration
type Config struct {
	Level      string `yaml:"level" validate:"oneof=debug info warn error"`
	Format     string `yaml:"format" validate:"oneof=json text"`
	OutputPath string `yaml:"output_path"`
	ErrorPath  string `yaml:"error_path"`
}

// Field represents a structured log field
type Field = zap.Field

// Global logger instance
var globalLogger *Logger

// String creates a string field
func String(key, val string) Field {
	return zap.String(key, val)
}

// Int creates an int field
func Int(key string, val int) Field {
	return zap.Int(key, val)
}

// Int64 creates an int64 field
func Int64(key string, val int64) Field {
	return zap.Int64(key, val)
}

// Float64 creates a float64 field
func Float64(key string, val float64) Field {
	return zap.Float64(key, val)
}

// Bool creates a bool field
func Bool(key string, val bool) Field {
	return zap.Bool(key, val)
}

// Duration creates a duration field
func Duration(key string, val time.Duration) Field {
	return zap.Duration(key, val)
}

// Err creates an error field
func Err(err error) Field {
	return zap.Error(err)
}

// Any creates a field with any type
func Any(key string, val interface{}) Field {
	return zap.Any(key, val)
}

// Strings creates a string slice field
func Strings(key string, val []string) Field {
	return zap.Strings(key, val)
}

// Time creates a time field
func Time(key string, val time.Time) Field {
	return zap.Time(key, val)
}

// NewLogger creates a new logger instance
func NewLogger(config Config) (*Logger, error) {
	// Parse log level
	level, err := parseLogLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level %s: %w", config.Level, err)
	}

	// Create encoder config
	encoderConfig := getEncoderConfig(config.Format)

	// Create encoder
	var encoder zapcore.Encoder
	switch strings.ToLower(config.Format) {
	case "json":
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	case "text", "console":
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	default:
		return nil, fmt.Errorf("unsupported log format: %s", config.Format)
	}

	// Configure output paths
	outputPath := config.OutputPath
	if outputPath == "" {
		outputPath = "stdout"
	}

	errorPath := config.ErrorPath
	if errorPath == "" {
		errorPath = "stderr"
	}

	// Create core
	writeSyncer := getWriteSyncer(outputPath)
	errorWriteSyncer := getWriteSyncer(errorPath)

	// Create separate cores for different levels
	infoCore := zapcore.NewCore(
		encoder,
		writeSyncer,
		zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= level && lvl < zapcore.ErrorLevel
		}),
	)

	errorCore := zapcore.NewCore(
		encoder,
		errorWriteSyncer,
		zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
			return lvl >= zapcore.ErrorLevel
		}),
	)

	core := zapcore.NewTee(infoCore, errorCore)

	// Create logger with options
	zapLogger := zap.New(core,
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)

	logger := &Logger{
		Logger: zapLogger,
		config: config,
	}

	return logger, nil
}

// InitializeGlobal initializes the global logger
func InitializeGlobal(config Config) error {
	logger, err := NewLogger(config)
	if err != nil {
		return err
	}

	globalLogger = logger
	return nil
}

// GetGlobal returns the global logger instance
func GetGlobal() *Logger {
	if globalLogger == nil {
		// Fallback to a basic logger if not initialized
		config := Config{
			Level:  "info",
			Format: "json",
		}
		logger, _ := NewLogger(config)
		globalLogger = logger
	}
	return globalLogger
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	return l.Logger.Sync()
}

// WithFields creates a logger with structured fields
func (l *Logger) WithFields(fields ...Field) *Logger {
	return &Logger{
		Logger: l.Logger.With(fields...),
		config: l.config,
	}
}

// WithComponent creates a logger for a specific component
func (l *Logger) WithComponent(component string) *Logger {
	return l.WithFields(String("component", component))
}

// WithRequestID creates a logger with request ID
func (l *Logger) WithRequestID(requestID string) *Logger {
	return l.WithFields(String("request_id", requestID))
}

// WithCollector creates a logger for a specific collector
func (l *Logger) WithCollector(collector string) *Logger {
	return l.WithFields(String("collector", collector))
}

// WithRegion creates a logger with AWS region
func (l *Logger) WithRegion(region string) *Logger {
	return l.WithFields(String("region", region))
}

// WithError creates a logger with error context
func (l *Logger) WithError(err error) *Logger {
	return l.WithFields(Err(err))
}

// LogStartup logs application startup information
func (l *Logger) LogStartup(version, buildTime, gitCommit string) {
	l.Info("Application starting",
		String("version", version),
		String("build_time", buildTime),
		String("git_commit", gitCommit),
	)
}

// LogConfigLoad logs configuration loading
func (l *Logger) LogConfigLoad(configPath string, regions []string) {
	l.Info("Configuration loaded",
		String("config_path", configPath),
		Strings("enabled_regions", regions),
	)
}

// LogCollectorStatus logs collector status changes
func (l *Logger) LogCollectorStatus(collector string, enabled bool, interval time.Duration) {
	l.Info("Collector configured",
		String("collector", collector),
		Bool("enabled", enabled),
		Duration("interval", interval),
	)
}

// LogMetricCollection logs metric collection events
func (l *Logger) LogMetricCollection(collector, region string, count int, duration time.Duration) {
	l.Info("Metrics collected",
		String("collector", collector),
		String("region", region),
		Int("metric_count", count),
		Duration("collection_duration", duration),
	)
}

// LogMetricExport logs metric export events
func (l *Logger) LogMetricExport(count int, duration time.Duration) {
	l.Info("Metrics exported",
		Int("metric_count", count),
		Duration("export_duration", duration),
	)
}

// LogError logs errors with context
func (l *Logger) LogError(operation string, err error, fields ...Field) {
	allFields := append([]Field{String("operation", operation), Err(err)}, fields...)
	l.Error("Operation failed", allFields...)
}

// LogAWSAPICall logs AWS API calls
func (l *Logger) LogAWSAPICall(service, operation, region string, duration time.Duration, err error) {
	fields := []Field{
		String("aws_service", service),
		String("aws_operation", operation),
		String("region", region),
		Duration("duration", duration),
	}

	if err != nil {
		fields = append(fields, Err(err))
		l.Warn("AWS API call failed", fields...)
	} else {
		l.Debug("AWS API call succeeded", fields...)
	}
}

// LogHealthCheck logs health check status
func (l *Logger) LogHealthCheck(component string, healthy bool, message string) {
	l.Info("Health check",
		String("component", component),
		Bool("healthy", healthy),
		String("message", message),
	)
}

// LogShutdown logs application shutdown
func (l *Logger) LogShutdown(reason string, duration time.Duration) {
	l.Info("Application shutting down",
		String("reason", reason),
		Duration("shutdown_duration", duration),
	)
}

// Global logging functions that use the global logger

// Debug logs a debug message
func Debug(msg string, fields ...Field) {
	GetGlobal().Debug(msg, fields...)
}

// Info logs an info message
func Info(msg string, fields ...Field) {
	GetGlobal().Info(msg, fields...)
}

// Warn logs a warning message
func Warn(msg string, fields ...Field) {
	GetGlobal().Warn(msg, fields...)
}

// Error logs an error message
func Error(msg string, fields ...Field) {
	GetGlobal().Error(msg, fields...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string, fields ...Field) {
	GetGlobal().Fatal(msg, fields...)
}

// Sync flushes the global logger
func Sync() error {
	return GetGlobal().Sync()
}

// WithFields creates a logger with fields from global logger
func WithFields(fields ...Field) *Logger {
	return GetGlobal().WithFields(fields...)
}

// WithComponent creates a component logger from global logger
func WithComponent(component string) *Logger {
	return GetGlobal().WithComponent(component)
}

// Helper functions

func parseLogLevel(level string) (zapcore.Level, error) {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn", "warning":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	case "fatal":
		return zapcore.FatalLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("unknown log level: %s", level)
	}
}

func getEncoderConfig(format string) zapcore.EncoderConfig {
	config := zap.NewProductionEncoderConfig()
	config.TimeKey = "timestamp"
	config.LevelKey = "level"
	config.NameKey = "logger"
	config.CallerKey = "caller"
	config.MessageKey = "message"
	config.StacktraceKey = "stacktrace"
	config.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncodeLevel = zapcore.LowercaseLevelEncoder
	config.EncodeCaller = zapcore.ShortCallerEncoder

	if strings.ToLower(format) == "text" || strings.ToLower(format) == "console" {
		config.EncodeLevel = zapcore.CapitalColorLevelEncoder
		config.EncodeTime = zapcore.ISO8601TimeEncoder
		config.ConsoleSeparator = " "
	}

	return config
}

func getWriteSyncer(path string) zapcore.WriteSyncer {
	switch path {
	case "stdout", "":
		return zapcore.AddSync(os.Stdout)
	case "stderr":
		return zapcore.AddSync(os.Stderr)
	default:
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			// Fallback to stdout if file cannot be opened
			return zapcore.AddSync(os.Stdout)
		}
		return zapcore.AddSync(file)
	}
}
