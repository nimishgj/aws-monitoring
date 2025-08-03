package scheduler

import (
	"context"
	"sync"
	"testing"
	"time"

	"aws-monitoring/internal/collectors"
	"aws-monitoring/pkg/errors"
	"aws-monitoring/pkg/logger"
)

// Mock implementations for testing

type mockCollector struct {
	name        string
	description string
	collectFunc func(ctx context.Context, region string) *collectors.CollectionResult
}

func (m *mockCollector) Name() string        { return m.name }
func (m *mockCollector) Description() string { return m.description }
func (m *mockCollector) Start(_ context.Context) error { return nil }
func (m *mockCollector) Stop(_ context.Context) error  { return nil }
func (m *mockCollector) Health() error { return nil }
func (m *mockCollector) Info() collectors.CollectorInfo {
	return collectors.CollectorInfo{
		Name:        m.name,
		Description: m.description,
		Status:      collectors.StatusRunning,
	}
}

func (m *mockCollector) Collect(ctx context.Context, region string) *collectors.CollectionResult {
	if m.collectFunc != nil {
		return m.collectFunc(ctx, region)
	}
	return &collectors.CollectionResult{
		CollectorName:  m.name,
		Region:         region,
		CollectionTime: time.Now(),
		Metrics: []collectors.MetricData{
			{
				Name:      "test_metric",
				Value:     1.0,
				Unit:      "Count",
				Timestamp: time.Now(),
				Labels:    map[string]string{"region": region},
			},
		},
	}
}

type mockRegistry struct {
	collectors map[string]collectors.MetricCollector
	mu         sync.RWMutex
}

func newMockRegistry() *mockRegistry {
	return &mockRegistry{
		collectors: make(map[string]collectors.MetricCollector),
	}
}

func (r *mockRegistry) Register(collector collectors.MetricCollector) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.collectors[collector.Name()] = collector
	return nil
}

func (r *mockRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.collectors, name)
	return nil
}

func (r *mockRegistry) Get(name string) (collectors.MetricCollector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	collector, exists := r.collectors[name]
	return collector, exists
}

func (r *mockRegistry) List() []collectors.MetricCollector {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]collectors.MetricCollector, 0, len(r.collectors))
	for _, collector := range r.collectors {
		result = append(result, collector)
	}
	return result
}

func (r *mockRegistry) Start(_ context.Context) error { return nil }
func (r *mockRegistry) Stop(_ context.Context) error  { return nil }
func (r *mockRegistry) Status() map[string]collectors.CollectorInfo {
	return make(map[string]collectors.CollectorInfo)
}

type mockJobProcessor struct {
	results []ProcessedResult
	errors  []ProcessedError
	mu      sync.Mutex
}

type ProcessedResult struct {
	Job    *ScheduledJob
	Result *collectors.CollectionResult
}

type ProcessedError struct {
	Job   *ScheduledJob
	Error *errors.Error
}

func newMockJobProcessor() *mockJobProcessor {
	return &mockJobProcessor{
		results: []ProcessedResult{},
		errors:  []ProcessedError{},
	}
}

func (p *mockJobProcessor) ProcessResult(_ context.Context, job *ScheduledJob, result *collectors.CollectionResult) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.results = append(p.results, ProcessedResult{Job: job, Result: result})
	return nil
}

func (p *mockJobProcessor) ProcessError(_ context.Context, job *ScheduledJob, err *errors.Error) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.errors = append(p.errors, ProcessedError{Job: job, Error: err})
	return nil
}

func (p *mockJobProcessor) GetResults() []ProcessedResult {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]ProcessedResult{}, p.results...)
}

func (p *mockJobProcessor) GetErrors() []ProcessedError {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]ProcessedError{}, p.errors...)
}

func setupTest() (*MetricScheduler, *mockRegistry, *mockJobProcessor, *logger.Logger) {
	loggerConfig := logger.Config{
		Level:  "debug",
		Format: "json",
	}
	log, _ := logger.NewLogger(loggerConfig)
	
	registry := newMockRegistry()
	processor := newMockJobProcessor()
	
	config := Config{
		TickInterval:      100 * time.Millisecond,
		MaxConcurrentJobs: 2,
		JobTimeout:        5 * time.Second,
	}
	
	scheduler := NewMetricScheduler(config, registry, processor, log).(*MetricScheduler)
	
	return scheduler, registry, processor, log
}

func TestNewMetricScheduler(t *testing.T) {
	scheduler, _, _, _ := setupTest()
	
	if scheduler.status != StatusStopped {
		t.Errorf("Expected initial status stopped, got %s", scheduler.status)
	}
	
	if len(scheduler.jobs) != 0 {
		t.Errorf("Expected no initial jobs, got %d", len(scheduler.jobs))
	}
	
	info := scheduler.GetInfo()
	if info.Status != StatusStopped {
		t.Errorf("Expected info status stopped, got %s", info.Status)
	}
}

