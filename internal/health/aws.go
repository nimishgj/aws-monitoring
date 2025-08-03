package health

import (
	"context"
	"fmt"
	"time"

	"aws-monitoring/internal/aws"
	"aws-monitoring/internal/config"
	"aws-monitoring/pkg/logger"
)

// AWSChecker implements health checks for AWS connectivity
type AWSChecker struct {
	clientProvider aws.ClientProvider
	config         *config.Config
	logger         *logger.Logger
	name           string
}

// NewAWSChecker creates a new AWS connectivity health checker
func NewAWSChecker(clientProvider aws.ClientProvider, cfg *config.Config, log *logger.Logger) *AWSChecker {
	return &AWSChecker{
		clientProvider: clientProvider,
		config:         cfg,
		logger:         log.WithComponent("aws-health-checker"),
		name:           "aws_connectivity",
	}
}

// Name returns the unique identifier for this checker
func (c *AWSChecker) Name() string {
	return c.name
}

// Check performs AWS connectivity health checks
func (c *AWSChecker) Check(ctx context.Context) CheckResult {
	start := time.Now()
	result := CheckResult{
		Name:        c.name,
		LastChecked: start,
		Metadata:    make(map[string]interface{}),
	}

	// If no regions are enabled, mark as degraded
	if len(c.config.EnabledRegions) == 0 {
		result.Status = StatusDegraded
		result.Message = "No AWS regions enabled"
		result.Duration = time.Since(start)
		return result
	}

	// Check connectivity to all enabled regions
	regionResults := make(map[string]string)
	healthyRegions := 0
	totalRegions := len(c.config.EnabledRegions)

	for _, region := range c.config.EnabledRegions {
		regionStatus := c.checkRegion(ctx, region)
		regionResults[region] = regionStatus
		
		if regionStatus == "healthy" {
			healthyRegions++
		}
	}

	result.Metadata["regions"] = regionResults
	result.Metadata["healthy_regions"] = healthyRegions
	result.Metadata["total_regions"] = totalRegions
	result.Duration = time.Since(start)

	// Determine overall AWS connectivity status
	switch healthyRegions {
	case 0:
		result.Status = StatusUnhealthy
		result.Message = "No AWS regions accessible"
		result.Error = "All configured AWS regions are unreachable"
	case totalRegions:
		result.Status = StatusHealthy
		result.Message = fmt.Sprintf("All %d AWS regions accessible", totalRegions)
	default:
		result.Status = StatusDegraded
		result.Message = fmt.Sprintf("%d of %d AWS regions accessible", healthyRegions, totalRegions)
	}

	return result
}

// checkRegion checks connectivity to a specific AWS region
func (c *AWSChecker) checkRegion(ctx context.Context, region string) string {
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Get EC2 client for the region
	client, err := c.clientProvider.GetEC2Client(region)
	if err != nil {
		c.logger.Debug("Failed to create EC2 client",
			logger.String("region", region),
			logger.String("error", err.Error()))
		return "unhealthy"
	}

	// Perform a simple API call to test connectivity
	_, err = client.DescribeInstances(checkCtx, nil)
	if err != nil {
		c.logger.Debug("AWS connectivity check failed",
			logger.String("region", region),
			logger.String("error", err.Error()))
		return "unhealthy"
	}

	c.logger.Debug("AWS connectivity check successful", logger.String("region", region))
	return "healthy"
}

// BasicChecker implements a simple health check for basic application status
type BasicChecker struct {
	name    string
	service string
	version string
}

// NewBasicChecker creates a new basic health checker
func NewBasicChecker(service, version string) *BasicChecker {
	return &BasicChecker{
		name:    "basic",
		service: service,
		version: version,
	}
}

// Name returns the unique identifier for this checker
func (c *BasicChecker) Name() string {
	return c.name
}

// Check performs a basic health check
func (c *BasicChecker) Check(_ context.Context) CheckResult {
	start := time.Now()
	return CheckResult{
		Name:        c.name,
		Status:      StatusHealthy,
		Message:     fmt.Sprintf("%s is running", c.service),
		LastChecked: start,
		Duration:    time.Since(start),
		Metadata: map[string]interface{}{
			"service": c.service,
			"version": c.version,
		},
	}
}

// ConfigChecker implements health checks for configuration validity
type ConfigChecker struct {
	config *config.Config
	logger *logger.Logger
	name   string
}

// NewConfigChecker creates a new configuration health checker
func NewConfigChecker(cfg *config.Config, log *logger.Logger) *ConfigChecker {
	return &ConfigChecker{
		config: cfg,
		logger: log.WithComponent("config-health-checker"),
		name:   "configuration",
	}
}

// Name returns the unique identifier for this checker
func (c *ConfigChecker) Name() string {
	return c.name
}

// Check performs configuration validation health checks
func (c *ConfigChecker) Check(_ context.Context) CheckResult {
	start := time.Now()
	result := CheckResult{
		Name:        c.name,
		LastChecked: start,
		Metadata:    make(map[string]interface{}),
	}

	issues := []string{}

	// Check if regions are configured
	if len(c.config.EnabledRegions) == 0 {
		issues = append(issues, "no regions enabled")
	}

	// Check OTEL configuration
	if c.config.OTEL.CollectorEndpoint == "" {
		issues = append(issues, "OTEL collector endpoint not configured")
	}

	if c.config.OTEL.ServiceName == "" {
		issues = append(issues, "OTEL service name not configured")
	}

	// Check if any metrics collectors are enabled
	anyEnabled := c.config.Metrics.EC2.Enabled ||
		c.config.Metrics.RDS.Enabled ||
		c.config.Metrics.S3.Enabled ||
		c.config.Metrics.Lambda.Enabled ||
		c.config.Metrics.EBS.Enabled ||
		c.config.Metrics.ELB.Enabled ||
		c.config.Metrics.VPC.Enabled

	if !anyEnabled {
		issues = append(issues, "no metrics collectors enabled")
	}

	result.Metadata["enabled_regions"] = c.config.EnabledRegions
	result.Metadata["otel_endpoint"] = c.config.OTEL.CollectorEndpoint
	result.Metadata["enabled_collectors"] = c.getEnabledCollectors()
	result.Duration = time.Since(start)

	if len(issues) == 0 {
		result.Status = StatusHealthy
		result.Message = "Configuration is valid"
	} else if len(issues) <= 2 {
		result.Status = StatusDegraded
		result.Message = fmt.Sprintf("Configuration has minor issues: %v", issues)
	} else {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("Configuration has significant issues: %v", issues)
		result.Error = fmt.Sprintf("Configuration validation failed: %v", issues)
	}

	return result
}

// getEnabledCollectors returns a list of enabled metric collectors
func (c *ConfigChecker) getEnabledCollectors() []string {
	var enabled []string
	
	if c.config.Metrics.EC2.Enabled {
		enabled = append(enabled, "ec2")
	}
	if c.config.Metrics.RDS.Enabled {
		enabled = append(enabled, "rds")
	}
	if c.config.Metrics.S3.Enabled {
		enabled = append(enabled, "s3")
	}
	if c.config.Metrics.Lambda.Enabled {
		enabled = append(enabled, "lambda")
	}
	if c.config.Metrics.EBS.Enabled {
		enabled = append(enabled, "ebs")
	}
	if c.config.Metrics.ELB.Enabled {
		enabled = append(enabled, "elb")
	}
	if c.config.Metrics.VPC.Enabled {
		enabled = append(enabled, "vpc")
	}
	
	return enabled
}