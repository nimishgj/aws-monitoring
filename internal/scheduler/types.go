// Package scheduler provides metric collection scheduling functionality.
package scheduler

import (
	"context"
	"time"

	"aws-monitoring/internal/collectors"
	"aws-monitoring/pkg/errors"
)

// Status represents the current status of the scheduler
type Status string

const (
	// StatusStarting indicates the scheduler is initializing
	StatusStarting Status = "starting"
	// StatusRunning indicates the scheduler is actively running
	StatusRunning Status = "running"
	// StatusStopping indicates the scheduler is shutting down
	StatusStopping Status = "stopping"
	// StatusStopped indicates the scheduler is stopped
	StatusStopped Status = "stopped"
	// StatusError indicates the scheduler is in an error state
	StatusError Status = "error"
)

// ScheduledJob represents a scheduled collection job
type ScheduledJob struct {
	// ID is a unique identifier for this job
	ID string `json:"id"`
	// CollectorName is the name of the collector to run
	CollectorName string `json:"collector_name"`
	// Region is the AWS region to collect from
	Region string `json:"region"`
	// Interval is how often to run this job
	Interval time.Duration `json:"interval"`
	// NextRun is when this job should next execute
	NextRun time.Time `json:"next_run"`
	// LastRun is when this job last executed
	LastRun *time.Time `json:"last_run,omitempty"`
	// LastResult is the result of the last execution
	LastResult *collectors.CollectionResult `json:"last_result,omitempty"`
	// Enabled indicates if this job should run
	Enabled bool `json:"enabled"`
}

// Config provides configuration for the scheduler
type Config struct {
	// TickInterval is how often the scheduler checks for jobs to run
	TickInterval time.Duration `json:"tick_interval"`
	// MaxConcurrentJobs is the maximum number of jobs that can run simultaneously
	MaxConcurrentJobs int `json:"max_concurrent_jobs"`
	// JobTimeout is the maximum time a single job can run
	JobTimeout time.Duration `json:"job_timeout"`
	// EnabledRegions restricts scheduling to specific regions
	EnabledRegions []string `json:"enabled_regions,omitempty"`
}

// DefaultConfig returns sensible defaults for scheduler configuration
func DefaultConfig() Config {
	return Config{
		TickInterval:      30 * time.Second,
		MaxConcurrentJobs: 10,
		JobTimeout:        5 * time.Minute,
	}
}

// Info provides information about the scheduler
type Info struct {
	// Status is the current status of the scheduler
	Status Status `json:"status"`
	// StartTime is when the scheduler was started
	StartTime *time.Time `json:"start_time,omitempty"`
	// JobCount is the total number of scheduled jobs
	JobCount int `json:"job_count"`
	// ActiveJobs is the number of currently running jobs
	ActiveJobs int `json:"active_jobs"`
	// CompletedJobs is the total number of completed jobs
	CompletedJobs int64 `json:"completed_jobs"`
	// FailedJobs is the total number of failed jobs
	FailedJobs int64 `json:"failed_jobs"`
	// LastTickTime is when the scheduler last checked for jobs
	LastTickTime *time.Time `json:"last_tick_time,omitempty"`
}

// Scheduler defines the interface for metric collection scheduling
type Scheduler interface {
	// Start begins the scheduler execution
	Start(ctx context.Context) error
	
	// Stop gracefully shuts down the scheduler
	Stop(ctx context.Context) error
	
	// ScheduleCollector schedules a collector to run at specified intervals
	ScheduleCollector(collectorName string, regions []string, interval time.Duration) error
	
	// UnscheduleCollector removes a collector from the schedule
	UnscheduleCollector(collectorName string, region string) error
	
	// GetScheduledJobs returns all currently scheduled jobs
	GetScheduledJobs() []ScheduledJob
	
	// GetInfo returns current scheduler status and statistics
	GetInfo() Info
	
	// Health returns the health status of the scheduler
	Health() error
}

// JobProcessor defines how to process collection results
type JobProcessor interface {
	// ProcessResult handles the result of a collection job
	ProcessResult(ctx context.Context, job *ScheduledJob, result *collectors.CollectionResult) error
	
	// ProcessError handles errors that occur during collection
	ProcessError(ctx context.Context, job *ScheduledJob, err *errors.Error) error
}

// JobExecutor defines how individual jobs are executed
type JobExecutor interface {
	// ExecuteJob runs a single collection job
	ExecuteJob(ctx context.Context, job *ScheduledJob) *collectors.CollectionResult
}