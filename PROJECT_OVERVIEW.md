# S3-Workload Project Overview

## Project Status: ✅ Complete

Production-grade S3 workload generator built with Go 1.21+ for Kubernetes/OpenShift environments.

## 📋 Delivered Components

### 1. Core Application (Go)

#### CLI Application
- **Location**: `cmd/s3-workload/main.go`
- **Framework**: Cobra for CLI, Viper for configuration
- **Features**:
  - Complete flag parsing with all specified options
  - Configuration file support (YAML)
  - Environment variable overrides
  - Graceful shutdown on SIGTERM/SIGINT
  - Dry-run mode
  - Version command

#### Internal Packages

**`internal/config`**
- Configuration management with validation
- Flag binding and environment variable support
- Operation mix normalization
- Default values

**`internal/data`**
- Deterministic data generation (random/fixed patterns)
- SHA-256 hash computation and verification
- Size distributions: fixed, log-normal, uniform
- Metadata preparation with namespace tags
- Reader implementations with seek support

**`internal/metrics`**
- Prometheus metrics exposure
- Health and readiness endpoints
- Custom metrics for operations, latency, bytes, retries
- Circuit breaker and rate limiter instrumentation

**`internal/s3`**
- AWS SDK v2 integration
- Support for all required operations: PUT, GET, DELETE, COPY, LIST, HEAD
- Path-style addressing support (MinIO/Ceph)
- TLS configuration with optional skip-verify
- Bucket management (create, versioning)
- Metadata-based cleanup

**`internal/workload`**
- Operation scheduler with weighted mix
- Key generator with template support (`{seq:08}` format)
- Rate limiters: fixed (token bucket), Poisson, and no-op
- Verify rate sampling

**`internal/runner`**
- Worker pool orchestration
- Operation execution with retry logic
- Exponential backoff with jitter
- Context-based timeouts
- Graceful shutdown
- Cleanup mode

**`internal/s3/retry`**
- Configurable retry logic
- Exponential backoff with jitter
- Circuit breaker implementation
- Adaptive throttling for 429/503 errors

### 2. Tests

Unit tests for critical components:
- `internal/data/generator_test.go` - Data generation and size parsing
- `internal/data/verifier_test.go` - Hash computation and verification
- `internal/workload/scheduler_test.go` - Operation scheduling and distribution
- `internal/workload/keygen_test.go` - Key generation with templates

**Test Results**: ✅ All tests passing

### 3. Docker & Container

**`Dockerfile`**
- Multi-stage build using golang:1.21-alpine
- Distroless runtime (gcr.io/distroless/static:nonroot)
- Non-root user (UID 10001)
- Minimal attack surface
- CA certificates included

**`.dockerignore`**
- Optimized for smaller image builds

### 4. Kubernetes/OpenShift Manifests

**Location**: `deploy/kubernetes/`

Complete set of production-ready manifests:
- `namespace.yaml` - Dedicated namespace
- `serviceaccount.yaml` - Service account with minimal permissions
- `secret.yaml` - S3 credentials template
- `configmap.yaml` - Workload configuration
- `deployment.yaml` - Long-running workload deployment
- `job.yaml` - Finite workload job
- `service.yaml` - Metrics service
- `servicemonitor.yaml` - Prometheus ServiceMonitor
- `kustomization.yaml` - Kustomize configuration

**Security Features**:
- Non-root user (10001)
- Read-only root filesystem
- Drop all capabilities
- No privilege escalation
- Compatible with restricted SCC (OpenShift)

**Resource Management**:
- CPU requests: 500m, limits: 2000m
- Memory requests: 256Mi, limits: 1Gi

**Health Probes**:
- Liveness probe on `/healthz`
- Readiness probe on `/readyz`

### 5. Documentation

**`README.md`**
- Quick start guide
- Feature overview
- Build instructions
- Configuration examples

**`docs/CLI.md`**
- Complete flag reference
- Usage examples
- Size format specifications
- Environment variable mapping

