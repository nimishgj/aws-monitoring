// Package errors provides enhanced error handling and reporting for the AWS monitoring application.
package errors

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"
)

// ErrorType represents different categories of errors
type ErrorType string

const (
	// ErrorTypeAWS represents AWS API-related errors
	ErrorTypeAWS ErrorType = "aws"
	// ErrorTypeConfig represents configuration-related errors
	ErrorTypeConfig ErrorType = "config"
	// ErrorTypeNetwork represents network-related errors
	ErrorTypeNetwork ErrorType = "network"
	// ErrorTypeValidation represents validation errors
	ErrorTypeValidation ErrorType = "validation"
	// ErrorTypeInternal represents internal application errors
	ErrorTypeInternal ErrorType = "internal"
	// ErrorTypeTimeout represents timeout errors
	ErrorTypeTimeout ErrorType = "timeout"
	// ErrorTypePermission represents permission/authorization errors
	ErrorTypePermission ErrorType = "permission"
	// ErrorTypeRateLimit represents rate limiting errors
	ErrorTypeRateLimit ErrorType = "rate_limit"
)

// Severity represents the severity level of an error
type Severity string

const (
	// SeverityLow represents low severity errors that can be ignored
	SeverityLow Severity = "low"
	// SeverityMedium represents medium severity errors that should be logged
	SeverityMedium Severity = "medium"
	// SeverityHigh represents high severity errors that require attention
	SeverityHigh Severity = "high"
	// SeverityCritical represents critical errors that may cause system failure
	SeverityCritical Severity = "critical"
)

// Error represents an enhanced error with additional context
type Error struct {
	// Type categorizes the error
	Type ErrorType `json:"type"`
	// Code is a specific error code for programmatic handling
	Code string `json:"code"`
	// Message is the human-readable error message
	Message string `json:"message"`
	// Severity indicates how critical this error is
	Severity Severity `json:"severity"`
	// Timestamp when the error occurred
	Timestamp time.Time `json:"timestamp"`
	// Operation that was being performed when the error occurred
	Operation string `json:"operation,omitempty"`
	// Region where the error occurred (for AWS errors)
	Region string `json:"region,omitempty"`
	// Service that caused the error
	Service string `json:"service,omitempty"`
	// Cause is the underlying error that caused this error
	Cause error `json:"cause,omitempty"`
	// Retryable indicates if this error can be retried
	Retryable bool `json:"retryable"`
	// RetryAfter suggests when to retry (for rate limit errors)
	RetryAfter *time.Duration `json:"retry_after,omitempty"`
	// StackTrace provides debugging information
	StackTrace []string `json:"stack_trace,omitempty"`
	// Metadata contains additional context-specific information
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Error implements the error interface
func (e *Error) Error() string {
	var parts []string
	
	if e.Operation != "" {
		parts = append(parts, fmt.Sprintf("operation=%s", e.Operation))
	}
	
	if e.Service != "" {
		parts = append(parts, fmt.Sprintf("service=%s", e.Service))
	}
	
	if e.Region != "" {
		parts = append(parts, fmt.Sprintf("region=%s", e.Region))
	}
	
	if e.Code != "" {
		parts = append(parts, fmt.Sprintf("code=%s", e.Code))
	}
	
	prefix := ""
	if len(parts) > 0 {
		prefix = fmt.Sprintf("[%s] ", strings.Join(parts, ", "))
	}
	
	return prefix + e.Message
}

// Unwrap returns the underlying cause error
func (e *Error) Unwrap() error {
	return e.Cause
}

// Is checks if this error matches the target error
func (e *Error) Is(target error) bool {
	if t, ok := target.(*Error); ok {
		return e.Type == t.Type && e.Code == t.Code
	}
	return errors.Is(e.Cause, target)
}

// WithMetadata adds metadata to the error
func (e *Error) WithMetadata(key string, value interface{}) *Error {
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	e.Metadata[key] = value
	return e
}

// WithRetryAfter sets the retry delay for rate limit errors
func (e *Error) WithRetryAfter(duration time.Duration) *Error {
	e.RetryAfter = &duration
	return e
}

// New creates a new Error with the given parameters
func New(errorType ErrorType, code, message string) *Error {
	return &Error{
		Type:       errorType,
		Code:       code,
		Message:    message,
		Severity:   SeverityMedium,
		Timestamp:  time.Now(),
		Retryable:  false,
		StackTrace: captureStackTrace(),
	}
}

// Wrap wraps an existing error with additional context
func Wrap(err error, errorType ErrorType, code, message string) *Error {
	if err == nil {
		return nil
	}
	
	// If it's already our Error type, enhance it
	if e, ok := err.(*Error); ok {
		enhanced := *e // Copy the error
		if enhanced.Type == "" {
			enhanced.Type = errorType
		}
		if enhanced.Code == "" {
			enhanced.Code = code
		}
		if message != "" {
			enhanced.Message = message + ": " + enhanced.Message
		}
		return &enhanced
	}
	
	return &Error{
		Type:       errorType,
		Code:       code,
		Message:    message,
		Severity:   SeverityMedium,
		Timestamp:  time.Now(),
		Cause:      err,
		Retryable:  false,
		StackTrace: captureStackTrace(),
	}
}

// WithSeverity sets the severity level of the error
func WithSeverity(err *Error, severity Severity) *Error {
	if err != nil {
		err.Severity = severity
	}
	return err
}

// WithRetryable marks the error as retryable or not
func WithRetryable(err *Error, retryable bool) *Error {
	if err != nil {
		err.Retryable = retryable
	}
	return err
}

// WithOperation adds operation context to the error
func WithOperation(err *Error, operation string) *Error {
	if err != nil {
		err.Operation = operation
	}
	return err
}

// WithRegion adds region context to the error
func WithRegion(err *Error, region string) *Error {
	if err != nil {
		err.Region = region
	}
	return err
}

// WithService adds service context to the error
func WithService(err *Error, service string) *Error {
	if err != nil {
		err.Service = service
	}
	return err
}

// Common error constructors

// NewAWSError creates a new AWS-related error
func NewAWSError(code, message string) *Error {
	return WithRetryable(New(ErrorTypeAWS, code, message), true)
}

// NewConfigError creates a new configuration error
func NewConfigError(code, message string) *Error {
	return WithSeverity(New(ErrorTypeConfig, code, message), SeverityCritical)
}

// NewNetworkError creates a new network error
func NewNetworkError(code, message string) *Error {
	return WithRetryable(New(ErrorTypeNetwork, code, message), true)
}

// NewValidationError creates a new validation error
func NewValidationError(code, message string) *Error {
	return New(ErrorTypeValidation, code, message)
}

// NewTimeoutError creates a new timeout error
func NewTimeoutError(operation string, timeout time.Duration) *Error {
	return WithRetryable(
		WithOperation(
			New(ErrorTypeTimeout, "TIMEOUT", 
				fmt.Sprintf("operation timed out after %v", timeout)),
			operation),
		true)
}

// NewRateLimitError creates a new rate limit error
func NewRateLimitError(retryAfter time.Duration) *Error {
	return WithRetryable(
		New(ErrorTypeRateLimit, "RATE_LIMIT", "rate limit exceeded").
			WithRetryAfter(retryAfter),
		true)
}

// NewPermissionError creates a new permission error
func NewPermissionError(operation, resource string) *Error {
	return WithSeverity(
		WithOperation(
			New(ErrorTypePermission, "ACCESS_DENIED", 
				fmt.Sprintf("insufficient permissions for resource: %s", resource)),
			operation),
		SeverityHigh)
}

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	if e, ok := err.(*Error); ok {
		return e.Retryable
	}
	return false
}

