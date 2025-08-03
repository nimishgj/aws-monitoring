// Package main provides the entry point for the aws-monitor application.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"aws-monitoring/internal/config"
	"aws-monitoring/pkg/logger"
)

var (
	version   = "dev"
	buildTime = "unknown"
	gitCommit = "unknown"
)

func main() {
	// Parse command line flags
	var (
		configPath   = flag.String("config", "", "Path to configuration file")
		showVersion  = flag.Bool("version", false, "Show version information")
		validateOnly = flag.Bool("validate", false, "Validate configuration and exit")
	)
	flag.Parse()

	// Show version information
	if *showVersion {
		fmt.Printf("AWS Monitor %s\n", version)
		fmt.Printf("Build Time: %s\n", buildTime)
		fmt.Printf("Git Commit: %s\n", gitCommit)
		os.Exit(0)
	}

	// Load configuration first (needed for logger setup)
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger with configuration
	loggerConfig := logger.Config{
		Level:  cfg.Global.LogLevel,
		Format: cfg.Global.LogFormat,
	}

	err = logger.InitializeGlobal(loggerConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	// Ensure logs are flushed on exit
	defer func() {
		if err := logger.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to sync logger: %v\n", err)
		}
	}()

	// Create main logger
	mainLogger := logger.WithComponent("main")

	// Log application startup
	mainLogger.LogStartup(version, buildTime, gitCommit)

	// Log configuration details
	mainLogger.LogConfigLoad(*configPath, cfg.EnabledRegions)
	mainLogger.Info("OpenTelemetry configuration",
		logger.String("endpoint", cfg.OTEL.CollectorEndpoint),
		logger.String("service_name", cfg.OTEL.ServiceName),
		logger.Bool("insecure", cfg.OTEL.Insecure),
	)

	// Log collector configurations
	collectors := map[string]config.CollectorConfig{
		"ec2":    cfg.Metrics.EC2,
		"rds":    cfg.Metrics.RDS,
		"s3":     cfg.Metrics.S3,
		"lambda": cfg.Metrics.Lambda,
		"ebs":    cfg.Metrics.EBS,
		"elb":    cfg.Metrics.ELB,
		"vpc":    cfg.Metrics.VPC,
	}

	for name, collectorCfg := range collectors {
		mainLogger.LogCollectorStatus(name, collectorCfg.Enabled, time.Duration(collectorCfg.CollectionInterval))
	}

	// If validate-only mode, exit after successful validation
	if *validateOnly {
		mainLogger.Info("Configuration validation successful")
		os.Exit(0)
	}

	// Setup graceful shutdown
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	// Start application components (placeholder)
	mainLogger.Info("Starting application components",
		logger.Int("max_workers", cfg.Global.MaxConcurrentWorkers),
		logger.Int("health_port", cfg.Global.HealthCheckPort),
	)

	// TODO: Initialize and start the actual application components
	// - AWS clients
	// - Metric collectors
	// - OpenTelemetry exporter
	// - Health check server
	// - Scheduler

	mainLogger.Info("Application startup complete")

	// Wait for shutdown signal
	sig := <-shutdownChan
	shutdownStart := time.Now()

	mainLogger.Info("Received shutdown signal",
		logger.String("signal", sig.String()),
	)

	// TODO: Implement graceful shutdown
	// - Stop scheduler
	// - Drain worker pools
	// - Flush remaining metrics
	// - Close connections

	mainLogger.LogShutdown(sig.String(), time.Since(shutdownStart))
}