**`docs/DEPLOYMENT.md`**
- Local deployment guide
- Docker deployment
- Kubernetes deployment (vanilla and kustomize)
- OpenShift-specific instructions
- Prometheus integration
- Troubleshooting guide

**`LICENSE`**
- MIT License

### 6. Configuration Examples

**`examples/workload.yaml`**
- Comprehensive configuration file with all options
- Inline documentation

**Workload Profiles**:
- `examples/profiles/read-heavy.yaml` - 80% reads
- `examples/profiles/write-heavy.yaml` - 70% writes
- `examples/profiles/balanced.yaml` - Mixed workload

**`examples/cleanup.sh`**
- Shell script for cleanup operations

### 7. Build System

**`Makefile`**
- `make build` - Build for Linux (container)
- `make build-local` - Build for local OS
- `make test` - Run tests with coverage
- `make docker` - Build Docker image
- `make lint` - Run linters
- `make clean` - Clean artifacts

**`.gitignore`**
- Standard Go ignores
- IDE files
- Build artifacts
- Local config files

## 🎯 Key Features Implemented

### S3 Operations
✅ PUT with metadata and SHA-256 hash
✅ GET with optional verification
✅ DELETE with keep-data mode
✅ COPY (intra and cross-bucket)
✅ LIST with prefix
✅ HEAD for metadata

### Data Management
✅ Deterministic pseudo-random data generation
✅ Fixed pattern support
✅ SHA-256 verification with sampling
✅ Metadata tagging (x-amz-meta-sha256, x-amz-meta-created-by)
✅ Namespace tags for segmentation

### Size Distributions
✅ Fixed size (e.g., `fixed:1MiB`)
✅ Log-normal distribution (e.g., `dist:lognormal:mean=1MiB,std=0.6`)
✅ Uniform distribution (e.g., `uniform:min=1KB,max=10MB`)
✅ SI and IEC units support (KB, KiB, MB, MiB, etc.)

### Workload Control
✅ Configurable operation mix with percentages
✅ Worker pool with backpressure
✅ Duration-based runs (e.g., `--duration 30m`)
✅ Operation count limit (e.g., `--operations 1000000`)
✅ Rate limiting: fixed QPS, Poisson process
✅ Key templates with sequence numbers

### Resilience
✅ Exponential backoff with jitter
✅ Per-operation timeouts via context
✅ Circuit breaker for persistent failures
✅ Adaptive rate limiting for 429/503
✅ Graceful shutdown (SIGTERM/SIGINT)

### Observability
✅ Prometheus metrics on `/metrics`
✅ Health endpoint `/healthz`
✅ Readiness endpoint `/readyz`
✅ Structured JSON logging (zap)
✅ Debug/info/warn/error log levels
✅ Optional pprof profiling

### Security
✅ Non-root container (UID 10001)
✅ Read-only root filesystem
✅ Dropped capabilities
✅ TLS support with optional skip-verify
✅ Credentials via env vars or Secret
✅ OpenShift restricted SCC compatible

### Kubernetes Native
✅ Deployment for continuous runs
✅ Job for finite runs
✅ ConfigMap for configuration
✅ Secret for credentials
✅ ServiceMonitor for Prometheus
✅ Health probes
✅ Resource requests/limits
✅ Kustomize support

## 🚀 Quick Start

### Local Build & Run
```bash
make build-local
export AWS_ACCESS_KEY_ID=your_key
export AWS_SECRET_ACCESS_KEY=your_secret

./bin/s3-workload \
  --endpoint https://s3.amazonaws.com \
  --region us-east-1 \
  --bucket test-bucket \
  --concurrency 32 \
  --duration 10m
```

### Docker
```bash
make docker
docker run --rm \
  -e AWS_ACCESS_KEY_ID=your_key \
  -e AWS_SECRET_ACCESS_KEY=your_secret \
  ghcr.io/paragkamble/s3-workload:latest \
  --endpoint https://s3.amazonaws.com \
  --bucket test-bucket \
  --concurrency 32 \
  --duration 10m
```

