package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestMain tests the main function behavior
func TestMain(t *testing.T) {
	// Skip if we're in a short test run
	if testing.Short() {
		t.Skip("Skipping main function test in short mode")
	}

	tests := []struct {
		name     string
		args     []string
		wantExit int
		wantOut  string
	}{
		{
			name:     "version flag",
			args:     []string{"--version"},
			wantExit: 0,
			wantOut:  "AWS Monitor",
		},
		{
			name:     "help flag",
			args:     []string{"--help"},
			wantExit: 0,
			wantOut:  "Usage of",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build the binary for testing
			tmpDir := t.TempDir()
			binaryPath := filepath.Join(tmpDir, "aws-monitor-test")
			
			cmd := exec.Command("go", "build", "-o", binaryPath, ".")
			if err := cmd.Run(); err != nil {
				t.Fatalf("Failed to build binary: %v", err)
			}

			// Run the binary with test arguments
			cmd = exec.Command(binaryPath, tt.args...)
			output, err := cmd.CombinedOutput()

			// Check exit code
			exitCode := 0
			if err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					exitCode = exitError.ExitCode()
				} else {
					t.Fatalf("Failed to run binary: %v", err)
				}
			}

			if exitCode != tt.wantExit {
				t.Errorf("Expected exit code %d, got %d", tt.wantExit, exitCode)
			}

			// Check output content
			outputStr := string(output)
			if !strings.Contains(outputStr, tt.wantOut) {
				t.Errorf("Expected output to contain %q, got: %s", tt.wantOut, outputStr)
			}
		})
	}
}

func TestMainWithValidateFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping main function test in short mode")
	}

	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")
	
	configContent := `
enabled_regions:
  - us-east-1
aws:
  access_key_id: "test-key"
  secret_access_key: "test-secret"
  default_region: us-east-1
otel:
  collector_endpoint: "http://localhost:4317"
  service_name: "aws-monitor-test"
metrics:
  ec2:
    enabled: true
`

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Build the binary
	binaryPath := filepath.Join(tmpDir, "aws-monitor-test")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// Test validate flag
	cmd = exec.Command(binaryPath, "--validate", "--config", configPath)
	output, err := cmd.CombinedOutput()

	// Should exit with code 0 for successful validation
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
	}

	if exitCode != 0 {
		t.Errorf("Expected exit code 0 for validation, got %d. Output: %s", exitCode, string(output))
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Configuration validation successful") {
		t.Errorf("Expected validation success message, got: %s", outputStr)
	}
}

func TestMainWithInvalidConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping main function test in short mode")
	}

	// Create a temporary invalid config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid-config.yaml")
	
	invalidConfigContent := `
enabled_regions: []
aws:
  access_key_id: ""
  secret_access_key: ""
otel:
  collector_endpoint: "invalid-url"
  service_name: ""
`

	if err := os.WriteFile(configPath, []byte(invalidConfigContent), 0600); err != nil {
		t.Fatalf("Failed to write invalid config: %v", err)
	}

	// Build the binary
	binaryPath := filepath.Join(tmpDir, "aws-monitor-test")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// Test with invalid config
	cmd = exec.Command(binaryPath, "--config", configPath)
	output, err := cmd.CombinedOutput()

	// Should exit with non-zero code for invalid config
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
	}

	if exitCode == 0 {
		t.Errorf("Expected non-zero exit code for invalid config, got 0. Output: %s", string(output))
	}
}

func TestMainWithNonExistentConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping main function test in short mode")
	}

	// Build the binary
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "aws-monitor-test")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// Test with non-existent config file
	nonExistentPath := filepath.Join(tmpDir, "non-existent.yaml")
	cmd = exec.Command(binaryPath, "--config", nonExistentPath)
	output, err := cmd.CombinedOutput()

	// Should exit with non-zero code for missing config
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
	}

	if exitCode == 0 {
		t.Errorf("Expected non-zero exit code for missing config, got 0. Output: %s", string(output))
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Failed to load configuration") {
		t.Errorf("Expected configuration load error, got: %s", outputStr)
	}
}

// TestVersionVariable tests that version variables are properly set
func TestVersionVariables(t *testing.T) {
	// Test that version variables exist and have expected default values
	if version == "" {
		t.Error("Version variable should not be empty")
	}

	if buildTime == "" {
		t.Error("BuildTime variable should not be empty")
	}

	if gitCommit == "" {
		t.Error("GitCommit variable should not be empty")
	}

	// Test default values
	if version != "dev" {
		t.Errorf("Expected default version 'dev', got '%s'", version)
	}

	if buildTime != "unknown" {
		t.Errorf("Expected default buildTime 'unknown', got '%s'", buildTime)
	}

	if gitCommit != "unknown" {
		t.Errorf("Expected default gitCommit 'unknown', got '%s'", gitCommit)
	}
}

// TestMainTimeout tests that main doesn't hang indefinitely
func TestMainTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timeout test in short mode")
	}

	// Create a minimal valid config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "timeout-test-config.yaml")
	
	configContent := `
enabled_regions:
  - us-east-1
aws:
  access_key_id: "test-key"
  secret_access_key: "test-secret"
  default_region: us-east-1
otel:
  collector_endpoint: "http://localhost:4317"
  service_name: "aws-monitor-test"
metrics:
  ec2:
    enabled: true
`

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Build the binary
	binaryPath := filepath.Join(tmpDir, "aws-monitor-test")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v", err)
	}

	// Test that validate mode exits quickly
	cmd = exec.Command(binaryPath, "--validate", "--config", configPath)
	
	done := make(chan error, 1)
	go func() {
		_, err := cmd.CombinedOutput()
		done <- err
	}()

	select {
	case err := <-done:
		// Command completed - check it was successful validation
		exitCode := 0
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				exitCode = exitError.ExitCode()
			}
		}
		if exitCode != 0 {
			t.Errorf("Expected successful validation, got exit code %d", exitCode)
		}
	case <-time.After(10 * time.Second):
		// Kill the process if it's still running
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		t.Fatal("Main function with --validate flag took too long (>10s)")
	}
}