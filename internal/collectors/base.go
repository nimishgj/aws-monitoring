package collectors

import (
	"context"
	"sync"
	"time"

	"aws-monitoring/internal/aws"
	"aws-monitoring/internal/config"
	"aws-monitoring/pkg/errors"
	"aws-monitoring/pkg/logger"
)

// BaseCollector provides common functionality for all metric collectors
type BaseCollector struct {
	// name is the unique identifier for this collector
	name string
	// description explains what this collector does
	description string
	// config contains the application configuration
	config *config.Config
	// collectorConfig contains collector-specific configuration
	collectorConfig CollectorConfig
	// awsProvider provides AWS client access
	awsProvider aws.ClientProvider
	// logger for structured logging
	logger *logger.Logger
	// errorHandler handles and processes errors
	errorHandler ErrorHandler
	
	// State management
	mu                    sync.RWMutex
	status               CollectorStatus
	lastCollection       *time.Time
	lastError            *errors.Error
	metricsCollected     int64
	errorCount           int64
	successfulCollections int64
	
	// Lifecycle management
	startTime time.Time
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewBaseCollector creates a new base collector
func NewBaseCollector(
	name, description string,
	config *config.Config,
	collectorConfig CollectorConfig,
	awsProvider aws.ClientProvider,
	logger *logger.Logger,
) *BaseCollector {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &BaseCollector{
		name:            name,
		description:     description,
		config:          config,
		collectorConfig: collectorConfig,
		awsProvider:     awsProvider,
		logger:          logger.WithComponent("collector-" + name),
		status:          StatusStopped,
		ctx:             ctx,
		cancel:          cancel,
		errorHandler:    NewDefaultErrorHandler(logger),
	}
}

// Name returns the collector name
func (bc *BaseCollector) Name() string {
	return bc.name
}

// Description returns the collector description
func (bc *BaseCollector) Description() string {
	return bc.description
}

// Start initializes the collector
func (bc *BaseCollector) Start(_ context.Context) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	if bc.status == StatusRunning {
		return nil
	}
	
	bc.logger.Info("Starting collector", logger.String("collector", bc.name))
	bc.status = StatusStarting
	bc.startTime = time.Now()
	
	// Validate configuration
	if err := bc.validateConfig(); err != nil {
		bc.status = StatusError
		bc.lastError = err
		return err
	}
	
	bc.status = StatusRunning
	bc.logger.Info("Collector started successfully", 
		logger.String("collector", bc.name),
		logger.Strings("regions", bc.getEnabledRegions()))
	
	return nil
}

// Stop gracefully shuts down the collector
func (bc *BaseCollector) Stop(ctx context.Context) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	
	if bc.status == StatusStopped {
		return nil
	}
	
	bc.logger.Info("Stopping collector", logger.String("collector", bc.name))
	bc.status = StatusStopping
	bc.cancel()
	
	// Give some time for any ongoing operations to complete
	select {
	case <-time.After(5 * time.Second):
		bc.logger.Warn("Collector stop timeout", logger.String("collector", bc.name))
	case <-ctx.Done():
		// Context cancelled, force stop
	}
	
	bc.status = StatusStopped
	bc.logger.Info("Collector stopped", logger.String("collector", bc.name))
	
	return nil
}

// Info returns current collector information
func (bc *BaseCollector) Info() CollectorInfo {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	
	return CollectorInfo{
		Name:                  bc.name,
		Description:           bc.description,
		Status:                bc.status,
		EnabledRegions:        bc.getEnabledRegions(),
		Interval:              bc.collectorConfig.Interval,
		LastCollection:        bc.lastCollection,
		LastError:             bc.lastError,
		MetricsCollected:      bc.metricsCollected,
		ErrorCount:            bc.errorCount,
		SuccessfulCollections: bc.successfulCollections,
	}
}

// Health returns the health status of the collector
func (bc *BaseCollector) Health() error {
	bc.mu.RLock()
	defer bc.mu.RUnlock()
	
	switch bc.status {
	case StatusRunning:
		// Check if we've had successful collections recently
		if bc.lastCollection != nil && time.Since(*bc.lastCollection) < 2*bc.collectorConfig.Interval {
			return nil
		}
		// Check error rate
		if bc.successfulCollections > 0 {
			errorRate := float64(bc.errorCount) / float64(bc.successfulCollections+bc.errorCount)
			if errorRate > 0.5 { // More than 50% error rate
				return errors.NewValidationError("HIGH_ERROR_RATE", 
					"collector has high error rate")
			}
		}
		return nil
	case StatusError:
		return bc.lastError
	case StatusStopped:
		return errors.NewValidationError("COLLECTOR_STOPPED", "collector is stopped")
	default:
		return errors.NewValidationError("COLLECTOR_NOT_READY", "collector is not ready")
	}
}

// CreateMetric creates a standardized metric data point
func (bc *BaseCollector) CreateMetric(name string, value float64, unit string, labels map[string]string) MetricData {
	// Add common labels
	commonLabels := bc.getCommonLabels()
	if labels == nil {
		labels = commonLabels
	} else {
		// Merge labels (specific labels override common ones)
		for k, v := range commonLabels {
			if _, exists := labels[k]; !exists {
				labels[k] = v
			}
		}
	}
	
	return MetricData{
		Name:      name,
		Value:     value,
		Unit:      unit,
		Timestamp: time.Now(),
		Labels:    labels,
	}
}

// CreateMetricWithDescription creates a metric with description
func (bc *BaseCollector) CreateMetricWithDescription(name string, value float64, unit, description string, labels map[string]string) MetricData {
	metric := bc.CreateMetric(name, value, unit, labels)
	metric.Description = description
	return metric
}

