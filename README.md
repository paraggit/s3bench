# s3-workload

A production-grade S3 workload generator for benchmarking and testing object storage systems (AWS S3, MinIO, Ceph RGW, OpenShift ODF, etc.) in Kubernetes/OpenShift environments.

## ⭐ OpenShift ODF/RGW Support

Fully compatible with **OpenShift Data Foundation (ODF)** and **Ceph Rados Gateway (RGW)**. See [docs/ODF_RGW_SETUP.md](docs/ODF_RGW_SETUP.md) for detailed setup guide.

## Features

- **Comprehensive S3 Operations**: PUT, GET, DELETE, COPY, LIST, HEAD, MULTIPART_PUT with configurable operation mix
- **Multipart Upload Support**: Efficient upload of large objects with configurable part size and concurrent part uploads
- **Data Verification**: Deterministic data generation with SHA-256 verification
- **Flexible Configuration**: Object size distributions, keyspace control, operation mix percentages
- **Production-Ready**: Prometheus metrics, health endpoints, structured logging, graceful shutdown
- **Kubernetes Native**: Runs as Job or Deployment with full OpenShift support
- **Resilient**: Retries with exponential backoff, circuit breaker, adaptive rate limiting
- **Secure**: Non-root container, TLS support, credential management via Secrets

## Quick Start

### Local Build

```bash
make build-local
./bin/s3-workload --endpoint https://s3.amazonaws.com \
  --region us-east-1 \
  --bucket my-test-bucket \
  --concurrency 32 \
  --mix put=50,get=40,delete=10 \
  --duration 5m
```

### Docker Build

**Simple build:**
```bash
make docker
docker run --rm ghcr.io/paragkamble/s3-workload:latest --help
```

**Using build scripts** (recommended):
```bash
# Build locally
./scripts/build-image.sh

# Build and push to registry
./scripts/build-image.sh --version v1.0.0 --push

# Build for multiple architectures
./scripts/build-multiarch.sh

# Or use make targets
make docker-build-push VERSION=v1.0.0
```

See [scripts/README.md](scripts/README.md) for detailed build instructions.

### Kubernetes Deployment

```bash
kubectl apply -f deploy/kubernetes/
```

### OpenShift ODF/RGW Deployment

#### Quick Deploy (Automated Script)

```bash
# Automatic deployment with credential detection
cd examples
./deploy-odf-rgw.sh

# Or with custom credentials
export RGW_ACCESS_KEY=your_access_key
export RGW_SECRET_KEY=your_secret_key
./deploy-odf-rgw.sh
```

#### Manual Deployment

```bash
# Quick start for OpenShift ODF
oc new-project s3-workload
oc create secret generic s3-creds \
  --from-literal=accessKey=YOUR_RGW_ACCESS_KEY \
  --from-literal=secretKey=YOUR_RGW_SECRET_KEY

# Deploy with ODF-specific configuration
oc apply -f deploy/kubernetes/namespace.yaml
oc apply -f deploy/kubernetes/serviceaccount.yaml
oc apply -f deploy/kubernetes/configmap-odf-rgw.yaml
oc apply -f deploy/kubernetes/deployment-odf-rgw.yaml
oc apply -f deploy/kubernetes/service.yaml

# Follow logs
oc logs -n s3-workload -l app=s3-workload -f
```

See [docs/ODF_RGW_SETUP.md](docs/ODF_RGW_SETUP.md) for complete setup instructions.

## CLI Reference

```bash
s3-workload \
  --endpoint https://rgw.example:443 \
  --region us-east-1 \
  --bucket bench-bucket \
  --create-bucket \
  --concurrency 64 \
  --mix put=40,get=40,delete=10,copy=5,list=5 \
  --size dist:lognormal:mean=1MiB,std=0.6 \
  --keys 100000 --prefix bench/ --key-template "obj-{seq:08}.bin" \
  --pattern random:42 --verify-rate 0.2 \
  --duration 30m \
  --versioning off \
  --path-style \
  --metrics-port 9090 \
  --log-level info
```

See [docs/CLI.md](docs/CLI.md) for full flag reference.

### ODF/RGW Example

```bash
s3-workload \
  --endpoint https://s3.openshift-storage.svc.cluster.local \
  --region us-east-1 \
  --bucket odf-bench-bucket \
  --path-style \
  --create-bucket \
  --concurrency 64 \
  --mix put=40,get=40,delete=10,copy=5,list=5 \
  --duration 30m \
  --config examples/profiles/odf-rgw-balanced.yaml
```

