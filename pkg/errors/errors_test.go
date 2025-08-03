package errors

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNewError(t *testing.T) {
	err := New(ErrorTypeAWS, "TEST_CODE", "test message")
	
	if err.Type != ErrorTypeAWS {
		t.Errorf("Expected type %s, got %s", ErrorTypeAWS, err.Type)
	}
	
	if err.Code != "TEST_CODE" {
		t.Errorf("Expected code TEST_CODE, got %s", err.Code)
	}
	
	if err.Message != "test message" {
		t.Errorf("Expected message 'test message', got %s", err.Message)
	}
	
	if err.Severity != SeverityMedium {
		t.Errorf("Expected default severity medium, got %s", err.Severity)
	}
	
	if err.Retryable {
		t.Error("Expected default retryable to be false")
	}
	
	if len(err.StackTrace) == 0 {
		t.Error("Expected stack trace to be captured")
	}
}

func TestWrapError(t *testing.T) {
	originalErr := errors.New("original error")
	wrappedErr := Wrap(originalErr, ErrorTypeNetwork, "WRAP_CODE", "wrapped message")
	
	if wrappedErr.Type != ErrorTypeNetwork {
		t.Errorf("Expected type %s, got %s", ErrorTypeNetwork, wrappedErr.Type)
	}
	
	if wrappedErr.Code != "WRAP_CODE" {
		t.Errorf("Expected code WRAP_CODE, got %s", wrappedErr.Code)
	}
	
	if wrappedErr.Message != "wrapped message" {
		t.Errorf("Expected message 'wrapped message', got %s", wrappedErr.Message)
	}
	
	if wrappedErr.Cause != originalErr {
		t.Error("Expected cause to be original error")
	}
	
	if !errors.Is(wrappedErr, originalErr) {
		t.Error("Expected wrapped error to unwrap to original error")
	}
}

func TestWrapNilError(t *testing.T) {
	wrappedErr := Wrap(nil, ErrorTypeNetwork, "TEST", "message")
	if wrappedErr != nil {
		t.Error("Expected nil when wrapping nil error")
	}
}

func TestErrorMethods(t *testing.T) {
	err := New(ErrorTypeAWS, "TEST_CODE", "test message")
	err = WithOperation(err, "test-operation")
	err = WithRegion(err, "us-east-1")
	err = WithService(err, "ec2")
	
	errorString := err.Error()
	expectedParts := []string{"operation=test-operation", "service=ec2", "region=us-east-1", "code=TEST_CODE", "test message"}
	
	for _, part := range expectedParts {
		if !strings.Contains(errorString, part) {
			t.Errorf("Expected error string to contain '%s', got: %s", part, errorString)
		}
	}
}

func TestWithMethods(t *testing.T) {
	err := New(ErrorTypeInternal, "TEST", "message")
	
	// Test WithSeverity
	err = WithSeverity(err, SeverityCritical)
	if err.Severity != SeverityCritical {
		t.Errorf("Expected severity critical, got %s", err.Severity)
	}
	
	// Test WithRetryable
	err = WithRetryable(err, true)
	if !err.Retryable {
		t.Error("Expected retryable to be true")
	}
	
	// Test WithOperation
	err = WithOperation(err, "test-op")
	if err.Operation != "test-op" {
		t.Errorf("Expected operation 'test-op', got %s", err.Operation)
	}
	
	// Test WithRegion
	err = WithRegion(err, "us-west-2")
	if err.Region != "us-west-2" {
		t.Errorf("Expected region 'us-west-2', got %s", err.Region)
	}
	
	// Test WithService
	err = WithService(err, "rds")
	if err.Service != "rds" {
		t.Errorf("Expected service 'rds', got %s", err.Service)
	}
}

func TestWithMetadata(t *testing.T) {
	err := New(ErrorTypeValidation, "TEST", "message")
	err = err.WithMetadata("key1", "value1")
	err = err.WithMetadata("key2", 42)
	
	if len(err.Metadata) != 2 {
		t.Errorf("Expected 2 metadata entries, got %d", len(err.Metadata))
	}
	
	if err.Metadata["key1"] != "value1" {
		t.Errorf("Expected metadata key1 to be 'value1', got %v", err.Metadata["key1"])
	}
	
	if err.Metadata["key2"] != 42 {
		t.Errorf("Expected metadata key2 to be 42, got %v", err.Metadata["key2"])
	}
}

func TestWithRetryAfter(t *testing.T) {
	duration := 30 * time.Second
	err := New(ErrorTypeRateLimit, "RATE_LIMIT", "rate limited")
	err = err.WithRetryAfter(duration)
	
	if err.RetryAfter == nil {
		t.Error("Expected RetryAfter to be set")
	}
	
	if *err.RetryAfter != duration {
		t.Errorf("Expected RetryAfter to be %v, got %v", duration, *err.RetryAfter)
	}
}

