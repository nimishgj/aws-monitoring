// Package collectors provides the metric collection interfaces and implementations.
package collectors

import (
	"context"
	"time"

	"aws-monitoring/pkg/errors"
)

// MetricData represents a single metric data point
type MetricData struct {
	// Name is the metric name
	Name string `json:"name"`
	// Value is the metric value
	Value float64 `json:"value"`
	// Unit is the metric unit (e.g., "Count", "Bytes", "Percent")
	Unit string `json:"unit"`
	// Timestamp when the metric was collected
	Timestamp time.Time `json:"timestamp"`
	// Labels are key-value pairs that identify the metric
	Labels map[string]string `json:"labels"`
	// Description provides context about what this metric measures
	Description string `json:"description,omitempty"`
}

// CollectionResult represents the result of a metric collection operation
type CollectionResult struct {
	// CollectorName identifies which collector produced this result
	CollectorName string `json:"collector_name"`
	// Region where the collection occurred
	Region string `json:"region"`
	// Metrics contains all collected metric data points
	Metrics []MetricData `json:"metrics"`
	// CollectionTime when the collection started
	CollectionTime time.Time `json:"collection_time"`
	// Duration how long the collection took
	Duration time.Duration `json:"duration"`
	// Error if the collection failed
	Error *errors.Error `json:"error,omitempty"`
	// Warnings are non-fatal issues encountered during collection
	Warnings []*errors.Error `json:"warnings,omitempty"`
	// Metadata contains additional context about the collection
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// CollectorStatus represents the current status of a collector
type CollectorStatus string

const (
	// StatusStarting indicates the collector is initializing
	StatusStarting CollectorStatus = "starting"
	// StatusRunning indicates the collector is actively collecting
	StatusRunning CollectorStatus = "running"
	// StatusStopping indicates the collector is shutting down
	StatusStopping CollectorStatus = "stopping"
	// StatusStopped indicates the collector is stopped
	StatusStopped CollectorStatus = "stopped"
	// StatusError indicates the collector is in an error state
	StatusError CollectorStatus = "error"
)

// CollectorInfo provides information about a collector
type CollectorInfo struct {
	// Name is the unique identifier for this collector
	Name string `json:"name"`
	// Description explains what this collector does
	Description string `json:"description"`
	// Status is the current operational status
	Status CollectorStatus `json:"status"`
	// EnabledRegions are the regions this collector operates in
	EnabledRegions []string `json:"enabled_regions"`
	// Interval is how often this collector runs
	Interval time.Duration `json:"interval"`
	// LastCollection is when this collector last ran
	LastCollection *time.Time `json:"last_collection,omitempty"`
	// LastError is the most recent error encountered
	LastError *errors.Error `json:"last_error,omitempty"`
	// MetricsCollected is the total number of metrics collected
	MetricsCollected int64 `json:"metrics_collected"`
	// ErrorCount is the number of errors encountered
	ErrorCount int64 `json:"error_count"`
	// SuccessfulCollections is the number of successful collection runs
	SuccessfulCollections int64 `json:"successful_collections"`
}

// MetricCollector defines the interface that all metric collectors must implement
type MetricCollector interface {
	// Name returns the unique name of this collector
	Name() string
	
	// Description returns a human-readable description of what this collector does
	Description() string
	
	// Collect performs metric collection for the specified region
	// It should be safe to call concurrently from multiple goroutines
	Collect(ctx context.Context, region string) *CollectionResult
	
	// Start initializes the collector and prepares it for collection
	Start(ctx context.Context) error
	
	// Stop gracefully shuts down the collector
	Stop(ctx context.Context) error
	
	// Info returns current status and statistics about this collector
	Info() CollectorInfo
	
	// Health returns the health status of this collector
	Health() error
}

// CollectorConfig provides configuration for collectors
type CollectorConfig struct {
	// Enabled determines if this collector should be active
	Enabled bool `json:"enabled"`
	// Interval defines how often to collect metrics
	Interval time.Duration `json:"interval"`
	// Timeout defines the maximum time to wait for collection
	Timeout time.Duration `json:"timeout"`
	// Retries defines how many times to retry failed collections
	Retries int `json:"retries"`
	// RetryDelay defines the delay between retries
	RetryDelay time.Duration `json:"retry_delay"`
	// EnabledRegions restricts collection to specific regions
	EnabledRegions []string `json:"enabled_regions,omitempty"`
	// MetricFilters allow filtering which metrics to collect
	MetricFilters []string `json:"metric_filters,omitempty"`
	// CustomTags are additional tags to add to all metrics
	CustomTags map[string]string `json:"custom_tags,omitempty"`
}

// DefaultCollectorConfig returns sensible defaults for collector configuration
func DefaultCollectorConfig() CollectorConfig {
	return CollectorConfig{
		Enabled:    true,
		Interval:   5 * time.Minute,
		Timeout:    30 * time.Second,
		Retries:    3,
		RetryDelay: 10 * time.Second,
		CustomTags: make(map[string]string),
	}
}

// Registry defines the interface for managing collectors
type Registry interface {
	// Register adds a collector to the registry
	Register(collector MetricCollector) error
	
	// Unregister removes a collector from the registry
	Unregister(name string) error
	
	// Get returns a collector by name
	Get(name string) (MetricCollector, bool)
	
	// List returns all registered collectors
	List() []MetricCollector
	
	// Start starts all enabled collectors
	Start(ctx context.Context) error
	
	// Stop stops all collectors
	Stop(ctx context.Context) error
	
	// Status returns the status of all collectors
	Status() map[string]CollectorInfo
}

// ErrorHandler defines how to handle collector errors
type ErrorHandler interface {
	// HandleError processes an error from a collector
	HandleError(collectorName string, err *errors.Error)
	
	// ShouldRetry determines if an operation should be retried
	ShouldRetry(err *errors.Error, attempt int) bool
	
	// GetRetryDelay returns how long to wait before retrying
	GetRetryDelay(err *errors.Error, attempt int) time.Duration
}

// MetricProcessor defines how to process collected metrics
type MetricProcessor interface {
	// Process handles a collection result
	Process(ctx context.Context, result *CollectionResult) error
	
	// Start initializes the processor
	Start(ctx context.Context) error
	
	// Stop shuts down the processor
	Stop(ctx context.Context) error
}

// CollectorFactory creates collectors based on configuration
type CollectorFactory interface {
	// Create creates a new collector instance
	Create(name string, config CollectorConfig) (MetricCollector, error)
	
	// SupportedTypes returns the types of collectors this factory can create
	SupportedTypes() []string
}