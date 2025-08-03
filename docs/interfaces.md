# Interface and Class Design

This document defines the key interfaces, structures, and design patterns for the AWS monitoring application.

## Core Interfaces

### MetricCollector Interface

```go
type MetricCollector interface {
    // GetName returns the collector's identifier
    GetName() string
    
    // Collect gathers metrics for the specific AWS resource
    Collect(ctx context.Context, region string) ([]MetricData, error)
    
    // IsEnabled checks if collection is enabled for this collector
    IsEnabled() bool
    
    // GetCollectionInterval returns how often to collect metrics
    GetCollectionInterval() time.Duration
    
    // Validate ensures the collector is properly configured
    Validate() error
}
```

### AWSClientProvider Interface

```go
type AWSClientProvider interface {
    // GetEC2Client returns an EC2 client for the specified region
    GetEC2Client(region string) (ec2iface.EC2API, error)
    
    // GetRDSClient returns an RDS client for the specified region
    GetRDSClient(region string) (rdsiface.RDSAPI, error)
    
    // GetS3Client returns an S3 client
    GetS3Client() (s3iface.S3API, error)
    
    // GetLambdaClient returns a Lambda client for the specified region
    GetLambdaClient(region string) (lambdaiface.LambdaAPI, error)
    
    // GetELBClient returns an ELB client for the specified region
    GetELBClient(region string) (elbiface.ELBAPI, error)
    
    // GetEC2VPCClient returns a VPC client for the specified region
    GetEC2VPCClient(region string) (ec2iface.EC2API, error)
}
```

### MetricExporter Interface

```go
type MetricExporter interface {
    // Export sends metrics to the OpenTelemetry collector
    Export(ctx context.Context, metrics []MetricData) error
    
    // Shutdown gracefully shuts down the exporter
    Shutdown(ctx context.Context) error
    
    // IsHealthy checks if the exporter is working properly
    IsHealthy() bool
}
```

### Scheduler Interface

```go
type Scheduler interface {
    // Start begins the metric collection scheduling
    Start(ctx context.Context) error
    
    // Stop gracefully stops all scheduled collections
    Stop(ctx context.Context) error
    
    // AddCollector registers a collector for scheduling
    AddCollector(collector MetricCollector) error
    
    // RemoveCollector unregisters a collector
    RemoveCollector(collectorName string) error
    
    // GetStatus returns the current scheduler status
    GetStatus() SchedulerStatus
}
```

## Core Data Structures

### Configuration Structures

```go
type Config struct {
    EnabledRegions []string      `yaml:"enabled_regions"`
    AWS            AWSConfig     `yaml:"aws"`
    OTEL           OTELConfig    `yaml:"otel"`
    Metrics        MetricsConfig `yaml:"metrics"`
    Global         GlobalConfig  `yaml:"global"`
}

type AWSConfig struct {
    AccessKeyID   string   `yaml:"access_key_id"`
    SecretKey     string   `yaml:"secret_access_key"`
    Region        string   `yaml:"default_region"`
    MaxRetries    int      `yaml:"max_retries"`
    Timeout       Duration `yaml:"timeout"`
}

type OTELConfig struct {
    CollectorEndpoint string            `yaml:"collector_endpoint"`
    ServiceName       string            `yaml:"service_name"`
    Headers           map[string]string `yaml:"headers"`
    Insecure          bool              `yaml:"insecure"`
}

type MetricsConfig struct {
    EC2    CollectorConfig `yaml:"ec2"`
    RDS    CollectorConfig `yaml:"rds"`
    S3     CollectorConfig `yaml:"s3"`
    Lambda CollectorConfig `yaml:"lambda"`
    EBS    CollectorConfig `yaml:"ebs"`
    ELB    CollectorConfig `yaml:"elb"`
    VPC    CollectorConfig `yaml:"vpc"`
}

type CollectorConfig struct {
    Enabled            bool     `yaml:"enabled"`
    CollectionInterval Duration `yaml:"collection_interval"`
}

type GlobalConfig struct {
    LogLevel             string   `yaml:"log_level"`
    HealthCheckPort      int      `yaml:"health_check_port"`
    DefaultInterval      Duration `yaml:"default_collection_interval"`
    MaxConcurrentWorkers int      `yaml:"max_concurrent_workers"`
}
```

### Metric Data Structures

```go
type MetricData struct {
    Name        string
    Description string
    Unit        string
    Value       float64
    Labels      map[string]string
    Timestamp   time.Time
    Type        MetricType
}

type MetricType int

const (
    MetricTypeGauge MetricType = iota
    MetricTypeCounter
    MetricTypeHistogram
)
```

### Health Check Structures

```go
type HealthStatus struct {
    Status      string                    `json:"status"`
    Timestamp   time.Time                 `json:"timestamp"`
    Collectors  map[string]CollectorHealth `json:"collectors"`
    Exporter    ExporterHealth            `json:"exporter"`
    Scheduler   SchedulerHealth           `json:"scheduler"`
}

type CollectorHealth struct {
    Enabled       bool      `json:"enabled"`
    LastSuccess   time.Time `json:"last_success"`
    LastError     string    `json:"last_error,omitempty"`
    ErrorCount    int       `json:"error_count"`
    SuccessCount  int       `json:"success_count"`
}

type ExporterHealth struct {
    Connected     bool      `json:"connected"`
    LastExport    time.Time `json:"last_export"`
    ExportCount   int       `json:"export_count"`
    ErrorCount    int       `json:"error_count"`
}

type SchedulerHealth struct {
    Running       bool      `json:"running"`
    ActiveWorkers int       `json:"active_workers"`
    QueueSize     int       `json:"queue_size"`
}
```

