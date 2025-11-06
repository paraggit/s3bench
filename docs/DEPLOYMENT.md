# Deployment Guide

## Local Deployment

### Build Locally

```bash
make build-local
./bin/s3-workload --help
```

### Run Locally

```bash
export AWS_ACCESS_KEY_ID=your_access_key
export AWS_SECRET_ACCESS_KEY=your_secret_key

./bin/s3-workload \
  --endpoint https://s3.amazonaws.com \
  --region us-east-1 \
  --bucket my-bucket \
  --concurrency 32 \
  --duration 10m
```

## Docker Deployment

### Build Docker Image

```bash
make docker
```

### Run with Docker

```bash
docker run --rm \
  -e AWS_ACCESS_KEY_ID=your_access_key \
  -e AWS_SECRET_ACCESS_KEY=your_secret_key \
  ghcr.io/paragkamble/s3-workload:latest \
  --endpoint https://s3.amazonaws.com \
  --region us-east-1 \
  --bucket my-bucket \
  --concurrency 32 \
  --duration 10m
```

## Kubernetes Deployment

### Prerequisites

- Kubernetes cluster (1.20+)
- kubectl configured
- S3-compatible storage endpoint

### Quick Start

1. **Update credentials:**

```bash
# Edit deploy/kubernetes/secret.yaml
# Replace YOUR_ACCESS_KEY and YOUR_SECRET_KEY with actual values
```

2. **Update configuration:**

```bash
# Edit deploy/kubernetes/configmap.yaml
# Set endpoint, bucket, and other parameters
```

3. **Deploy:**

```bash
kubectl apply -k deploy/kubernetes/
```

4. **Check status:**

```bash
kubectl get pods -n s3-workload
kubectl logs -n s3-workload -l app=s3-workload -f
```

5. **Access metrics:**

```bash
kubectl port-forward -n s3-workload svc/s3-workload-metrics 9090:9090
# Open http://localhost:9090/metrics
```

### Deployment Options

#### Long-Running Deployment

Use `deployment.yaml` for continuous workload:

```bash
kubectl apply -f deploy/kubernetes/deployment.yaml
```

#### Finite Job

Use `job.yaml` for one-time workload:

```bash
kubectl apply -f deploy/kubernetes/job.yaml
kubectl wait --for=condition=complete job/s3-workload-job -n s3-workload
```

### Customization

#### Using Kustomize

Create `kustomization.yaml`:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - github.com/paragkamble/s3bench/deploy/kubernetes

namespace: my-namespace

configMapGenerator:
  - name: s3-workload-config
    behavior: merge
    literals:
      - endpoint=https://my-s3.example.com
      - bucket=my-bucket
      - concurrency=128
```

Apply:

```bash
kubectl apply -k .
```

## OpenShift Deployment

### Prerequisites

- OpenShift cluster (4.10+)
- oc CLI configured
- S3-compatible storage (e.g., **ODF/Ceph RGW**, MinIO, or external S3)

> **💡 For OpenShift ODF/RGW:** See comprehensive setup guide at [ODF_RGW_SETUP.md](ODF_RGW_SETUP.md)

### Deploy to OpenShift

1. **Create project:**

```bash
oc new-project s3-workload
```

2. **Create secret:**

```bash
oc create secret generic s3-creds \
  --from-literal=accessKey=your_access_key \
  --from-literal=secretKey=your_secret_key
```

3. **Create config:**

For ODF/RGW (recommended):
```bash
# Use the ODF-specific ConfigMap
oc apply -f deploy/kubernetes/configmap-odf-rgw.yaml
```

Or create manually:
```bash
oc create configmap s3-workload-config \
  --from-literal=endpoint=https://s3.openshift-storage.svc.cluster.local \
  --from-literal=region=us-east-1 \
  --from-literal=bucket=bench-bucket \
  --from-literal=path-style=true \
  --from-literal=skip-tls-verify=false \
  --from-literal=concurrency=64 \
  --from-literal=duration=30m \
  --from-literal=mix="put=40,get=40,delete=10,copy=5,list=5"