func TestSchedulerStartStop(t *testing.T) {
	scheduler, _, _, _ := setupTest()
	ctx := context.Background()
	
	// Test start
	err := scheduler.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	
	info := scheduler.GetInfo()
	if info.Status != StatusRunning {
		t.Errorf("Expected status running, got %s", info.Status)
	}
	
	if info.StartTime == nil {
		t.Error("Expected start time to be set")
	}
	
	// Test start when already running
	err = scheduler.Start(ctx)
	if err != nil {
		t.Errorf("Expected no error when starting already running scheduler, got: %v", err)
	}
	
	// Test stop
	stopCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	
	err = scheduler.Stop(stopCtx)
	if err != nil {
		t.Errorf("Failed to stop scheduler: %v", err)
	}
	
	info = scheduler.GetInfo()
	if info.Status != StatusStopped {
		t.Errorf("Expected status stopped, got %s", info.Status)
	}
}

func TestScheduleCollector(t *testing.T) {
	scheduler, registry, _, _ := setupTest()
	
	// Register a test collector
	collector := &mockCollector{name: "test-collector", description: "Test collector"}
	err := registry.Register(collector)
	if err != nil {
		t.Fatalf("Failed to register collector: %v", err)
	}
	
	// Schedule the collector
	err = scheduler.ScheduleCollector("test-collector", []string{"us-east-1", "us-west-2"}, 5*time.Minute)
	if err != nil {
		t.Fatalf("Failed to schedule collector: %v", err)
	}
	
	jobs := scheduler.GetScheduledJobs()
	if len(jobs) != 2 {
		t.Errorf("Expected 2 jobs, got %d", len(jobs))
	}
	
	// Verify job details
	for _, job := range jobs {
		if job.CollectorName != "test-collector" {
			t.Errorf("Expected collector name 'test-collector', got %s", job.CollectorName)
		}
		
		if job.Interval != 5*time.Minute {
			t.Errorf("Expected interval 5m, got %v", job.Interval)
		}
		
		if !job.Enabled {
			t.Error("Expected job to be enabled")
		}
		
		if job.Region != "us-east-1" && job.Region != "us-west-2" {
			t.Errorf("Unexpected region: %s", job.Region)
		}
	}
}

func TestScheduleNonExistentCollector(t *testing.T) {
	scheduler, _, _, _ := setupTest()
	
	err := scheduler.ScheduleCollector("non-existent", []string{"us-east-1"}, 5*time.Minute)
	if err == nil {
		t.Error("Expected error when scheduling non-existent collector")
	}
	
	if !errors.IsType(err, errors.ErrorTypeValidation) {
		t.Errorf("Expected validation error, got %T", err)
	}
}

func TestUnscheduleCollector(t *testing.T) {
	scheduler, registry, _, _ := setupTest()
	
	// Register and schedule a collector
	collector := &mockCollector{name: "test-collector", description: "Test collector"}
	err := registry.Register(collector)
	if err != nil {
		t.Fatalf("Failed to register collector: %v", err)
	}
	err = scheduler.ScheduleCollector("test-collector", []string{"us-east-1"}, 5*time.Minute)
	if err != nil {
		t.Fatalf("Failed to schedule collector: %v", err)
	}
	
	// Verify job exists
	jobs := scheduler.GetScheduledJobs()
	if len(jobs) != 1 {
		t.Fatalf("Expected 1 job, got %d", len(jobs))
	}
	
	// Unschedule
	err = scheduler.UnscheduleCollector("test-collector", "us-east-1")
	if err != nil {
		t.Fatalf("Failed to unschedule collector: %v", err)
	}
	
	// Verify job removed
	jobs = scheduler.GetScheduledJobs()
	if len(jobs) != 0 {
		t.Errorf("Expected no jobs, got %d", len(jobs))
	}
}

func TestJobExecution(t *testing.T) {
	scheduler, registry, processor, _ := setupTest()
	
	// Register a test collector
	collector := &mockCollector{
		name:        "test-collector",
		description: "Test collector",
	}
	err := registry.Register(collector)
	if err != nil {
		t.Fatalf("Failed to register collector: %v", err)
	}
	
	// Start scheduler
	ctx := context.Background()
	err = scheduler.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer func() {
		if stopErr := scheduler.Stop(ctx); stopErr != nil {
			t.Errorf("Failed to stop scheduler: %v", stopErr)
		}
	}()
	
	// Schedule a job with short interval
	err = scheduler.ScheduleCollector("test-collector", []string{"us-east-1"}, 200*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to schedule collector: %v", err)
	}
	
	// Wait for job execution
	time.Sleep(800 * time.Millisecond)
	
	// Check that results were processed
	results := processor.GetResults()
	if len(results) == 0 {
		t.Error("Expected at least one result to be processed")
	}
	
	// Verify result details
	if len(results) > 0 {
		result := results[0]
		if result.Job.CollectorName != "test-collector" {
			t.Errorf("Expected collector name 'test-collector', got %s", result.Job.CollectorName)
		}
		
		if result.Job.Region != "us-east-1" {
			t.Errorf("Expected region 'us-east-1', got %s", result.Job.Region)
		}
		
		if len(result.Result.Metrics) != 1 {
			t.Errorf("Expected 1 metric, got %d", len(result.Result.Metrics))
		}
	}
}

