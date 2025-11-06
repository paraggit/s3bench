# s3-workload

A production-grade S3 workload generator for benchmarking and testing object storage systems (AWS S3, MinIO, Ceph RGW, etc.) in Kubernetes/OpenShift environments.

## Features

- **Comprehensive S3 Operations**: PUT, GET, DELETE, COPY, LIST, HEAD with configurable operation mix
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

```bash
make docker
docker run --rm ghcr.io/paragkamble/s3-workload:latest --help
```

### Kubernetes Deployment

```bash
kubectl apply -f deploy/kubernetes/
```

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

## Configuration

Configuration can be provided via:
1. Command-line flags
2. YAML configuration file (`--config workload.yaml`)
3. Environment variables (see [docs/CONFIGURATION.md](docs/CONFIGURATION.md))

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