## Class Implementations

### BaseCollector (Abstract Base)

```go
type BaseCollector struct {
    name               string
    config             CollectorConfig
    awsClientProvider  AWSClientProvider
    logger             *zap.Logger
    metrics            *CollectorMetrics
}

func (bc *BaseCollector) GetName() string {
    return bc.name
}

func (bc *BaseCollector) IsEnabled() bool {
    return bc.config.Enabled
}

func (bc *BaseCollector) GetCollectionInterval() time.Duration {
    return time.Duration(bc.config.CollectionInterval)
}

func (bc *BaseCollector) Validate() error {
    if bc.awsClientProvider == nil {
        return errors.New("AWS client provider is required")
    }
    return nil
}

// Abstract method - must be implemented by concrete collectors
func (bc *BaseCollector) Collect(ctx context.Context, region string) ([]MetricData, error) {
    panic("Collect method must be implemented by concrete collectors")
}
```

### EC2Collector (Concrete Implementation)

```go
type EC2Collector struct {
    BaseCollector
}

func NewEC2Collector(config CollectorConfig, provider AWSClientProvider, logger *zap.Logger) *EC2Collector {
    return &EC2Collector{
        BaseCollector: BaseCollector{
            name:              "ec2",
            config:            config,
            awsClientProvider: provider,
            logger:            logger,
        },
    }
}

func (ec *EC2Collector) Collect(ctx context.Context, region string) ([]MetricData, error) {
    client, err := ec.awsClientProvider.GetEC2Client(region)
    if err != nil {
        return nil, fmt.Errorf("failed to get EC2 client: %w", err)
    }
    
    // Implementation for collecting EC2 metrics
    instances, err := ec.describeInstances(ctx, client)
    if err != nil {
        return nil, err
    }
    
    return ec.processInstances(instances, region), nil
}
```

### ClientFactory

```go
type ClientFactory struct {
    session    *session.Session
    clientLock sync.RWMutex
    clients    map[string]interface{}
}

func NewClientFactory(cfg AWSConfig) (*ClientFactory, error) {
    sess, err := session.NewSession(&aws.Config{
        Region: aws.String(cfg.Region),
        // Additional configuration
    })
    if err != nil {
        return nil, err
    }
    
    return &ClientFactory{
        session: sess,
        clients: make(map[string]interface{}),
    }, nil
}

func (cf *ClientFactory) GetEC2Client(region string) (ec2iface.EC2API, error) {
    cf.clientLock.RLock()
    key := fmt.Sprintf("ec2-%s", region)
    if client, exists := cf.clients[key]; exists {
        cf.clientLock.RUnlock()
        return client.(ec2iface.EC2API), nil
    }
    cf.clientLock.RUnlock()
    
    // Create new client
    client := ec2.New(cf.session, &aws.Config{Region: aws.String(region)})
    
    cf.clientLock.Lock()
    cf.clients[key] = client
    cf.clientLock.Unlock()
    
    return client, nil
}
```

### WorkerScheduler

```go
type WorkerScheduler struct {
    collectors   map[string]MetricCollector
    exporter     MetricExporter
    workerPool   chan struct{}
    ticker       map[string]*time.Ticker
    stopChannels map[string]chan struct{}
    wg           sync.WaitGroup
    mu           sync.RWMutex
    logger       *zap.Logger
}

func NewWorkerScheduler(maxWorkers int, exporter MetricExporter, logger *zap.Logger) *WorkerScheduler {
    return &WorkerScheduler{
        collectors:   make(map[string]MetricCollector),
        exporter:     exporter,
        workerPool:   make(chan struct{}, maxWorkers),
        ticker:       make(map[string]*time.Ticker),
        stopChannels: make(map[string]chan struct{}),
        logger:       logger,
    }
}

func (ws *WorkerScheduler) Start(ctx context.Context) error {
    ws.mu.Lock()
    defer ws.mu.Unlock()
    
    for name, collector := range ws.collectors {
        if !collector.IsEnabled() {
            continue
        }
        
        ws.startCollectorSchedule(ctx, name, collector)
    }
    
    return nil
}
```

## Design Patterns Used

### Factory Pattern
- **AWSClientFactory**: Creates and manages AWS service clients
- **CollectorFactory**: Creates appropriate collectors based on configuration

### Strategy Pattern
- **MetricCollector**: Different collection strategies for each AWS service
- **MetricExporter**: Different export strategies (OTEL, logging, etc.)

### Observer Pattern
- **HealthChecker**: Observes collector and exporter status
- **MetricAggregator**: Observes metric collection events

### Singleton Pattern
- **ConfigManager**: Single source of configuration
- **Logger**: Single logging instance

### Worker Pool Pattern
- **WorkerScheduler**: Manages concurrent metric collection
- **ExportWorker**: Handles metric export in background

## Error Handling Strategy

```go
type CollectorError struct {
    CollectorName string
    Region        string
    Operation     string
    Err           error
    Retryable     bool
}

func (ce *CollectorError) Error() string {
    return fmt.Sprintf("collector %s failed in region %s during %s: %v", 
        ce.CollectorName, ce.Region, ce.Operation, ce.Err)
}

func (ce *CollectorError) IsRetryable() bool {
    return ce.Retryable
}
```

This design promotes:
- **Modularity**: Clear separation of concerns
- **Testability**: Interfaces enable easy mocking
- **Extensibility**: New collectors can be added easily
- **Maintainability**: SOLID principles throughout
- **Performance**: Concurrent collection with worker pools
- **Reliability**: Comprehensive error handling and health checks