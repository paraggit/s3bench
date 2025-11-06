# CLI Reference

## Global Flags

### S3 Connection

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--endpoint` | string | required | S3 endpoint URL (e.g., https://s3.amazonaws.com) |
| `--region` | string | us-east-1 | AWS region |
| `--bucket` | string | required | S3 bucket name |
| `--access-key` | string | - | AWS access key (or use AWS_ACCESS_KEY_ID env) |
| `--secret-key` | string | - | AWS secret key (or use AWS_SECRET_ACCESS_KEY env) |
| `--path-style` | bool | false | Use path-style addressing (required for MinIO/Ceph) |
| `--skip-tls-verify` | bool | false | Skip TLS certificate verification (insecure!) |

### Bucket Management

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--create-bucket` | bool | false | Create bucket if it doesn't exist |
| `--versioning` | string | keep | Bucket versioning: on, off, or keep |

### Workload Parameters

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--concurrency` | int | 32 | Number of concurrent workers |
| `--mix` | map | put=50,get=50 | Operation mix (e.g., put=40,get=40,delete=10,copy=5,list=5) |
| `--duration` | duration | 10m | Workload duration (0 for unlimited) |
| `--operations` | int64 | 0 | Total operations (0 for unlimited, overrides duration) |

### Object Configuration

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--size` | string | fixed:1MiB | Object size: fixed:1MiB or dist:lognormal:mean=1MiB,std=0.6 |
| `--keys` | int | 10000 | Number of unique keys in keyspace |
| `--prefix` | string | "" | Key prefix |
| `--key-template` | string | obj-{seq:08}.bin | Key template with {seq} or {seq:08} placeholder |
| `--random-keys` | bool | false | Use random key selection instead of sequential |

### Data Pattern & Verification

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--pattern` | string | random:42 | Data pattern: random:<seed> or fixed:<hex> |
| `--verify-rate` | float64 | 0.1 | Fraction of GETs to verify (0.0-1.0) |

### Rate Limiting

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--rate-type` | string | fixed | Rate limiter type: fixed or poisson |
| `--rate-limit` | float64 | 0 | Rate limit (QPS for fixed, lambda for poisson; 0=unlimited) |

### Timeouts & Retries

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--op-timeout` | duration | 30s | Per-operation timeout |
| `--max-retries` | int | 3 | Maximum retry attempts |
| `--retry-backoff` | duration | 100ms | Initial retry backoff |

### Copy Operation

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--copy-dst-bucket` | string | "" | Destination bucket for COPY operations (same bucket if empty) |

### Safety & Cleanup

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--namespace-tag` | string | "" | Namespace tag for object metadata (e.g., env=perf) |
| `--keep-data` | bool | false | Keep data (skip cleanup deletes) |
| `--cleanup` | bool | false | Cleanup mode: delete only tool-created objects |
| `--dry-run` | bool | false | Dry run: print config and exit |

### Observability

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--metrics-port` | int | 9090 | Prometheus metrics port |
| `--http-bind` | string | 0.0.0.0 | HTTP bind address |
| `--log-level` | string | info | Log level: debug, info, warn, error |
| `--pprof-port` | int | 0 | Pprof port (0 to disable) |

### Configuration File

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | "" | Config file path (YAML) |

## Examples

### Basic Workload

```bash
s3-workload \
  --endpoint https://s3.amazonaws.com \
  --region us-east-1 \
  --bucket my-bucket \
  --concurrency 32 \
  --duration 10m
```

### Advanced Workload with Mix

```bash
s3-workload \
  --endpoint https://rgw.example:443 \
  --region us-east-1 \
  --bucket bench-bucket \
  --create-bucket \
  --concurrency 64 \
  --mix put=40,get=40,delete=10,copy=5,list=5 \
  --size dist:lognormal:mean=1MiB,std=0.6 \
  --keys 100000 \
  --prefix bench/ \
  --pattern random:42 \
  --verify-rate 0.2 \
  --duration 30m \
  --path-style
```

### Read-Heavy Workload

```bash
s3-workload \
  --endpoint https://minio:9000 \
  --bucket demo \
  --mix get=95,head=5 \
  --concurrency 256 \
  --duration 30m \
  --keep-data
```

### Cleanup Mode

```bash
s3-workload \
  --endpoint https://rgw:443 \
  --bucket bench \
  --prefix bench/ \
  --cleanup
```

### Using Config File

```bash
s3-workload --config workload.yaml
```

### Dry Run

```bash
s3-workload --config workload.yaml --dry-run
```

## Environment Variables

All flags can be set via environment variables with `S3BENCH_` prefix:

```bash
export S3BENCH_ENDPOINT=https://s3.amazonaws.com
export S3BENCH_REGION=us-east-1
export S3BENCH_BUCKET=my-bucket
export AWS_ACCESS_KEY_ID=your_access_key
export AWS_SECRET_ACCESS_KEY=your_secret_key

s3-workload --concurrency 64 --duration 10m
```

## Size Formats

### Fixed Size

```
fixed:1MiB
fixed:512KB
fixed:10GB
```

### Log-Normal Distribution

```
dist:lognormal:mean=1MiB,std=0.6
dist:lognormal:mean=512KB,std=0.3
```

### Uniform Distribution

```
uniform:min=1KB,max=10MB
```

## Operation Mix

Specify percentages for each operation. They will be normalized to 100%.

```
--mix put=40,get=40,delete=10,copy=5,list=5
```

Supported operations:
- `put` - Upload object
- `get` - Download object
- `delete` - Delete object
- `copy` - Copy object
- `list` - List objects
- `head` - HEAD request (metadata only)

## Data Patterns

### Random (Deterministic)

```
--pattern random:42
```

Generates pseudo-random data based on seed. Same key always produces same data.

### Fixed Pattern

```
--pattern fixed:DEADBEEF
```

Repeats the hex pattern for all objects.

