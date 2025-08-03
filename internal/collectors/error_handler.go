package collectors

import (
	"math"
	"strings"
	"time"

	"aws-monitoring/pkg/errors"
	"aws-monitoring/pkg/logger"
)

// DefaultErrorHandler provides a default implementation of ErrorHandler
type DefaultErrorHandler struct {
	logger     *logger.Logger
	maxRetries int
	baseDelay  time.Duration
}

// NewDefaultErrorHandler creates a new default error handler
func NewDefaultErrorHandler(log *logger.Logger) ErrorHandler {
	return &DefaultErrorHandler{
		logger:     log.WithComponent("error-handler"),
		maxRetries: 3,
		baseDelay:  time.Second,
	}
}

// HandleError processes an error from a collector
func (eh *DefaultErrorHandler) HandleError(collectorName string, err *errors.Error) {
	if err == nil {
		return
	}
	
	// Log the error with appropriate level based on severity
	switch err.Severity {
	case errors.SeverityLow:
		eh.logger.Debug("Collector error",
			logger.String("collector", collectorName),
			logger.String("error_type", string(err.Type)),
			logger.String("error_code", err.Code),
			logger.String("error", err.Error()))
		
	case errors.SeverityMedium:
		eh.logger.Info("Collector error",
			logger.String("collector", collectorName),
			logger.String("error_type", string(err.Type)),
			logger.String("error_code", err.Code),
			logger.String("region", err.Region),
			logger.String("error", err.Error()))
		
	case errors.SeverityHigh:
		eh.logger.Warn("Collector error",
			logger.String("collector", collectorName),
			logger.String("error_type", string(err.Type)),
			logger.String("error_code", err.Code),
			logger.String("operation", err.Operation),
			logger.String("region", err.Region),
			logger.String("service", err.Service),
			logger.String("error", err.Error()))
		
	case errors.SeverityCritical:
		eh.logger.Error("Critical collector error",
			logger.String("collector", collectorName),
			logger.String("error_type", string(err.Type)),
			logger.String("error_code", err.Code),
			logger.String("operation", err.Operation),
			logger.String("region", err.Region),
			logger.String("service", err.Service),
			logger.String("error", err.Error()),
			logger.Any("stack_trace", err.StackTrace))
	}
	
	// TODO: Here you could add additional error handling like:
	// - Sending alerts for critical errors
	// - Recording error metrics
	// - Updating health status
	// - Implementing circuit breaker logic
}

// ShouldRetry determines if an operation should be retried
func (eh *DefaultErrorHandler) ShouldRetry(err *errors.Error, attempt int) bool {
	if err == nil {
		return false
	}
	
	// Check if we've exceeded max retries
	if attempt >= eh.maxRetries {
		return false
	}
	
	// Check if the error is explicitly marked as retryable
	if err.Retryable {
		return true
	}
	
	// Determine retryability based on error type and code
	switch err.Type {
	case errors.ErrorTypeAWS:
		return eh.shouldRetryAWSError(err)
	case errors.ErrorTypeNetwork:
		return true // Network errors are generally retryable
	case errors.ErrorTypeTimeout:
		return true // Timeout errors are retryable
	case errors.ErrorTypeRateLimit:
		return true // Rate limit errors should be retried with backoff
	case errors.ErrorTypePermission:
		return false // Permission errors are not retryable
	case errors.ErrorTypeConfig:
		return false // Config errors are not retryable
	case errors.ErrorTypeValidation:
		return false // Validation errors are not retryable
	case errors.ErrorTypeInternal:
		return eh.shouldRetryInternalError(err)
	default:
		return false
	}
}

// GetRetryDelay returns how long to wait before retrying
func (eh *DefaultErrorHandler) GetRetryDelay(err *errors.Error, attempt int) time.Duration {
	if err == nil {
		return eh.baseDelay
	}
	
	// Check if the error specifies a retry delay (for rate limits)
	if retryAfter := errors.GetRetryAfter(err); retryAfter != nil {
		return *retryAfter
	}
	
	// Use exponential backoff with jitter
	delay := time.Duration(float64(eh.baseDelay) * math.Pow(2, float64(attempt)))
	
	// Add jitter (±25%)
	jitter := float64(delay) * 0.25
	jitterAdjustment := time.Duration((2*time.Now().UnixNano()%int64(jitter)) - int64(jitter))
	delay += jitterAdjustment
	
	// Cap the delay at 30 seconds
	maxDelay := 30 * time.Second
	if delay > maxDelay {
		delay = maxDelay
	}
	
	// Ensure minimum delay
	minDelay := 100 * time.Millisecond
	if delay < minDelay {
		delay = minDelay
	}
	
	return delay
}