```

4. **Deploy:**

For ODF/RGW (recommended):
```bash
# Deploy all resources including ServiceAccount, ConfigMap, Deployment, and Service
oc apply -f deploy/kubernetes/serviceaccount.yaml
oc apply -f deploy/kubernetes/configmap-odf-rgw.yaml
oc apply -f deploy/kubernetes/deployment-odf-rgw.yaml
oc apply -f deploy/kubernetes/service.yaml
```

For other S3 services:
```bash
oc apply -f deploy/kubernetes/deployment.yaml
```

5. **Expose metrics (optional):**

```bash
oc expose svc/s3-workload-metrics
oc get route s3-workload-metrics
```

### OpenShift Considerations

- **Security Context Constraints (SCC):** The deployment uses non-root user (10001) and read-only root filesystem, compatible with restricted SCC.

- **ODF/RGW Specific Settings:**
  - Always use `--path-style=true` for RGW compatibility
  - Internal endpoint: `https://s3.openshift-storage.svc.cluster.local`
  - For external access, create a route or use the existing RGW route
  - See [ODF_RGW_SETUP.md](ODF_RGW_SETUP.md) for credential setup

- **Service Mesh:** If using OpenShift Service Mesh, add sidecar injection:

```yaml
metadata:
  annotations:
    sidecar.istio.io/inject: "true"
```

- **Routes:** Expose metrics externally if needed:

```bash
oc create route edge s3-workload-metrics \
  --service=s3-workload-metrics \
  --port=http
```

### OpenShift ODF/RGW Quick Reference

```bash
# Get RGW endpoint
oc get route -n openshift-storage | grep s3

# Get RGW credentials (if using Noobaa)
oc get secret noobaa-admin -n openshift-storage -o jsonpath='{.data.AWS_ACCESS_KEY_ID}' | base64 -d

# Create RGW user (if using Ceph directly)
TOOLS_POD=$(oc get pods -n openshift-storage -l app=rook-ceph-tools -o jsonpath='{.items[0].metadata.name}')
oc exec -n openshift-storage $TOOLS_POD -- radosgw-admin user create --uid=s3-benchmark

# Test connectivity
oc run -it --rm test-s3 --image=amazon/aws-cli --restart=Never -- \
  s3 ls --endpoint-url=https://s3.openshift-storage.svc.cluster.local
```

For complete ODF/RGW setup instructions, see [ODF_RGW_SETUP.md](ODF_RGW_SETUP.md).

## Prometheus Integration

### ServiceMonitor

If using Prometheus Operator:

```bash
kubectl apply -f deploy/kubernetes/servicemonitor.yaml
```

### Manual Scrape Config

Add to Prometheus config:

```yaml
scrape_configs:
  - job_name: 's3-workload'
    static_configs:
      - targets: ['s3-workload-metrics.s3-workload.svc:9090']
```

## Monitoring

### Key Metrics

- `s3_ops_total{op,status}` - Total operations
- `s3_op_latency_seconds{op}` - Operation latency
- `s3_bytes_written_total` - Bytes written
- `s3_bytes_read_total` - Bytes read
- `s3_verify_failures_total` - Verification failures
- `s3_active_workers` - Active workers

### Grafana Dashboard

Import example dashboard from `examples/grafana-dashboard.json` (TODO).

## Scaling

### Horizontal Scaling

Scale deployment:

```bash
kubectl scale deployment s3-workload --replicas=5 -n s3-workload
```

### Vertical Scaling

Adjust resources in deployment:

```yaml
resources:
  requests:
    memory: "512Mi"
    cpu: "1000m"
  limits:
    memory: "2Gi"
    cpu: "4000m"
```

## Troubleshooting

### Check Logs

```bash
kubectl logs -n s3-workload -l app=s3-workload -f
```

### Check Health

```bash
kubectl get pods -n s3-workload
kubectl describe pod <pod-name> -n s3-workload
```

### Debug Mode

Enable debug logging:

```bash
kubectl set env deployment/s3-workload LOG_LEVEL=debug -n s3-workload
```

### Common Issues

1. **Connection Refused**: Check endpoint URL and network policies
2. **Authentication Failed**: Verify credentials in secret
3. **Timeout**: Adjust `op_timeout` in config
4. **High Error Rate**: Check S3 backend capacity and network

## Cleanup

### Remove Deployment

```bash
kubectl delete -k deploy/kubernetes/
```

### Cleanup S3 Objects

```bash
kubectl run s3-workload-cleanup --rm -it --restart=Never \
  --image=ghcr.io/paragkamble/s3-workload:latest \
  --env AWS_ACCESS_KEY_ID=xxx \
  --env AWS_SECRET_ACCESS_KEY=xxx \
  -- \
  --endpoint https://s3.example.com \
  --bucket bench-bucket \
  --prefix bench/ \
  --cleanup
```

