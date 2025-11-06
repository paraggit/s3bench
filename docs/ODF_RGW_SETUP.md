# OpenShift ODF RGW Setup Guide

This guide walks you through deploying the S3 workload generator against OpenShift Data Foundation (ODF) Rados Gateway (RGW) buckets.

## Prerequisites

- OpenShift cluster (4.10+) with ODF installed
- `oc` CLI configured and authenticated
- ODF MultiCloudGateway (MCG) or RGW deployed
- Cluster admin or namespace admin permissions

## Quick Start

```bash
# 1. Create namespace
oc new-project s3-workload

# 2. Get RGW credentials (see section below)
# 3. Create secret with RGW credentials
oc create secret generic s3-creds \
  --from-literal=accessKey=YOUR_RGW_ACCESS_KEY \
  --from-literal=secretKey=YOUR_RGW_SECRET_KEY

# 4. Deploy the workload
oc apply -f deploy/kubernetes/namespace.yaml
oc apply -f deploy/kubernetes/serviceaccount.yaml
oc apply -f deploy/kubernetes/configmap-odf-rgw.yaml
oc apply -f deploy/kubernetes/deployment-odf-rgw.yaml
oc apply -f deploy/kubernetes/service.yaml

# 5. Check status
oc get pods -n s3-workload
oc logs -n s3-workload -l app=s3-workload -f
```

## Step-by-Step Setup

### 1. Verify ODF Installation

Check that ODF is installed and healthy:

```bash
# Check ODF operators
oc get csv -n openshift-storage

# Check storage cluster
oc get storagecluster -n openshift-storage

# Check RGW deployment
oc get pods -n openshift-storage | grep rgw
```

### 2. Get RGW Endpoint

#### Option A: Internal Service Endpoint (Recommended)

For workloads running inside OpenShift, use the internal service endpoint:

```bash
# Get the RGW service
oc get svc -n openshift-storage | grep s3

# Typical endpoint format:
# s3.openshift-storage.svc.cluster.local
# OR
# rook-ceph-rgw-ocs-storagecluster-cephobjectstore.openshift-storage.svc.cluster.local
```

Update the ConfigMap:
```yaml
endpoint: "https://s3.openshift-storage.svc.cluster.local"
```

#### Option B: External Route

For testing from outside the cluster:

```bash
# Check if route exists
oc get route -n openshift-storage | grep s3

# If route doesn't exist, create one
oc expose svc rook-ceph-rgw-ocs-storagecluster-cephobjectstore -n openshift-storage

# Get the route URL
RGW_ROUTE=$(oc get route -n openshift-storage -o jsonpath='{.items[0].spec.host}')
echo "RGW Endpoint: https://$RGW_ROUTE"
```

Update the ConfigMap:
```yaml
endpoint: "https://s3-rgw-openshift-storage.apps.your-cluster.example.com"
```

### 3. Create RGW User and Get Credentials

#### Method 1: Using Noobaa (MCG)

If using ODF's MultiCloudGateway (Noobaa):

```bash
# Get Noobaa admin credentials
NOOBAA_ACCESS_KEY=$(oc get secret noobaa-admin -n openshift-storage -o jsonpath='{.data.AWS_ACCESS_KEY_ID}' | base64 -d)
NOOBAA_SECRET_KEY=$(oc get secret noobaa-admin -n openshift-storage -o jsonpath='{.data.AWS_SECRET_ACCESS_KEY}' | base64 -d)

echo "Access Key: $NOOBAA_ACCESS_KEY"
echo "Secret Key: $NOOBAA_SECRET_KEY"

# Get Noobaa S3 endpoint
oc get noobaa -n openshift-storage
```

#### Method 2: Using Ceph RGW Directly

Create a dedicated RGW user for benchmarking:

```bash
# Get the toolbox pod
TOOLS_POD=$(oc get pods -n openshift-storage -l app=rook-ceph-tools -o jsonpath='{.items[0].metadata.name}')

# Create RGW user
oc exec -n openshift-storage $TOOLS_POD -- radosgw-admin user create \
  --uid=s3-benchmark \
  --display-name="S3 Benchmark User" \
  --access-key=benchmark-access-key \
  --secret-key=benchmark-secret-key

# Or let it auto-generate credentials
oc exec -n openshift-storage $TOOLS_POD -- radosgw-admin user create \
  --uid=s3-benchmark \
  --display-name="S3 Benchmark User"

# Get user credentials
oc exec -n openshift-storage $TOOLS_POD -- radosgw-admin user info --uid=s3-benchmark
```

Example output:
```json
{
    "user_id": "s3-benchmark",
    "display_name": "S3 Benchmark User",
    "keys": [
        {
            "user": "s3-benchmark",
            "access_key": "ABC123DEF456GHI789",
            "secret_key": "xyz789abc123def456ghi789jkl012"
        }
    ]
}
```

### 4. Create Kubernetes Secret

```bash
# Create secret with RGW credentials
oc create secret generic s3-creds \
  --from-literal=accessKey=ABC123DEF456GHI789 \
  --from-literal=secretKey=xyz789abc123def456ghi789jkl012 \
  -n s3-workload
```