// shouldRetryAWSError determines if an AWS error should be retried
func (eh *DefaultErrorHandler) shouldRetryAWSError(err *errors.Error) bool {
	// AWS error codes that are typically retryable
	retryableAWSCodes := []string{
		"InternalError",
		"InternalFailure", 
		"ServiceUnavailable",
		"Throttling",
		"ThrottlingException",
		"RequestLimitExceeded",
		"RequestTimeout",
		"RequestTimeoutException",
		"PriorRequestNotComplete",
		"ConnectionError",
		"NetworkError",
		"DNSError",
		"TimeoutError",
		"RequestExpired",
		"ServiceTemporarilyUnavailable",
	}
	
	for _, code := range retryableAWSCodes {
		if strings.Contains(err.Code, code) || strings.Contains(err.Message, code) {
			return true
		}
	}
	
	// Check for specific error message patterns
	retryablePatterns := []string{
		"connection reset",
		"connection timeout",
		"connection refused",
		"no such host",
		"network is unreachable",
		"temporary failure",
		"service temporarily unavailable",
		"internal server error",
		"bad gateway",
		"gateway timeout",
	}
	
	lowerMessage := strings.ToLower(err.Message)
	for _, pattern := range retryablePatterns {
		if strings.Contains(lowerMessage, pattern) {
			return true
		}
	}
	
	return false
}

// shouldRetryInternalError determines if an internal error should be retried
func (eh *DefaultErrorHandler) shouldRetryInternalError(err *errors.Error) bool {
	// Internal errors that might be retryable
	retryableInternalCodes := []string{
		"CONTEXT_CANCELLED",
		"TIMEOUT",
		"NETWORK_ERROR",
		"TEMPORARY_FAILURE",
	}
	
	for _, code := range retryableInternalCodes {
		if err.Code == code {
			return true
		}
	}
	
	return false
}

// CircuitBreakerErrorHandler implements circuit breaker pattern for error handling
type CircuitBreakerErrorHandler struct {
	*DefaultErrorHandler
	
	// Circuit breaker state
	state          CircuitBreakerState
	failureCount   int
	lastFailure    time.Time
	successCount   int
	failureThreshold int
	timeout        time.Duration
	recoveryThreshold int
}

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState string

const (
	// CircuitBreakerClosed allows all requests through
	CircuitBreakerClosed CircuitBreakerState = "closed"
	// CircuitBreakerOpen blocks all requests
	CircuitBreakerOpen CircuitBreakerState = "open"
	// CircuitBreakerHalfOpen allows limited requests to test recovery
	CircuitBreakerHalfOpen CircuitBreakerState = "half_open"
)

// NewCircuitBreakerErrorHandler creates a circuit breaker error handler
func NewCircuitBreakerErrorHandler(log *logger.Logger) ErrorHandler {
	return &CircuitBreakerErrorHandler{
		DefaultErrorHandler: &DefaultErrorHandler{
			logger:     log.WithComponent("circuit-breaker-error-handler"),
			maxRetries: 3,
			baseDelay:  time.Second,
		},
		state:             CircuitBreakerClosed,
		failureThreshold:  5,  // Open circuit after 5 failures
		timeout:           60 * time.Second, // Stay open for 60 seconds
		recoveryThreshold: 3,  // Need 3 successes to close circuit
	}
}

// ShouldRetry implements circuit breaker logic
func (cb *CircuitBreakerErrorHandler) ShouldRetry(err *errors.Error, attempt int) bool {
	// If circuit is open, don't retry
	if cb.isCircuitOpen() {
		return false
	}
	
	// Use default retry logic if circuit is closed or half-open
	return cb.DefaultErrorHandler.ShouldRetry(err, attempt)
}

// HandleError implements circuit breaker error handling
func (cb *CircuitBreakerErrorHandler) HandleError(collectorName string, err *errors.Error) {
	// Call default error handling first
	cb.DefaultErrorHandler.HandleError(collectorName, err)
	
	// Update circuit breaker state
	cb.recordFailure()
}

// RecordSuccess should be called when an operation succeeds
func (cb *CircuitBreakerErrorHandler) RecordSuccess() {
	switch cb.state {
	case CircuitBreakerHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.recoveryThreshold {
			cb.closeCircuit()
		}
	case CircuitBreakerOpen:
		// Success while open (shouldn't happen)
		cb.closeCircuit()
	case CircuitBreakerClosed:
		// Reset failure count on success
		cb.failureCount = 0
	}
}

func (cb *CircuitBreakerErrorHandler) recordFailure() {
	cb.lastFailure = time.Now()
	
	switch cb.state {
	case CircuitBreakerClosed:
		cb.failureCount++
		if cb.failureCount >= cb.failureThreshold {
			cb.openCircuit()
		}
	case CircuitBreakerHalfOpen:
		cb.openCircuit()
	case CircuitBreakerOpen:
		// Already open, just update timestamp
	}
}

func (cb *CircuitBreakerErrorHandler) isCircuitOpen() bool {
	if cb.state == CircuitBreakerOpen {
		// Check if timeout has passed
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.transitionToHalfOpen()
			return false
		}
		return true
	}
	return false
}

func (cb *CircuitBreakerErrorHandler) openCircuit() {
	cb.state = CircuitBreakerOpen
	cb.logger.Warn("Circuit breaker opened", 
		logger.Int("failure_count", cb.failureCount),
		logger.Duration("timeout", cb.timeout))
}

func (cb *CircuitBreakerErrorHandler) closeCircuit() {
	cb.state = CircuitBreakerClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.logger.Info("Circuit breaker closed")
}

func (cb *CircuitBreakerErrorHandler) transitionToHalfOpen() {
	cb.state = CircuitBreakerHalfOpen
	cb.successCount = 0
	cb.logger.Info("Circuit breaker half-open")
}