package scheduler

import (
	"context"

	"aws-monitoring/internal/collectors"
	"aws-monitoring/pkg/errors"
	"aws-monitoring/pkg/logger"
)

// DefaultJobExecutor implements JobExecutor using the collector registry
type DefaultJobExecutor struct {
	registry collectors.Registry
	logger   *logger.Logger
}

// NewDefaultJobExecutor creates a new default job executor
func NewDefaultJobExecutor(registry collectors.Registry, log *logger.Logger) JobExecutor {
	return &DefaultJobExecutor{
		registry: registry,
		logger:   log.WithComponent("job-executor"),
	}
}

// ExecuteJob runs a single collection job
func (e *DefaultJobExecutor) ExecuteJob(ctx context.Context, job *ScheduledJob) *collectors.CollectionResult {
	// Get the collector from registry
	collector, exists := e.registry.Get(job.CollectorName)
	if !exists {
		return &collectors.CollectionResult{
			CollectorName:  job.CollectorName,
			Region:         job.Region,
			CollectionTime: job.NextRun,
			Metrics:        []collectors.MetricData{},
			Error: errors.NewValidationError("COLLECTOR_NOT_FOUND", 
				"collector not found in registry").
				WithMetadata("collector", job.CollectorName),
		}
	}

	// Execute the collection
	return collector.Collect(ctx, job.Region)
}

// DefaultJobProcessor implements JobProcessor with basic logging
type DefaultJobProcessor struct {
	logger *logger.Logger
}

// NewDefaultJobProcessor creates a new default job processor
func NewDefaultJobProcessor(log *logger.Logger) JobProcessor {
	return &DefaultJobProcessor{
		logger: log.WithComponent("job-processor"),
	}
}

// ProcessResult handles the result of a collection job
func (p *DefaultJobProcessor) ProcessResult(_ context.Context, job *ScheduledJob, result *collectors.CollectionResult) error {
	p.logger.Info("Collection result processed",
		logger.String("job_id", job.ID),
		logger.String("collector", job.CollectorName),
		logger.String("region", job.Region),
		logger.Int("metric_count", len(result.Metrics)),
		logger.Duration("duration", result.Duration))
	
	// In a full implementation, this would:
	// - Send metrics to output destinations (e.g., CloudWatch, Prometheus)
	// - Store metrics in a database
	// - Forward to other processors
	// For now, we just log the result
	
	return nil
}

// ProcessError handles errors that occur during collection
func (p *DefaultJobProcessor) ProcessError(_ context.Context, job *ScheduledJob, err *errors.Error) error {
	p.logger.Error("Collection error processed",
		logger.String("job_id", job.ID),
		logger.String("collector", job.CollectorName),
		logger.String("region", job.Region),
		logger.String("error_type", string(err.Type)),
		logger.String("error_code", err.Code),
		logger.String("error", err.Error()))
	
	// In a full implementation, this would:
	// - Send error alerts
	// - Update error metrics
	// - Trigger recovery actions
	// For now, we just log the error
	
	return nil
}