# System Architecture and Program Flow

This document describes the overall architecture and program flow of the AWS monitoring application.

## High-Level Architecture

```
┌─────────────────┐                          ┌─────────────────┐
│   Config File   │                          │   AWS Services  │
│   (config.yaml) │                          │  (EC2, RDS,     │
│                 │                          │   S3, Lambda)   │
└─────────┬───────┘                          └─────────┬───────┘
          │                                            │
          v                                            v
┌─────────────────────────────────────────────────────────────────┐
│                     AWS Monitor Application                     │
│                                                                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │  Config Manager │  │ AWS Client      │  │   Scheduler     │ │
│  │                 │  │ Factory         │  │                 │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                Metric Collectors                            │ │
│  │  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐    │ │
│  │  │ EC2  │ │ RDS  │ │ S3   │ │Lambda│ │ EBS  │ │ ELB  │    │ │
│  │  └──────┘ └──────┘ └──────┘ └──────┘ └──────┘ └──────┘    │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                 │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │ Metric Provider │  │ Metric Exporter │  │ Health Checker  │ │
│  │    (OTEL)       │  │    (OTEL)       │  │                 │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                                │
                                v
                  ┌─────────────────────────┐
                  │  OpenTelemetry         │
                  │  Collector             │
                  └─────────────────────────┘
                                │
                                v
                  ┌─────────────────────────┐
                  │   Monitoring Systems    │
                  │ (Prometheus, Grafana,   │
                  │  Jaeger, etc.)          │
                  └─────────────────────────┘
```

## Component Architecture

### 1. Application Initialization Flow

```
main() 
├── Load Configuration
│   ├── Read config.yaml
│   └── Validate configuration
├── Initialize Logger
├── Setup AWS Client Factory
│   ├── Create AWS session
│   ├── Configure credentials
│   └── Set up region-specific clients
├── Initialize OpenTelemetry
│   ├── Setup metric provider
│   ├── Configure exporter
│   └── Start metric pipeline
├── Create and Register Collectors
│   ├── Initialize each enabled collector
│   ├── Register with collector registry
│   └── Validate collector configuration
├── Start Scheduler
│   ├── Create worker pool
│   ├── Schedule metric collection
│   └── Start health check server
└── Handle Graceful Shutdown
    ├── Stop scheduler
    ├── Drain worker pool
    ├── Flush remaining metrics
    └── Close connections
```

### 2. Metric Collection Flow

```
Scheduler Tick
├── For each enabled region
│   ├── For each enabled collector
│   │   ├── Acquire worker from pool
│   │   ├── Create collection context
│   │   ├── Execute collector.Collect()
│   │   │   ├── Get AWS client for region
│   │   │   ├── Make AWS API calls
│   │   │   ├── Process API responses
│   │   │   ├── Transform to metric data
│   │   │   └── Return metrics
│   │   ├── Send metrics to exporter
│   │   ├── Update collector health status
│   │   └── Release worker back to pool
│   └── Update region collection status
├── Export batch of metrics
│   ├── Convert to OTEL format
│   ├── Send to collector endpoint
│   └── Handle export errors
└── Update overall health status
```

### 3. Error Handling and Retry Flow

```
API Call Error
├── Classify Error Type
│   ├── Rate Limit → Wait and retry
│   ├── Credential Error → Mark collector as failed
│   ├── Network Error → Retry with backoff
│   ├── Service Error → Skip this collection cycle
│   └── Unknown Error → Log and continue
├── Update Error Metrics
├── Update Health Status
└── Continue with next collection
```

## Component Interactions

### 1. Configuration Management

```go
// Startup sequence
config := config.Load("config.yaml")
config.Validate()

// Runtime usage
if config.Metrics.EC2.Enabled {
    collector := collectors.NewEC2Collector(config.Metrics.EC2, clientFactory)
    scheduler.AddCollector(collector)
}
```

### 2. AWS Client Management

```go
// Client factory manages region-specific clients
clientFactory := aws.NewClientFactory(config.AWS)

// Collectors request clients for specific regions
for _, region := range config.EnabledRegions {
    ec2Client, err := clientFactory.GetEC2Client(region)
    metrics, err := collector.Collect(ctx, region, ec2Client)
}
```

### 3. Metric Collection Workflow

```go
// Scheduler coordinates collection
scheduler.Start(ctx)

// For each collection cycle
workItem := WorkItem{
    Collector: collector,
    Region:    region,
    Context:   ctx,
}

// Worker processes work item
worker.Process(workItem) {
    metrics, err := workItem.Collector.Collect(workItem.Context, workItem.Region)
    if err != nil {
        handleError(err)
        return
    }
    exporter.Export(workItem.Context, metrics)
}
```

