package collectors

import (
	"context"
	"fmt"
	"sync"

	"aws-monitoring/pkg/logger"
)

// CollectorRegistry manages a collection of metric collectors
type CollectorRegistry struct {
	collectors map[string]MetricCollector
	logger     *logger.Logger
	mu         sync.RWMutex
}

// NewCollectorRegistry creates a new collector registry
func NewCollectorRegistry(log *logger.Logger) Registry {
	return &CollectorRegistry{
		collectors: make(map[string]MetricCollector),
		logger:     log.WithComponent("collector-registry"),
	}
}

// Register adds a collector to the registry
func (r *CollectorRegistry) Register(collector MetricCollector) error {
	if collector == nil {
		return fmt.Errorf("collector cannot be nil")
	}
	
	name := collector.Name()
	if name == "" {
		return fmt.Errorf("collector name cannot be empty")
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.collectors[name]; exists {
		return fmt.Errorf("collector %s already registered", name)
	}
	
	r.collectors[name] = collector
	r.logger.Info("Collector registered", 
		logger.String("collector", name),
		logger.String("description", collector.Description()))
	
	return nil
}

// Unregister removes a collector from the registry
func (r *CollectorRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	collector, exists := r.collectors[name]
	if !exists {
		return fmt.Errorf("collector %s not found", name)
	}
	
	// Stop the collector if it's running
	if err := collector.Stop(context.Background()); err != nil {
		r.logger.Warn("Error stopping collector during unregister",
			logger.String("collector", name),
			logger.String("error", err.Error()))
	}
	
	delete(r.collectors, name)
	r.logger.Info("Collector unregistered", logger.String("collector", name))
	
	return nil
}

// Get returns a collector by name
func (r *CollectorRegistry) Get(name string) (MetricCollector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	collector, exists := r.collectors[name]
	return collector, exists
}

// List returns all registered collectors
func (r *CollectorRegistry) List() []MetricCollector {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	collectors := make([]MetricCollector, 0, len(r.collectors))
	for _, collector := range r.collectors {
		collectors = append(collectors, collector)
	}
	
	return collectors
}

// Start starts all enabled collectors
func (r *CollectorRegistry) Start(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	r.logger.Info("Starting all collectors", logger.Int("count", len(r.collectors)))
	
	var startErrors []error
	
	for name, collector := range r.collectors {
		if err := collector.Start(ctx); err != nil {
			startErrors = append(startErrors, fmt.Errorf("failed to start collector %s: %w", name, err))
			r.logger.Error("Failed to start collector",
				logger.String("collector", name),
				logger.String("error", err.Error()))
		} else {
			r.logger.Info("Collector started", logger.String("collector", name))
		}
	}
	
	if len(startErrors) > 0 {
		return fmt.Errorf("failed to start %d collectors: %v", len(startErrors), startErrors)
	}
	
	r.logger.Info("All collectors started successfully")
	return nil
}

// Stop stops all collectors
func (r *CollectorRegistry) Stop(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	r.logger.Info("Stopping all collectors", logger.Int("count", len(r.collectors)))
	
	var stopErrors []error
	
	for name, collector := range r.collectors {
		if err := collector.Stop(ctx); err != nil {
			stopErrors = append(stopErrors, fmt.Errorf("failed to stop collector %s: %w", name, err))
			r.logger.Error("Failed to stop collector",
				logger.String("collector", name),
				logger.String("error", err.Error()))
		} else {
			r.logger.Info("Collector stopped", logger.String("collector", name))
		}
	}
	
	if len(stopErrors) > 0 {
		return fmt.Errorf("failed to stop %d collectors: %v", len(stopErrors), stopErrors)
	}
	
	r.logger.Info("All collectors stopped successfully")
	return nil
}

// Status returns the status of all collectors
func (r *CollectorRegistry) Status() map[string]CollectorInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	status := make(map[string]CollectorInfo)
	for name, collector := range r.collectors {
		status[name] = collector.Info()
	}
	
	return status
}

// MetricProcessorRegistry manages metric processors
type MetricProcessorRegistry struct {
	processors []MetricProcessor
	logger     *logger.Logger
	mu         sync.RWMutex
}

// NewMetricProcessorRegistry creates a new processor registry
func NewMetricProcessorRegistry(log *logger.Logger) *MetricProcessorRegistry {
	return &MetricProcessorRegistry{
		processors: make([]MetricProcessor, 0),
		logger:     log.WithComponent("processor-registry"),
	}
}

// Register adds a processor to the registry
func (r *MetricProcessorRegistry) Register(processor MetricProcessor) error {
	if processor == nil {
		return fmt.Errorf("processor cannot be nil")
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.processors = append(r.processors, processor)
	r.logger.Info("Metric processor registered")
	
	return nil
}

// Start starts all processors
func (r *MetricProcessorRegistry) Start(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	r.logger.Info("Starting metric processors", logger.Int("count", len(r.processors)))
	
	for i, processor := range r.processors {
		if err := processor.Start(ctx); err != nil {
			return fmt.Errorf("failed to start processor %d: %w", i, err)
		}
	}
	
	r.logger.Info("All metric processors started")
	return nil
}

// Stop stops all processors
func (r *MetricProcessorRegistry) Stop(ctx context.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	r.logger.Info("Stopping metric processors", logger.Int("count", len(r.processors)))
	
	var stopErrors []error
	for i, processor := range r.processors {
		if err := processor.Stop(ctx); err != nil {
			stopErrors = append(stopErrors, fmt.Errorf("failed to stop processor %d: %w", i, err))
		}
	}
	
	if len(stopErrors) > 0 {
		return fmt.Errorf("failed to stop processors: %v", stopErrors)
	}
	
	r.logger.Info("All metric processors stopped")
	return nil
}

// Process sends collection results to all processors
func (r *MetricProcessorRegistry) Process(ctx context.Context, result *CollectionResult) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var processErrors []error
	
	for i, processor := range r.processors {
		if err := processor.Process(ctx, result); err != nil {
			processErrors = append(processErrors, fmt.Errorf("processor %d failed: %w", i, err))
		}
	}
	
	if len(processErrors) > 0 {
		return fmt.Errorf("processing errors: %v", processErrors)
	}
	
	return nil
}

// DefaultCollectorFactory provides a basic implementation of CollectorFactory
type DefaultCollectorFactory struct {
	logger *logger.Logger
}

// NewDefaultCollectorFactory creates a new default collector factory
func NewDefaultCollectorFactory(log *logger.Logger) CollectorFactory {
	return &DefaultCollectorFactory{
		logger: log.WithComponent("collector-factory"),
	}
}

// Create creates a new collector instance based on the name and configuration
func (f *DefaultCollectorFactory) Create(name string, _ CollectorConfig) (MetricCollector, error) {
	// This is a placeholder implementation
	// In a real implementation, you would create specific collector types based on the name
	return nil, fmt.Errorf("collector type %s not supported by default factory", name)
}

// SupportedTypes returns the types of collectors this factory can create
func (f *DefaultCollectorFactory) SupportedTypes() []string {
	// This would return the actual supported types in a real implementation
	return []string{}
}