// IsType checks if an error is of a specific type
func IsType(err error, errorType ErrorType) bool {
	if e, ok := err.(*Error); ok {
		return e.Type == errorType
	}
	return false
}

// GetRetryAfter returns the retry delay if the error is a rate limit error
func GetRetryAfter(err error) *time.Duration {
	if e, ok := err.(*Error); ok && e.RetryAfter != nil {
		return e.RetryAfter
	}
	return nil
}

// captureStackTrace captures the current stack trace
func captureStackTrace() []string {
	const maxDepth = 10
	var stackTrace []string
	
	for i := 2; i < maxDepth; i++ { // Skip this function and the caller
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		
		// Simplify file paths to just the package/file
		if idx := strings.LastIndex(file, "/"); idx >= 0 {
			file = file[idx+1:]
		}
		
		stackTrace = append(stackTrace, fmt.Sprintf("%s:%d", file, line))
	}
	
	return stackTrace
}

// Multi-error handling

// MultiError represents multiple errors that occurred together
type MultiError struct {
	Errors []*Error `json:"errors"`
}

// Error implements the error interface for MultiError
func (m *MultiError) Error() string {
	if len(m.Errors) == 0 {
		return "no errors"
	}
	
	if len(m.Errors) == 1 {
		return m.Errors[0].Error()
	}
	
	var messages []string
	for _, err := range m.Errors {
		messages = append(messages, err.Error())
	}
	
	return fmt.Sprintf("multiple errors occurred: %s", strings.Join(messages, "; "))
}

// Add adds an error to the MultiError
func (m *MultiError) Add(err error) {
	if err == nil {
		return
	}
	
	if e, ok := err.(*Error); ok {
		m.Errors = append(m.Errors, e)
	} else {
		m.Errors = append(m.Errors, Wrap(err, ErrorTypeInternal, "WRAPPED", "wrapped error"))
	}
}

// HasErrors returns true if there are any errors
func (m *MultiError) HasErrors() bool {
	return len(m.Errors) > 0
}

// ErrorOrNil returns the MultiError if it has errors, otherwise nil
func (m *MultiError) ErrorOrNil() error {
	if m.HasErrors() {
		return m
	}
	return nil
}

// NewMultiError creates a new MultiError
func NewMultiError() *MultiError {
	return &MultiError{
		Errors: make([]*Error, 0),
	}
}