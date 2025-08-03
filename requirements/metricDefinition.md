I want to build a application which uses aws sdk to get the resources and convert it
to opentelemetry metrics and send it to a opentelmetry collector.

## Requirements for the application:
- language of the application should be go.
- It should have a working docker setup with Dockerfile and docker-compose files.
- It should have linting setup.
- It should have Tests covering every aspect.
- It should Follow SOLID Principles.
- It should have a proper README.md.
- It should have github ci actions.
- It should not make any aws calls which incures costs.
- It should read AWS credentials and configuration from config.yaml file.
- It should read OTEL_COLLECTOR_ENDPOINT, OTEL_SERVICE_NAME from config.yaml file.
- It should also have conig file where the collection of metrics can be enabled or disabled.

Following is the defnitions of the metrics

### EC2 Resource

info:
    It should get the count of ec2 by region, state and type

Type: Gauge
Name: ec2
MetricUnit: count
MetricDescription: EC2 Count
Value: {Number of ec2 instance with filters}

Attributes:
- region: The aws region
- type: Type of ec2 instance
- state: state of ec2 instance

Claude prompt: It should get per region per type count, also per region total count of ec2(type should be all in this case).

### RDS Resource

info:
    It should get the count of RDS instances by region, engine, and status

Type: Gauge
Name: rds
MetricUnit: count
MetricDescription: RDS Instance Count
Value: {Number of RDS instances with filters}

Attributes:
- region: The aws region
- engine: RDS engine type (mysql, postgres, aurora, etc.)
- status: Status of RDS instance (available, stopped, etc.)

Claude prompt: It should get per region per engine count, also per region total count of RDS instances.

### S3 Resource

info:
    It should get the count of S3 buckets by region

Type: Gauge
Name: s3_buckets
MetricUnit: count
MetricDescription: S3 Bucket Count
Value: {Number of S3 buckets}

Attributes:
- region: The aws region

Claude prompt: It should get per region count of S3 buckets.

### Lambda Resource

info:
    It should get the count of Lambda functions by region, runtime, and state

Type: Gauge
Name: lambda_functions
MetricUnit: count
MetricDescription: Lambda Function Count
Value: {Number of Lambda functions with filters}

Attributes:
- region: The aws region
- runtime: Lambda runtime (nodejs, python, go, etc.)
- state: State of Lambda function (Active, Inactive, etc.)

Claude prompt: It should get per region per runtime count, also per region total count of Lambda functions.

### EBS Resource

info:
    It should get the count of EBS volumes by region, type, and state

Type: Gauge
Name: ebs_volumes
MetricUnit: count
MetricDescription: EBS Volume Count
Value: {Number of EBS volumes with filters}

Attributes:
- region: The aws region
- type: EBS volume type (gp2, gp3, io1, etc.)
- state: State of EBS volume (available, in-use, etc.)

Claude prompt: It should get per region per type count, also per region total count of EBS volumes.

### ELB Resource

info:
    It should get the count of Load Balancers by region, type, and state

Type: Gauge
Name: load_balancers
MetricUnit: count
MetricDescription: Load Balancer Count
Value: {Number of Load Balancers with filters}

Attributes:
- region: The aws region
- type: Load Balancer type (application, network, classic)
- state: State of Load Balancer (active, provisioning, etc.)

Claude prompt: It should get per region per type count, also per region total count of Load Balancers.

### VPC Resource

info:
    It should get the count of VPCs by region and state

Type: Gauge
Name: vpcs
MetricUnit: count
MetricDescription: VPC Count
Value: {Number of VPCs with filters}

Attributes:
- region: The aws region
- state: State of VPC (available, pending, etc.)

Claude prompt: It should get per region count of VPCs with state information.