Or create from file:

```bash
# Create secret.yaml
cat <<EOF > odf-rgw-secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: s3-creds
  namespace: s3-workload
type: Opaque
stringData:
  accessKey: "ABC123DEF456GHI789"
  secretKey: "xyz789abc123def456ghi789jkl012"
EOF

oc apply -f odf-rgw-secret.yaml
```

### 5. Configure TLS Settings

#### Self-Signed Certificates

If RGW uses self-signed certificates, you have two options:

**Option 1: Skip TLS Verification (Not Recommended for Production)**

Update ConfigMap:
```yaml
skip-tls-verify: "true"
```

**Option 2: Trust the Certificate (Recommended)**

```bash
# Get the CA certificate
oc get secret -n openshift-storage ocs-storagecluster-cephobjectstore-crt \
  -o jsonpath='{.data.tls\.crt}' | base64 -d > rgw-ca.crt

# Create configmap with CA cert
oc create configmap rgw-ca-cert \
  --from-file=ca.crt=rgw-ca.crt \
  -n s3-workload

# Mount in deployment (modify deployment-odf-rgw.yaml)
# Add to volumes:
#   - name: ca-cert
#     configMap:
#       name: rgw-ca-cert
# Add to volumeMounts:
#   - name: ca-cert
#     mountPath: /etc/ssl/certs/rgw-ca.crt
#     subPath: ca.crt
```

### 6. Update Configuration

Edit `deploy/kubernetes/configmap-odf-rgw.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: s3-workload-config
  namespace: s3-workload
data:
  # Use your actual RGW endpoint
  endpoint: "https://s3.openshift-storage.svc.cluster.local"
  region: "us-east-1"
  bucket: "odf-bench-bucket"
  
  # RGW requires path-style addressing
  path-style: "true"
  
  # Set based on your TLS setup
  skip-tls-verify: "false"
  
  # Workload parameters
  concurrency: "64"
  duration: "30m"
  mix: "put=40,get=40,delete=10,copy=5,list=5"
  # ... other settings
```

### 7. Deploy the Workload

```bash
# Apply all manifests
oc apply -f deploy/kubernetes/namespace.yaml
oc apply -f deploy/kubernetes/serviceaccount.yaml
oc apply -f deploy/kubernetes/configmap-odf-rgw.yaml
oc apply -f deploy/kubernetes/deployment-odf-rgw.yaml
oc apply -f deploy/kubernetes/service.yaml

# Optional: ServiceMonitor for Prometheus
oc apply -f deploy/kubernetes/servicemonitor.yaml
```

### 8. Verify Deployment

```bash
# Check pod status
oc get pods -n s3-workload
oc describe pod -l app=s3-workload -n s3-workload

# View logs
oc logs -n s3-workload -l app=s3-workload -f

# Check metrics
oc port-forward -n s3-workload svc/s3-workload-metrics 9090:9090
# Access http://localhost:9090/metrics

# Check health
oc port-forward -n s3-workload svc/s3-workload-metrics 9090:9090
curl http://localhost:9090/healthz
curl http://localhost:9090/readyz
```

## Workload Profiles for ODF RGW

### Balanced Workload

```yaml
mix: "put=40,get=40,delete=10,copy=5,list=5"
size: "dist:lognormal:mean=1MiB,std=0.6"
concurrency: "64"
```

### Write-Heavy Workload

```yaml
mix: "put=70,get=20,delete=5,copy=5"
size: "fixed:4MiB"
concurrency: "128"
```

### Read-Heavy Workload

```yaml
mix: "get=70,head=15,list=10,put=5"
size: "dist:lognormal:mean=512KiB,std=0.4"
concurrency: "256"
```

### Large Object Testing

```yaml
mix: "put=50,get=40,delete=10"
size: "dist:lognormal:mean=100MiB,std=0.5"
concurrency: "32"
keys: "10000"
```

## Monitoring and Metrics

### Expose Metrics via Route

```bash
# Create route for metrics
oc create route edge s3-workload-metrics \
  --service=s3-workload-metrics \
  --port=http \
  -n s3-workload

# Get metrics URL
METRICS_URL=$(oc get route s3-workload-metrics -n s3-workload -o jsonpath='{.spec.host}')
curl -k https://$METRICS_URL/metrics
```

### Integrate with OpenShift Monitoring

```bash
# Apply ServiceMonitor
oc apply -f deploy/kubernetes/servicemonitor.yaml

# View in OpenShift console
# Observe -> Metrics -> Query: s3_ops_total
```

### Key Metrics to Monitor

- `s3_ops_total{op="put",status="success"}` - Successful PUT operations
- `s3_op_latency_seconds{op="get"}` - GET operation latency
- `s3_bytes_written_total` - Total bytes written to RGW
- `s3_bytes_read_total` - Total bytes read from RGW
- `s3_verify_failures_total` - Data verification failures
- `s3_retries_total` - Retry counts (high retries may indicate issues)

## Troubleshooting

### Connection Issues

