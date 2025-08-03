package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"aws-monitoring/internal/collectors"
	"aws-monitoring/pkg/errors"
	"aws-monitoring/pkg/logger"
)

// MetricScheduler implements the Scheduler interface
type MetricScheduler struct {
	// Configuration
	config Config
	
	// Dependencies
	registry    collectors.Registry
	processor   JobProcessor
	executor    JobExecutor
	logger      *logger.Logger
	
	// State management
	mu            sync.RWMutex
	status        Status
	startTime     *time.Time
	lastTickTime  *time.Time
	
	// Job management
	jobs          map[string]*ScheduledJob
	activeJobs    map[string]context.CancelFunc
	completedJobs int64
	failedJobs    int64
	
	// Control channels
	stopCh   chan struct{}
	doneCh   chan struct{}
	
	// Job execution
	jobSemaphore chan struct{}
}

// NewMetricScheduler creates a new metric collection scheduler
func NewMetricScheduler(
	config Config,
	registry collectors.Registry,
	processor JobProcessor,
	log *logger.Logger,
) Scheduler {
	if processor == nil {
		processor = NewDefaultJobProcessor(log)
	}
	
	scheduler := &MetricScheduler{
		config:       config,
		registry:     registry,
		processor:    processor,
		executor:     NewDefaultJobExecutor(registry, log),
		logger:       log.WithComponent("scheduler"),
		status:       StatusStopped,
		jobs:         make(map[string]*ScheduledJob),
		activeJobs:   make(map[string]context.CancelFunc),
		stopCh:       make(chan struct{}),
		doneCh:       make(chan struct{}),
		jobSemaphore: make(chan struct{}, config.MaxConcurrentJobs),
	}
	
	return scheduler
}

// Start begins the scheduler execution
func (s *MetricScheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.status == StatusRunning {
		return nil // Already running
	}
	
	s.logger.Info("Starting metric scheduler", 
		logger.Duration("tick_interval", s.config.TickInterval),
		logger.Int("max_concurrent_jobs", s.config.MaxConcurrentJobs))
	
	s.status = StatusStarting
	now := time.Now()
	s.startTime = &now
	
	// Validate configuration
	if err := s.validateConfig(); err != nil {
		s.status = StatusError
		return err
	}
	
	s.status = StatusRunning
	
	// Start the main scheduler loop
	go s.run(ctx)
	
	s.logger.Info("Metric scheduler started successfully")
	return nil
}

// Stop gracefully shuts down the scheduler
func (s *MetricScheduler) Stop(ctx context.Context) error {
	s.mu.Lock()
	if s.status != StatusRunning {
		s.mu.Unlock()
		return nil // Not running
	}
	
	s.logger.Info("Stopping metric scheduler")
	s.status = StatusStopping
	s.mu.Unlock()
	
	// Signal stop
	close(s.stopCh)
	
	// Wait for scheduler to stop or timeout
	select {
	case <-s.doneCh:
		s.logger.Info("Metric scheduler stopped gracefully")
	case <-ctx.Done():
		s.logger.Warn("Metric scheduler stop timeout")
		return errors.NewTimeoutError("scheduler-stop", 
			s.config.JobTimeout).WithMetadata("operation", "stop")
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Cancel any active jobs
	for jobID, cancel := range s.activeJobs {
		s.logger.Debug("Cancelling active job", logger.String("job_id", jobID))
		cancel()
	}
	
	s.status = StatusStopped
	return nil
}

// ScheduleCollector schedules a collector to run at specified intervals
func (s *MetricScheduler) ScheduleCollector(collectorName string, regions []string, interval time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Validate collector exists
	if _, exists := s.registry.Get(collectorName); !exists {
		return errors.NewValidationError("COLLECTOR_NOT_FOUND", 
			fmt.Sprintf("collector %s not found in registry", collectorName))
	}
	
	// Filter regions if scheduler has enabled regions configured
	if len(s.config.EnabledRegions) > 0 {
		filteredRegions := []string{}
		enabledMap := make(map[string]bool)
		for _, region := range s.config.EnabledRegions {
			enabledMap[region] = true
		}
		
		for _, region := range regions {
			if enabledMap[region] {
				filteredRegions = append(filteredRegions, region)
			}
		}
		regions = filteredRegions
	}
	
	// Create jobs for each region
	for _, region := range regions {
		jobID := fmt.Sprintf("%s-%s", collectorName, region)
		
		job := &ScheduledJob{
			ID:            jobID,
			CollectorName: collectorName,
			Region:        region,
			Interval:      interval,
			NextRun:       time.Now().Add(100 * time.Millisecond), // Start soon
			Enabled:       true,
		}
		
		s.jobs[jobID] = job
		s.logger.Info("Scheduled collector job",
			logger.String("job_id", jobID),
			logger.String("collector", collectorName),
			logger.String("region", region),
			logger.Duration("interval", interval))
	}
	
	return nil
}

// UnscheduleCollector removes a collector from the schedule
func (s *MetricScheduler) UnscheduleCollector(collectorName string, region string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	jobID := fmt.Sprintf("%s-%s", collectorName, region)
	
	if _, exists := s.jobs[jobID]; exists {
		// Cancel if currently running
		if cancel, running := s.activeJobs[jobID]; running {
			cancel()
			delete(s.activeJobs, jobID)
		}
		
		delete(s.jobs, jobID)
		s.logger.Info("Unscheduled collector job",
			logger.String("job_id", jobID),
			logger.String("collector", collectorName),
			logger.String("region", region))
		
		return nil
	}
	
	return errors.NewValidationError("JOB_NOT_FOUND",
		fmt.Sprintf("job %s not found", jobID))
}

// GetScheduledJobs returns all currently scheduled jobs
func (s *MetricScheduler) GetScheduledJobs() []ScheduledJob {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	jobs := make([]ScheduledJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, *job)
	}
	
	return jobs
}