func TestCommonErrorConstructors(t *testing.T) {
	// Test NewAWSError
	awsErr := NewAWSError("THROTTLE", "API throttled")
	if awsErr.Type != ErrorTypeAWS {
		t.Errorf("Expected AWS error type, got %s", awsErr.Type)
	}
	if !awsErr.Retryable {
		t.Error("Expected AWS error to be retryable by default")
	}
	
	// Test NewConfigError
	configErr := NewConfigError("INVALID", "invalid config")
	if configErr.Type != ErrorTypeConfig {
		t.Errorf("Expected config error type, got %s", configErr.Type)
	}
	if configErr.Severity != SeverityCritical {
		t.Errorf("Expected critical severity, got %s", configErr.Severity)
	}
	
	// Test NewNetworkError
	netErr := NewNetworkError("CONNECTION", "connection failed")
	if netErr.Type != ErrorTypeNetwork {
		t.Errorf("Expected network error type, got %s", netErr.Type)
	}
	if !netErr.Retryable {
		t.Error("Expected network error to be retryable")
	}
	
	// Test NewTimeoutError
	timeoutErr := NewTimeoutError("test-operation", 30*time.Second)
	if timeoutErr.Type != ErrorTypeTimeout {
		t.Errorf("Expected timeout error type, got %s", timeoutErr.Type)
	}
	if timeoutErr.Operation != "test-operation" {
		t.Errorf("Expected operation 'test-operation', got %s", timeoutErr.Operation)
	}
	
	// Test NewRateLimitError
	rateLimitErr := NewRateLimitError(60 * time.Second)
	if rateLimitErr.Type != ErrorTypeRateLimit {
		t.Errorf("Expected rate limit error type, got %s", rateLimitErr.Type)
	}
	if rateLimitErr.RetryAfter == nil || *rateLimitErr.RetryAfter != 60*time.Second {
		t.Error("Expected RetryAfter to be 60 seconds")
	}
	
	// Test NewPermissionError
	permErr := NewPermissionError("read", "s3://bucket/key")
	if permErr.Type != ErrorTypePermission {
		t.Errorf("Expected permission error type, got %s", permErr.Type)
	}
	if permErr.Severity != SeverityHigh {
		t.Errorf("Expected high severity, got %s", permErr.Severity)
	}
}

func TestUtilityFunctions(t *testing.T) {
	// Test IsRetryable
	retryableErr := WithRetryable(New(ErrorTypeInternal, "TEST", "message"), true)
	nonRetryableErr := WithRetryable(New(ErrorTypeInternal, "TEST", "message"), false)
	
	if !IsRetryable(retryableErr) {
		t.Error("Expected retryable error to be retryable")
	}
	
	if IsRetryable(nonRetryableErr) {
		t.Error("Expected non-retryable error to not be retryable")
	}
	
	if IsRetryable(errors.New("standard error")) {
		t.Error("Expected standard error to not be retryable")
	}
	
	// Test IsType
	awsErr := New(ErrorTypeAWS, "TEST", "message")
	if !IsType(awsErr, ErrorTypeAWS) {
		t.Error("Expected AWS error to be AWS type")
	}
	
	if IsType(awsErr, ErrorTypeNetwork) {
		t.Error("Expected AWS error to not be network type")
	}
	
	// Test GetRetryAfter
	rateLimitErr := NewRateLimitError(45 * time.Second)
	retryAfter := GetRetryAfter(rateLimitErr)
	if retryAfter == nil || *retryAfter != 45*time.Second {
		t.Error("Expected retry after to be 45 seconds")
	}
	
	normalErr := New(ErrorTypeInternal, "TEST", "message")
	if GetRetryAfter(normalErr) != nil {
		t.Error("Expected normal error to have no retry after")
	}
}

func TestMultiError(t *testing.T) {
	multiErr := NewMultiError()
	
	if multiErr.HasErrors() {
		t.Error("Expected new MultiError to have no errors")
	}
	
	if multiErr.ErrorOrNil() != nil {
		t.Error("Expected ErrorOrNil to return nil for empty MultiError")
	}
	
	// Add some errors
	err1 := New(ErrorTypeAWS, "ERR1", "first error")
	err2 := New(ErrorTypeNetwork, "ERR2", "second error")
	standardErr := errors.New("standard error")
	
	multiErr.Add(err1)
	multiErr.Add(err2)
	multiErr.Add(standardErr)
	multiErr.Add(nil) // Should be ignored
	
	if !multiErr.HasErrors() {
		t.Error("Expected MultiError to have errors after adding")
	}
	
	if len(multiErr.Errors) != 3 {
		t.Errorf("Expected 3 errors, got %d", len(multiErr.Errors))
	}
	
	errorString := multiErr.Error()
	if !strings.Contains(errorString, "multiple errors occurred") {
		t.Error("Expected error string to mention multiple errors")
	}
	
	if multiErr.ErrorOrNil() == nil {
		t.Error("Expected ErrorOrNil to return the MultiError when it has errors")
	}
}

func TestMultiErrorSingleError(t *testing.T) {
	multiErr := NewMultiError()
	singleErr := New(ErrorTypeValidation, "SINGLE", "single error")
	multiErr.Add(singleErr)
	
	errorString := multiErr.Error()
	if errorString != singleErr.Error() {
		t.Errorf("Expected single error string, got: %s", errorString)
	}
}

func TestErrorIs(t *testing.T) {
	originalErr := errors.New("original")
	wrappedErr := Wrap(originalErr, ErrorTypeNetwork, "WRAPPED", "wrapped")
	
	if !errors.Is(wrappedErr, originalErr) {
		t.Error("Expected wrapped error to match original via errors.Is")
	}
	
	err1 := New(ErrorTypeAWS, "TEST_CODE", "message")
	err2 := New(ErrorTypeAWS, "TEST_CODE", "different message")
	err3 := New(ErrorTypeAWS, "DIFFERENT_CODE", "message")
	
	if !err1.Is(err2) {
		t.Error("Expected errors with same type and code to match")
	}
	
	if err1.Is(err3) {
		t.Error("Expected errors with different codes to not match")
	}
}