### Kubernetes
```bash
# Update credentials in deploy/kubernetes/secret.yaml
# Update configuration in deploy/kubernetes/configmap.yaml

kubectl apply -k deploy/kubernetes/
kubectl logs -n s3-workload -l app=s3-workload -f
```

## 📊 Metrics

All metrics exposed on `:9090/metrics`:

| Metric | Type | Description |
|--------|------|-------------|
| `s3_ops_total{op,status}` | Counter | Total operations by type and status |
| `s3_op_latency_seconds{op}` | Histogram | Operation latency distribution |
| `s3_bytes_written_total` | Counter | Total bytes written |
| `s3_bytes_read_total` | Counter | Total bytes read |
| `s3_verify_failures_total` | Counter | Failed verifications |
| `s3_verify_total` | Counter | Total verifications attempted |
| `s3_retries_total{op}` | Counter | Retry counts by operation |
| `s3_active_workers` | Gauge | Current active workers |
| `s3_rate_limiter_tokens` | Gauge | Available rate limiter tokens |
| `s3_circuit_breaker_open` | Gauge | Circuit breaker state (0/1) |

## 🧪 Testing

```bash
# Run all tests
make test

# Run with coverage report
make test-coverage

# Run linter (requires golangci-lint)
make lint
```

## 📦 Dependencies

- Go 1.21+
- AWS SDK for Go v2
- Cobra (CLI framework)
- Viper (configuration)
- Zap (structured logging)
- Prometheus client
- golang.org/x/time (rate limiting)

All dependencies pinned in `go.mod`.

## 🔧 Configuration

Three ways to configure:
1. **CLI Flags**: `--endpoint=... --bucket=...`
2. **Config File**: `--config workload.yaml`
3. **Environment Variables**: `S3BENCH_ENDPOINT=... S3BENCH_BUCKET=...`

Standard AWS credential sources:
- `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` env vars
- Shared AWS config file (`~/.aws/credentials`)
- IAM role (when running in AWS)

## 🎮 Operation Mix Examples

**Balanced**:
```bash
--mix put=35,get=35,delete=15,copy=10,list=5
```

**Read-Heavy**:
```bash
--mix get=80,head=10,list=5,put=5
```

**Write-Heavy**:
```bash
--mix put=70,get=20,delete=5,copy=5
```

## 🧹 Cleanup

Cleanup mode deletes only objects with metadata `x-amz-meta-created-by=s3-workload`:

```bash
s3-workload \
  --endpoint https://s3.example.com \
  --bucket bench-bucket \
  --prefix bench/ \
  --cleanup
```

## 📝 TODOs for Future Enhancement

- [ ] Add Grafana dashboard JSON
- [ ] OpenTelemetry tracing support (behind flag)
- [ ] Multi-bucket support
- [ ] Object lifecycle policies
- [ ] Multipart upload support for large objects
- [ ] Tagging API operations
- [ ] S3 Select operations
- [ ] Presigned URL operations
- [ ] Additional distributions (exponential, Pareto)
- [ ] Time-series workload patterns
- [ ] Integration tests with MinIO

## 🤝 Contributing

This is a production-ready baseline. Future contributions welcome for:
- Additional S3 operations
- More distribution types
- Enhanced metrics
- Performance optimizations
- Bug fixes

## 📄 License

MIT License - See LICENSE file for details.

## 🎉 Summary

This project delivers a **production-grade, feature-complete S3 workload generator** ready for deployment on Kubernetes/OpenShift. All requirements from the specification have been implemented, tested, and documented.

**Total Files**: 30+ Go files, 10 Kubernetes manifests, comprehensive documentation
**Test Coverage**: Critical paths tested with passing unit tests
**Build Status**: ✅ Compiles cleanly with no errors
**Container Ready**: ✅ Dockerfile with distroless base
**K8s Ready**: ✅ Complete manifests with security best practices