func TestJobExecutionWithError(t *testing.T) {
	scheduler, registry, processor, _ := setupTest()
	
	// Register a collector that returns errors
	collector := &mockCollector{
		name:        "error-collector",
		description: "Error collector",
		collectFunc: func(_ context.Context, region string) *collectors.CollectionResult {
			return &collectors.CollectionResult{
				CollectorName:  "error-collector",
				Region:         region,
				CollectionTime: time.Now(),
				Metrics:        []collectors.MetricData{},
				Error:          errors.NewNetworkError("CONNECTION_ERROR", "connection failed"),
			}
		},
	}
	err := registry.Register(collector)
	if err != nil {
		t.Fatalf("Failed to register collector: %v", err)
	}
	
	// Start scheduler
	ctx := context.Background()
	err = scheduler.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer func() {
		if stopErr := scheduler.Stop(ctx); stopErr != nil {
			t.Errorf("Failed to stop scheduler: %v", stopErr)
		}
	}()
	
	// Schedule a job
	err = scheduler.ScheduleCollector("error-collector", []string{"us-east-1"}, 200*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to schedule collector: %v", err)
	}
	
	// Wait for job execution
	time.Sleep(800 * time.Millisecond)
	
	// Check that errors were processed
	errors := processor.GetErrors()
	if len(errors) == 0 {
		t.Error("Expected at least one error to be processed")
	}
	
	// Verify error details
	if len(errors) > 0 {
		processedError := errors[0]
		if processedError.Job.CollectorName != "error-collector" {
			t.Errorf("Expected collector name 'error-collector', got %s", processedError.Job.CollectorName)
		}
		
		if processedError.Error.Code != "CONNECTION_ERROR" {
			t.Errorf("Expected error code 'CONNECTION_ERROR', got %s", processedError.Error.Code)
		}
	}
}

func TestSchedulerHealth(t *testing.T) {
	scheduler, _, _, _ := setupTest()
	
	// Test health when stopped
	health := scheduler.Health()
	if health == nil {
		t.Error("Expected health error when scheduler is stopped")
	}
	
	// Start scheduler
	ctx := context.Background()
	err := scheduler.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}
	defer func() {
		if stopErr := scheduler.Stop(ctx); stopErr != nil {
			t.Errorf("Failed to stop scheduler: %v", stopErr)
		}
	}()
	
	// Wait a bit for first tick
	time.Sleep(200 * time.Millisecond)
	
	// Test health when running
	health = scheduler.Health()
	if health != nil {
		t.Errorf("Expected no health error when running, got: %v", health)
	}
}

func TestSchedulerInfo(t *testing.T) {
	scheduler, registry, _, _ := setupTest()
	
	// Register a collector and schedule it
	collector := &mockCollector{name: "test-collector", description: "Test collector"}
	err := registry.Register(collector)
	if err != nil {
		t.Fatalf("Failed to register collector: %v", err)
	}
	err = scheduler.ScheduleCollector("test-collector", []string{"us-east-1", "us-west-2"}, 5*time.Minute)
	if err != nil {
		t.Fatalf("Failed to schedule collector: %v", err)
	}
	
	info := scheduler.GetInfo()
	
	if info.JobCount != 2 {
		t.Errorf("Expected job count 2, got %d", info.JobCount)
	}
	
	if info.ActiveJobs != 0 {
		t.Errorf("Expected active jobs 0, got %d", info.ActiveJobs)
	}
	
	if info.CompletedJobs != 0 {
		t.Errorf("Expected completed jobs 0, got %d", info.CompletedJobs)
	}
	
	if info.FailedJobs != 0 {
		t.Errorf("Expected failed jobs 0, got %d", info.FailedJobs)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	if config.TickInterval != 30*time.Second {
		t.Errorf("Expected tick interval 30s, got %v", config.TickInterval)
	}
	
	if config.MaxConcurrentJobs != 10 {
		t.Errorf("Expected max concurrent jobs 10, got %d", config.MaxConcurrentJobs)
	}
	
	if config.JobTimeout != 5*time.Minute {
		t.Errorf("Expected job timeout 5m, got %v", config.JobTimeout)
	}
}