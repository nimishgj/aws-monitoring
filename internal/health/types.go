// Package health provides health check functionality for the AWS monitoring application.
package health

import (
	"context"
	"time"
)

// Status represents the health status of a component
type Status string

const (
	// StatusHealthy indicates the component is functioning properly
	StatusHealthy Status = "healthy"
	// StatusUnhealthy indicates the component is not functioning properly
	StatusUnhealthy Status = "unhealthy"
	// StatusDegraded indicates the component is functioning but with issues
	StatusDegraded Status = "degraded"
	// StatusUnknown indicates the component status cannot be determined
	StatusUnknown Status = "unknown"
)

// CheckResult represents the result of a health check
type CheckResult struct {
	// Name is the identifier for this health check
	Name string `json:"name"`
	// Status is the current health status
	Status Status `json:"status"`
	// Message provides additional context about the status
	Message string `json:"message,omitempty"`
	// LastChecked is when this check was last performed
	LastChecked time.Time `json:"last_checked"`
	// Duration is how long the check took to complete
	Duration time.Duration `json:"duration"`
	// Error contains error details if the check failed
	Error string `json:"error,omitempty"`
	// Metadata contains additional check-specific information
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Checker defines the interface for health check implementations
type Checker interface {
	// Check performs the health check and returns the result
	Check(ctx context.Context) CheckResult
	// Name returns the unique identifier for this checker
	Name() string
}

// OverallHealth represents the overall health status of the application
type OverallHealth struct {
	// Status is the aggregated health status
	Status Status `json:"status"`
	// Timestamp when this health summary was generated
	Timestamp time.Time `json:"timestamp"`
	// Uptime since the application started
	Uptime time.Duration `json:"uptime"`
	// Version information
	Version string `json:"version,omitempty"`
	// ServiceName identifies the service
	ServiceName string `json:"service_name"`
	// Checks contains individual health check results
	Checks map[string]CheckResult `json:"checks"`
	// Summary provides a human-readable overview
	Summary string `json:"summary,omitempty"`
}

// CheckerConfig defines configuration for health checkers
type CheckerConfig struct {
	// Enabled determines if this checker should be active
	Enabled bool `json:"enabled"`
	// Interval defines how often to run this check
	Interval time.Duration `json:"interval"`
	// Timeout defines the maximum time to wait for a check
	Timeout time.Duration `json:"timeout"`
	// Retries defines how many times to retry a failed check
	Retries int `json:"retries"`
	// RetryDelay defines the delay between retries
	RetryDelay time.Duration `json:"retry_delay"`
}

// DefaultCheckerConfig returns sensible defaults for health checker configuration
func DefaultCheckerConfig() CheckerConfig {
	return CheckerConfig{
		Enabled:    true,
		Interval:   30 * time.Second,
		Timeout:    10 * time.Second,
		Retries:    2,
		RetryDelay: 1 * time.Second,
	}
}