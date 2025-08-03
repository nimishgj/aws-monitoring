package health

import (
	"context"
	"fmt"
	"sync"
	"time"

	"aws-monitoring/pkg/logger"
)

// Manager manages health checks and provides aggregated health status
type Manager struct {
	checkers   map[string]Checker
	results    map[string]CheckResult
	startTime  time.Time
	version    string
	service    string
	logger     *logger.Logger
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	running    bool
}

// NewManager creates a new health check manager
func NewManager(service, version string, log *logger.Logger) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		checkers:  make(map[string]Checker),
		results:   make(map[string]CheckResult),
		startTime: time.Now(),
		version:   version,
		service:   service,
		logger:    log.WithComponent("health"),
		ctx:       ctx,
		cancel:    cancel,
		running:   false,
	}
}

// RegisterChecker adds a health checker to the manager
func (m *Manager) RegisterChecker(checker Checker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	name := checker.Name()
	m.checkers[name] = checker
	m.logger.Info("Health checker registered", logger.String("checker", name))
}

// UnregisterChecker removes a health checker from the manager
func (m *Manager) UnregisterChecker(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.checkers[name]; exists {
		delete(m.checkers, name)
		delete(m.results, name)
		m.logger.Info("Health checker unregistered", logger.String("checker", name))
	}
}

// RunChecks executes all registered health checks
func (m *Manager) RunChecks(ctx context.Context) {
	m.mu.Lock()
	checkers := make(map[string]Checker)
	for name, checker := range m.checkers {
		checkers[name] = checker
	}
	m.mu.Unlock()

	if len(checkers) == 0 {
		m.logger.Debug("No health checkers registered")
		return
	}

	var wg sync.WaitGroup
	resultsChan := make(chan CheckResult, len(checkers))

	// Run all checks concurrently
	for _, checker := range checkers {
		wg.Add(1)
		go func(c Checker) {
			defer wg.Done()
			start := time.Now()
			
			checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			
			result := c.Check(checkCtx)
			result.Duration = time.Since(start)
			result.LastChecked = start
			
			select {
			case resultsChan <- result:
			case <-ctx.Done():
				return
			}
		}(checker)
	}

	// Wait for all checks to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	m.mu.Lock()
	for result := range resultsChan {
		m.results[result.Name] = result
		m.logger.Debug("Health check completed",
			logger.String("checker", result.Name),
			logger.String("status", string(result.Status)),
			logger.Duration("duration", result.Duration))
	}
	m.mu.Unlock()
}

// GetHealth returns the current overall health status
func (m *Manager) GetHealth() OverallHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()

	health := OverallHealth{
		Timestamp:   time.Now(),
		Uptime:      time.Since(m.startTime),
		Version:     m.version,
		ServiceName: m.service,
		Checks:      make(map[string]CheckResult),
	}

	// Copy current results
	for name, result := range m.results {
		health.Checks[name] = result
	}

	// Determine overall status
	health.Status, health.Summary = m.aggregateStatus(health.Checks)

	return health
}

// aggregateStatus determines the overall health status from individual checks
func (m *Manager) aggregateStatus(checks map[string]CheckResult) (Status, string) {
	if len(checks) == 0 {
		return StatusUnknown, "No health checks configured"
	}

	healthyCount := 0
	unhealthyCount := 0
	degradedCount := 0
	unknownCount := 0
	totalChecks := len(checks)

	for _, result := range checks {
		switch result.Status {
		case StatusHealthy:
			healthyCount++
		case StatusUnhealthy:
			unhealthyCount++
		case StatusDegraded:
			degradedCount++
		case StatusUnknown:
			unknownCount++
		}
	}

	// Determine overall status based on individual check results
	if unhealthyCount > 0 {
		return StatusUnhealthy, generateSummary(healthyCount, unhealthyCount, degradedCount, unknownCount, totalChecks)
	}
	
	if degradedCount > 0 {
		return StatusDegraded, generateSummary(healthyCount, unhealthyCount, degradedCount, unknownCount, totalChecks)
	}
	
	if unknownCount == totalChecks {
		return StatusUnknown, generateSummary(healthyCount, unhealthyCount, degradedCount, unknownCount, totalChecks)
	}
	
	if healthyCount == totalChecks {
		return StatusHealthy, generateSummary(healthyCount, unhealthyCount, degradedCount, unknownCount, totalChecks)
	}

	// Mixed status with some unknown
	if healthyCount > 0 {
		return StatusDegraded, generateSummary(healthyCount, unhealthyCount, degradedCount, unknownCount, totalChecks)
	}

	return StatusUnknown, generateSummary(healthyCount, unhealthyCount, degradedCount, unknownCount, totalChecks)
}

// generateSummary creates a human-readable summary of health check results
func generateSummary(healthy, unhealthy, degraded, unknown, total int) string {
	if total == 1 {
		if healthy == 1 {
			return "All systems operational"
		}
		if unhealthy == 1 {
			return "System experiencing issues"
		}
		if degraded == 1 {
			return "System performance degraded"
		}
		return "System status unknown"
	}

	if unhealthy > 0 {
		return fmt.Sprintf("%d of %d checks failing", unhealthy, total)
	}
	
	if degraded > 0 {
		return fmt.Sprintf("%d of %d checks degraded", degraded, total)
	}
	
	if unknown > 0 && healthy > 0 {
		return fmt.Sprintf("%d of %d checks healthy, %d unknown", healthy, total, unknown)
	}
	
	if healthy == total {
		return "All systems operational"
	}

	return fmt.Sprintf("%d checks total: %d healthy, %d degraded, %d unhealthy, %d unknown",
		total, healthy, degraded, unhealthy, unknown)
}

// Start begins periodic health checking
func (m *Manager) Start(interval time.Duration) {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	m.mu.Unlock()

	m.logger.Info("Starting health check manager", logger.Duration("interval", interval))

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Run initial check
		m.RunChecks(m.ctx)

		for {
			select {
			case <-ticker.C:
				m.RunChecks(m.ctx)
			case <-m.ctx.Done():
				m.logger.Info("Health check manager stopped")
				return
			}
		}
	}()
}

// Stop stops the health check manager
func (m *Manager) Stop() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	m.running = false
	m.mu.Unlock()

	m.cancel()
	m.logger.Info("Health check manager stopping")
}

// IsRunning returns whether the health check manager is currently running
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}