```bash
# Test RGW connectivity from within the cluster
oc run -it --rm debug --image=curlimages/curl --restart=Never -- \
  curl -v https://s3.openshift-storage.svc.cluster.local

# Check RGW service
oc get svc -n openshift-storage | grep rgw
oc get endpoints -n openshift-storage | grep rgw
```

### Authentication Errors

```bash
# Verify credentials
oc get secret s3-creds -n s3-workload -o yaml

# Test credentials with s3cmd or aws cli
oc run -it --rm aws-cli --image=amazon/aws-cli --restart=Never -- \
  s3 ls --endpoint-url=https://s3.openshift-storage.svc.cluster.local
```

### TLS Certificate Issues

```bash
# Check if certificate error
oc logs -n s3-workload -l app=s3-workload | grep -i tls

# Option 1: Skip verification temporarily
# Update configmap: skip-tls-verify: "true"

# Option 2: Add CA certificate (see step 5)
```

### Performance Issues

```bash
# Check RGW pod resources
oc get pods -n openshift-storage -l app=rook-ceph-rgw -o wide
oc top pods -n openshift-storage | grep rgw

# Check Ceph cluster health
TOOLS_POD=$(oc get pods -n openshift-storage -l app=rook-ceph-tools -o jsonpath='{.items[0].metadata.name}')
oc exec -n openshift-storage $TOOLS_POD -- ceph status
oc exec -n openshift-storage $TOOLS_POD -- ceph osd pool stats

# Scale RGW replicas if needed
oc scale deployment rook-ceph-rgw-ocs-storagecluster-cephobjectstore \
  --replicas=3 -n openshift-storage
```

### Pod Fails to Start

```bash
# Check pod events
oc describe pod -l app=s3-workload -n s3-workload

# Common issues:
# 1. ImagePullBackOff - check image name and registry access
# 2. CrashLoopBackOff - check logs for errors
# 3. Pending - check resource constraints and node availability

# Check logs
oc logs -n s3-workload -l app=s3-workload --previous
```

## Cleanup

### Remove Workload

```bash
# Delete deployment
oc delete -f deploy/kubernetes/deployment-odf-rgw.yaml

# Delete all resources
oc delete project s3-workload
```

### Cleanup Test Data from RGW

```bash
# Use cleanup mode
oc run s3-workload-cleanup --rm -it --restart=Never \
  --image=ghcr.io/paragkamble/s3-workload:latest \
  --env AWS_ACCESS_KEY_ID=your_access_key \
  --env AWS_SECRET_ACCESS_KEY=your_secret_key \
  -n s3-workload \
  -- \
  --endpoint https://s3.openshift-storage.svc.cluster.local \
  --bucket odf-bench-bucket \
  --prefix bench/ \
  --path-style \
  --cleanup
```

### Remove RGW User (Optional)

```bash
# Delete RGW user
TOOLS_POD=$(oc get pods -n openshift-storage -l app=rook-ceph-tools -o jsonpath='{.items[0].metadata.name}')
oc exec -n openshift-storage $TOOLS_POD -- radosgw-admin user rm --uid=s3-benchmark --purge-data
```

## Advanced Configuration

### Using Different Storage Classes

ODF provides different storage classes. To test specific backends:

```bash
# List available storage classes
oc get sc | grep openshift-storage

# Create bucket with specific storage class (if supported by RGW)
# This is typically handled at the RGW/Ceph level, not via S3 API
```

### Multi-Tenant Testing

Create separate namespaces and RGW users for isolated testing:

```bash
# Create multiple users
for i in {1..5}; do
  oc exec -n openshift-storage $TOOLS_POD -- radosgw-admin user create \
    --uid=s3-benchmark-$i \
    --display-name="S3 Benchmark User $i"
done

# Deploy multiple workload instances
for i in {1..5}; do
  oc new-project s3-workload-$i
  # Apply manifests with user-specific credentials
done
```

### Testing with Rate Limiting

```bash
# Update ConfigMap to add rate limiting
oc patch configmap s3-workload-config -n s3-workload --type merge -p '
data:
  rate-type: "fixed"
  rate-limit: "100"
'

# Restart deployment to apply changes
oc rollout restart deployment/s3-workload -n s3-workload
```

## Best Practices

1. **Start Small**: Begin with low concurrency and short duration to verify connectivity
2. **Monitor RGW**: Watch RGW pod resources during tests
3. **Use Path-Style Addressing**: Always set `path-style: "true"` for RGW
4. **Cleanup Data**: Use `--cleanup` mode to remove test data after runs
5. **Separate Buckets**: Use different buckets for different test scenarios
6. **Set Appropriate Quotas**: Configure RGW user quotas to prevent resource exhaustion
7. **Network Policies**: Ensure network policies allow traffic between namespaces if needed

## References

- [OpenShift Data Foundation Documentation](https://access.redhat.com/documentation/en-us/red_hat_openshift_data_foundation/)
- [Ceph RGW Admin Guide](https://docs.ceph.com/en/latest/radosgw/)
- [S3 API Compatibility](https://docs.ceph.com/en/latest/radosgw/s3/)