## Configuration

Configuration can be provided via:
1. Command-line flags
2. YAML configuration file (`--config workload.yaml`)
3. Environment variables (prefix with `S3BENCH_`)

Example `workload.yaml`:
```yaml
endpoint: https://s3.amazonaws.com
region: us-east-1
bucket: bench-bucket
concurrency: 64
mix:
  put: 40
  get: 40
  delete: 10
  copy: 5
  list: 5
```

### Workload Profiles

Pre-configured profiles available in `examples/profiles/`:
- `balanced.yaml` - General purpose workload (50% PUT, 50% GET)
- `read-heavy.yaml` - Read-intensive workload (80% reads)
- `write-heavy.yaml` - Write-intensive workload (70% writes)
- **`multipart-large-objects.yaml`** - Multipart upload testing for large objects (500 MiB avg)
- **`odf-rgw-balanced.yaml`** - Balanced workload optimized for ODF/RGW
- **`odf-rgw-read-heavy.yaml`** - Read-heavy workload for RGW
- **`odf-rgw-write-heavy.yaml`** - Write-heavy workload for RGW
- **`odf-rgw-large-objects.yaml`** - Large object testing for RGW with multipart upload

Use with: `--config examples/profiles/odf-rgw-balanced.yaml`

## Multipart Upload

The tool supports multipart upload for efficient handling of large objects. Multipart upload can be enabled in two ways:

### 1. Automatic Multipart Upload (Recommended)

Enable automatic multipart upload for objects exceeding a size threshold:

```bash
s3-workload \
  --endpoint https://s3.amazonaws.com \
  --bucket my-bucket \
  --multipart-enabled \
  --multipart-threshold 104857600 \      # 100 MiB threshold
  --multipart-part-size 10485760 \       # 10 MiB parts
  --multipart-max-parts 4 \              # 4 concurrent part uploads
  --size fixed:500MiB \
  --mix put=100
```

When enabled, PUT operations for objects larger than the threshold will automatically use multipart upload.

### 2. Explicit Multipart Upload

Use `multipart_put` in the operation mix to always use multipart upload:

```bash
s3-workload \
  --endpoint https://s3.amazonaws.com \
  --bucket my-bucket \
  --multipart-part-size 10485760 \
  --multipart-max-parts 8 \
  --size fixed:1GiB \
  --mix multipart_put=50,get=50
```

### Multipart Upload Parameters

- `--multipart-enabled`: Enable automatic multipart upload for large objects (default: false)
- `--multipart-threshold`: Size threshold in bytes to trigger multipart upload (default: 100 MiB, min: 5 MiB)
- `--multipart-part-size`: Size of each multipart part in bytes (default: 10 MiB, min: 5 MiB)
- `--multipart-max-parts`: Maximum number of parts to upload concurrently (default: 4, max: 10000)

### Example Configuration for Large Objects

```yaml
endpoint: https://s3.amazonaws.com
region: us-east-1
bucket: large-object-bench
concurrency: 16
duration: 30m

# Object configuration
size: "dist:lognormal:mean=500MiB,std=0.5"
keys: 1000
prefix: "large-objects/"

# Multipart configuration
multipart_enabled: true
multipart_threshold: 104857600    # 100 MiB
multipart_part_size: 52428800     # 50 MiB parts
multipart_max_parts: 8            # 8 concurrent uploads

# Operation mix
mix:
  put: 40
  get: 40
  delete: 10
  list: 10
```

### Performance Tips

1. **Part Size**: Larger part sizes reduce API calls but increase memory usage. The optimal size depends on your network and object storage configuration.
2. **Concurrency**: Higher concurrent part uploads increase throughput but consume more network bandwidth and memory. Start with 4-8 and adjust based on your environment.
3. **Threshold**: Set the threshold based on your typical object sizes. Objects below the threshold use regular PUT operations.

## Metrics

Exposed on `/metrics` (default port 9090):

- `s3_ops_total{op,status}` - Total operations by type and status
- `s3_op_latency_seconds{op}` - Operation latency histogram
- `s3_bytes_written_total`, `s3_bytes_read_total` - Data transferred
- `s3_verify_failures_total` - Verification failures
- `s3_retries_total{op}` - Retry counts
- `s3_active_workers` - Current active workers

## Development

```bash
# Download dependencies
make deps

# Run tests
make test

# Generate coverage report
make test-coverage

# Lint code
make lint

# Format code
make fmt
```

## License

MIT License - See [LICENSE](LICENSE) file for details.