// GetInfo returns current scheduler status and statistics
func (s *MetricScheduler) GetInfo() Info {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return Info{
		Status:        s.status,
		StartTime:     s.startTime,
		JobCount:      len(s.jobs),
		ActiveJobs:    len(s.activeJobs),
		CompletedJobs: s.completedJobs,
		FailedJobs:    s.failedJobs,
		LastTickTime:  s.lastTickTime,
	}
}

// Health returns the health status of the scheduler
func (s *MetricScheduler) Health() error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	switch s.status {
	case StatusRunning:
		// Check if scheduler is ticking
		if s.lastTickTime != nil && time.Since(*s.lastTickTime) > 2*s.config.TickInterval {
			return errors.NewValidationError("SCHEDULER_NOT_TICKING",
				"scheduler has not ticked recently")
		}
		return nil
	case StatusError:
		return errors.NewValidationError("SCHEDULER_ERROR",
			"scheduler is in error state")
	case StatusStopped:
		return errors.NewValidationError("SCHEDULER_STOPPED",
			"scheduler is stopped")
	default:
		return errors.NewValidationError("SCHEDULER_NOT_READY",
			"scheduler is not ready")
	}
}

// validateConfig validates the scheduler configuration
func (s *MetricScheduler) validateConfig() *errors.Error {
	if s.config.TickInterval <= 0 {
		return errors.NewConfigError("INVALID_TICK_INTERVAL",
			"tick interval must be positive")
	}
	
	if s.config.MaxConcurrentJobs <= 0 {
		return errors.NewConfigError("INVALID_MAX_CONCURRENT_JOBS",
			"max concurrent jobs must be positive")
	}
	
	if s.config.JobTimeout <= 0 {
		return errors.NewConfigError("INVALID_JOB_TIMEOUT",
			"job timeout must be positive")
	}
	
	return nil
}

// run is the main scheduler loop
func (s *MetricScheduler) run(ctx context.Context) {
	defer close(s.doneCh)
	
	ticker := time.NewTicker(s.config.TickInterval)
	defer ticker.Stop()
	
	s.logger.Debug("Scheduler main loop started")
	
	for {
		select {
		case <-ctx.Done():
			s.logger.Debug("Scheduler context cancelled")
			return
		case <-s.stopCh:
			s.logger.Debug("Scheduler stop signal received")
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

// tick checks for jobs that need to run and executes them
func (s *MetricScheduler) tick(ctx context.Context) {
	now := time.Now()
	
	s.mu.Lock()
	s.lastTickTime = &now
	jobsToRun := []*ScheduledJob{}
	
	// Find jobs that need to run
	for _, job := range s.jobs {
		if job.Enabled && now.After(job.NextRun) {
			// Check if job is already running
			if _, running := s.activeJobs[job.ID]; !running {
				jobsToRun = append(jobsToRun, job)
			}
		}
	}
	s.mu.Unlock()
	
	// Execute jobs
	for _, job := range jobsToRun {
		select {
		case s.jobSemaphore <- struct{}{}: // Acquire semaphore
			go s.executeJob(ctx, job)
		default:
			// No available slots, skip this job
			s.logger.Warn("Skipping job execution, max concurrent jobs reached",
				logger.String("job_id", job.ID),
				logger.Int("max_concurrent", s.config.MaxConcurrentJobs))
		}
	}
}

// executeJob runs a single job
func (s *MetricScheduler) executeJob(ctx context.Context, job *ScheduledJob) {
	defer func() { <-s.jobSemaphore }() // Release semaphore
	
	// Create job context with timeout
	jobCtx, cancel := context.WithTimeout(ctx, s.config.JobTimeout)
	defer cancel()
	
	// Track active job
	s.mu.Lock()
	s.activeJobs[job.ID] = cancel
	s.mu.Unlock()
	
	// Clean up active job tracking
	defer func() {
		s.mu.Lock()
		delete(s.activeJobs, job.ID)
		s.mu.Unlock()
	}()
	
	s.logger.Debug("Executing job", 
		logger.String("job_id", job.ID),
		logger.String("collector", job.CollectorName),
		logger.String("region", job.Region))
	
	// Execute the job
	result := s.executor.ExecuteJob(jobCtx, job)
	
	// Update job state
	s.mu.Lock()
	now := time.Now()
	job.LastRun = &now
	job.NextRun = now.Add(job.Interval)
	job.LastResult = result
	
	if result.Error != nil {
		s.failedJobs++
		s.logger.Warn("Job execution failed",
			logger.String("job_id", job.ID),
			logger.String("error", result.Error.Error()))
		
		// Process error
		if err := s.processor.ProcessError(jobCtx, job, result.Error); err != nil {
			s.logger.Error("Failed to process job error",
				logger.String("job_id", job.ID),
				logger.String("process_error", err.Error()))
		}
	} else {
		s.completedJobs++
		s.logger.Debug("Job execution completed",
			logger.String("job_id", job.ID),
			logger.Int("metric_count", len(result.Metrics)),
			logger.Duration("duration", result.Duration))
		
		// Process result
		if err := s.processor.ProcessResult(jobCtx, job, result); err != nil {
			s.logger.Error("Failed to process job result",
				logger.String("job_id", job.ID),
				logger.String("process_error", err.Error()))
		}
	}
	s.mu.Unlock()
}