// CollectWithRetry performs collection with retry logic
func (bc *BaseCollector) CollectWithRetry(ctx context.Context, region string, collectFunc func(ctx context.Context, region string) ([]MetricData, error)) *CollectionResult {
	start := time.Now()
	result := &CollectionResult{
		CollectorName:  bc.name,
		Region:         region,
		CollectionTime: start,
		Metrics:        []MetricData{},
		Warnings:       []*errors.Error{},
		Metadata:       make(map[string]interface{}),
	}
	
	var lastErr *errors.Error
	
	for attempt := 0; attempt < bc.collectorConfig.Retries+1; attempt++ {
		// Check if context is cancelled
		if ctx.Err() != nil {
			result.Error = errors.Wrap(ctx.Err(), errors.ErrorTypeInternal, "CONTEXT_CANCELLED", "collection cancelled")
			break
		}
		
		// Create a timeout context for this attempt
		collectCtx, cancel := context.WithTimeout(ctx, bc.collectorConfig.Timeout)
		
		metrics, err := collectFunc(collectCtx, region)
		cancel()
		
		if err == nil {
			// Success
			result.Metrics = metrics
			bc.recordSuccess()
			break
		}
		
		// Handle error
		if e, ok := err.(*errors.Error); ok {
			lastErr = e
		} else {
			lastErr = errors.Wrap(err, errors.ErrorTypeInternal, "COLLECTION_ERROR", "collection failed")
		}
		
		lastErr = errors.WithRegion(errors.WithOperation(lastErr, "collect"), region)
		
		// Check if we should retry
		if !bc.errorHandler.ShouldRetry(lastErr, attempt) {
			break
		}
		
		// Wait before retry (unless it's the last attempt)
		if attempt < bc.collectorConfig.Retries {
			retryDelay := bc.errorHandler.GetRetryDelay(lastErr, attempt)
			bc.logger.Warn("Collection failed, retrying",
				logger.String("collector", bc.name),
				logger.String("region", region),
				logger.Int("attempt", attempt+1),
				logger.Duration("retry_delay", retryDelay),
				logger.String("error", lastErr.Error()))
			
			select {
			case <-time.After(retryDelay):
				// Continue to retry
			case <-ctx.Done():
				result.Error = errors.Wrap(ctx.Err(), errors.ErrorTypeInternal, "CONTEXT_CANCELLED", "collection cancelled during retry")
				bc.recordError(result.Error)
				result.Duration = time.Since(start)
				return result
			}
		}
	}
	
	if lastErr != nil && len(result.Metrics) == 0 {
		result.Error = lastErr
		bc.recordError(lastErr)
		bc.errorHandler.HandleError(bc.name, lastErr)
	}
	
	result.Duration = time.Since(start)
	bc.recordCollection()
	
	// Add collection metadata
	result.Metadata["attempts"] = len(result.Warnings) + 1
	if result.Error != nil {
		result.Metadata["attempts"] = bc.collectorConfig.Retries + 1
	}
	result.Metadata["metric_count"] = len(result.Metrics)
	
	return result
}

// Helper methods

func (bc *BaseCollector) validateConfig() *errors.Error {
	if bc.collectorConfig.Interval <= 0 {
		return errors.NewConfigError("INVALID_INTERVAL", "collection interval must be positive")
	}
	
	if bc.collectorConfig.Timeout <= 0 {
		return errors.NewConfigError("INVALID_TIMEOUT", "collection timeout must be positive")
	}
	
	if bc.collectorConfig.Retries < 0 {
		return errors.NewConfigError("INVALID_RETRIES", "retries must be non-negative")
	}
	
	if len(bc.getEnabledRegions()) == 0 {
		return errors.NewConfigError("NO_REGIONS", "no regions enabled for collection")
	}
	
	return nil
}

func (bc *BaseCollector) getEnabledRegions() []string {
	if len(bc.collectorConfig.EnabledRegions) > 0 {
		return bc.collectorConfig.EnabledRegions
	}
	return bc.config.EnabledRegions
}

func (bc *BaseCollector) getCommonLabels() map[string]string {
	labels := map[string]string{
		"collector": bc.name,
		"service":   "aws-monitor",
	}
	
	// Add custom tags from configuration
	for k, v := range bc.collectorConfig.CustomTags {
		labels[k] = v
	}
	
	return labels
}

func (bc *BaseCollector) recordSuccess() {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.successfulCollections++
}

func (bc *BaseCollector) recordError(err *errors.Error) {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	bc.errorCount++
	bc.lastError = err
}

func (bc *BaseCollector) recordCollection() {
	bc.mu.Lock()
	defer bc.mu.Unlock()
	now := time.Now()
	bc.lastCollection = &now
	bc.metricsCollected++
}

// SetErrorHandler allows customizing error handling
func (bc *BaseCollector) SetErrorHandler(handler ErrorHandler) {
	bc.errorHandler = handler
}

// GetAWSProvider returns the AWS provider for subclasses
func (bc *BaseCollector) GetAWSProvider() aws.ClientProvider {
	return bc.awsProvider
}

// GetConfig returns the application configuration
func (bc *BaseCollector) GetConfig() *config.Config {
	return bc.config
}

// GetCollectorConfig returns the collector configuration
func (bc *BaseCollector) GetCollectorConfig() CollectorConfig {
	return bc.collectorConfig
}

// GetLogger returns the logger
func (bc *BaseCollector) GetLogger() *logger.Logger {
	return bc.logger
}