### 4. Health Monitoring

```go
// Health checker aggregates status
healthChecker.Start()

// HTTP endpoint provides status
http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    status := healthChecker.GetStatus()
    json.NewEncoder(w).Encode(status)
})
```

## Data Flow Diagram

```
┌─────────────┐
│   Config    │
│   Manager   │
└──────┬──────┘
       │
       v
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│ Scheduler   │───▶│   Worker    │───▶│ Collector   │
│             │    │    Pool     │    │  Registry   │
└─────────────┘    └─────────────┘    └──────┬──────┘
       │                  │                  │
       │                  │                  v
       │                  │           ┌─────────────┐
       │                  │           │    AWS      │
       │                  │           │   Clients   │
       │                  │           └──────┬──────┘
       │                  │                  │
       │                  │                  v
       │                  │           ┌─────────────┐
       │                  │           │    AWS      │
       │                  │           │  Services   │
       │                  │           └──────┬──────┘
       │                  │                  │
       │                  │                  v
       │                  │           ┌─────────────┐
       │                  └──────────▶│   Metric    │
       │                              │    Data     │
       │                              └──────┬──────┘
       │                                     │
       │                                     v
       │                              ┌─────────────┐
       │                              │   OTEL      │
       │                              │  Exporter   │
       │                              └──────┬──────┘
       │                                     │
       │                                     v
       │                              ┌─────────────┐
       │                              │   OTEL      │
       │                              │ Collector   │
       │                              └─────────────┘
       │
       v
┌─────────────┐
│   Health    │
│  Checker    │
└─────────────┘
```

## Concurrency Model

### 1. Worker Pool Pattern

```go
type WorkerPool struct {
    workers     chan struct{}
    workQueue   chan WorkItem
    results     chan WorkResult
    wg          sync.WaitGroup
}

// Bounded concurrency
func (wp *WorkerPool) Start(maxWorkers int) {
    wp.workers = make(chan struct{}, maxWorkers)
    
    for i := 0; i < maxWorkers; i++ {
        go wp.worker()
    }
}

func (wp *WorkerPool) worker() {
    for workItem := range wp.workQueue {
        wp.workers <- struct{}{} // Acquire worker slot
        
        result := wp.processWorkItem(workItem)
        wp.results <- result
        
        <-wp.workers // Release worker slot
    }
}
```

### 2. Collector Scheduling

```go
// Each collector runs on its own schedule
func (s *Scheduler) scheduleCollector(collector MetricCollector) {
    ticker := time.NewTicker(collector.GetCollectionInterval())
    go func() {
        for {
            select {
            case <-ticker.C:
                s.scheduleCollection(collector)
            case <-s.stopChan:
                ticker.Stop()
                return
            }
        }
    }()
}
```

### 3. Graceful Shutdown

```go
func (app *App) Shutdown(ctx context.Context) error {
    // Stop accepting new work
    app.scheduler.Stop()
    
    // Wait for current work to finish
    done := make(chan struct{})
    go func() {
        app.wg.Wait()
        close(done)
    }()
    
    select {
    case <-done:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

## Performance Considerations

### 1. Resource Management
- **Connection Pooling**: Reuse AWS clients across collections
- **Memory Management**: Stream processing for large result sets
- **CPU Optimization**: Parallel processing within collectors

### 2. Rate Limiting
- **AWS API Limits**: Respect service-specific rate limits
- **Exponential Backoff**: Implement retry logic with jitter
- **Circuit Breaker**: Prevent cascade failures

### 3. Metric Batching
- **Batch Size**: Optimal batch sizes for OTEL export
- **Time Windows**: Aggregate metrics over time windows
- **Memory Limits**: Prevent memory exhaustion with large batches

## Security Architecture

### 1. Credential Management
- **Environment Variables**: Secure credential injection
- **IAM Roles**: Prefer IAM roles over access keys
- **Least Privilege**: Minimal required permissions

### 2. Network Security
- **TLS**: Encrypted communication with OTEL collector
- **Firewall Rules**: Restrict outbound connections
- **Certificate Validation**: Verify collector certificates

### 3. Data Protection
- **No PII**: Avoid collecting personally identifiable information
- **Metric Sanitization**: Clean sensitive data from metrics
- **Audit Logging**: Log security-relevant events

This architecture ensures scalability, reliability, and maintainability while following Go best practices and cloud-native patterns.