// Package aws provides AWS service client management and configuration.
package aws

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ec2"

	appConfig "aws-monitoring/internal/config"
	"aws-monitoring/pkg/logger"
)

// EC2Client interface defines EC2 operations needed for metrics collection
type EC2Client interface {
	DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error)
	DescribeInstanceStatus(ctx context.Context, params *ec2.DescribeInstanceStatusInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstanceStatusOutput, error)
}

// ClientProvider interface for creating AWS service clients
type ClientProvider interface {
	GetEC2Client(region string) (EC2Client, error)
	Close() error
}

// clientProvider implements ClientProvider
type clientProvider struct {
	config     *appConfig.Config
	logger     *logger.Logger
	awsConfigs map[string]aws.Config
}

// NewClientProvider creates a new AWS client provider
func NewClientProvider(cfg *appConfig.Config, log *logger.Logger) ClientProvider {
	return &clientProvider{
		config:     cfg,
		logger:     log.WithComponent("aws-client"),
		awsConfigs: make(map[string]aws.Config),
	}
}

// GetEC2Client returns an EC2 client for the specified region
func (cp *clientProvider) GetEC2Client(region string) (EC2Client, error) {
	awsCfg, err := cp.getAWSConfig(region)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS config for region %s: %w", region, err)
	}

	client := ec2.NewFromConfig(awsCfg)
	cp.logger.Debug("Created EC2 client", logger.String("region", region))

	return client, nil
}

// getAWSConfig returns AWS config for the specified region, creating it if needed
func (cp *clientProvider) getAWSConfig(region string) (aws.Config, error) {
	// Check if we already have a config for this region
	if cfg, exists := cp.awsConfigs[region]; exists {
		return cfg, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var awsCfg aws.Config
	var err error

	// Load config based on whether we have explicit credentials
	if cp.config.AWS.AccessKeyID != "" && cp.config.AWS.SecretAccessKey != "" {
		// Use explicit credentials from config
		awsCfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				cp.config.AWS.AccessKeyID,
				cp.config.AWS.SecretAccessKey,
				"", // session token
			)),
			config.WithRetryMaxAttempts(cp.config.AWS.MaxRetries),
		)
	} else {
		// Use default credential chain (IAM roles, environment, etc.)
		awsCfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
			config.WithRetryMaxAttempts(cp.config.AWS.MaxRetries),
		)
	}

	if err != nil {
		return aws.Config{}, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Apply timeout configuration
	awsCfg.HTTPClient = &http.Client{
		Timeout: time.Duration(cp.config.AWS.Timeout),
	}

	// Store the config for reuse
	cp.awsConfigs[region] = awsCfg

	cp.logger.Info("AWS config loaded",
		logger.String("region", region),
		logger.Int("max_retries", cp.config.AWS.MaxRetries),
		logger.Duration("timeout", time.Duration(cp.config.AWS.Timeout)),
	)

	return awsCfg, nil
}

// Close cleans up any resources used by the client provider
func (cp *clientProvider) Close() error {
	cp.logger.Debug("Closing AWS client provider")
	// Clear cached configs
	cp.awsConfigs = make(map[string]aws.Config)
	return nil
}

// ValidateCredentials checks if AWS credentials are valid by making a simple API call
func (cp *clientProvider) ValidateCredentials(region string) error {
	client, err := cp.GetEC2Client(region)
	if err != nil {
		return fmt.Errorf("failed to create EC2 client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Make a simple API call to validate credentials
	_, err = client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		MaxResults: aws.Int32(1), // Limit to 1 result to minimize API cost
	})

	if err != nil {
		return fmt.Errorf("credential validation failed: %w", err)
	}

	cp.logger.Info("AWS credentials validated successfully", logger.String("region", region))
	return nil
}