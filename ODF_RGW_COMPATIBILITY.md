# OpenShift ODF/RGW Compatibility - Implementation Summary

## Overview

The s3-workload tool has been enhanced to be fully compatible with **OpenShift Data Foundation (ODF)** and **Ceph Rados Gateway (RGW)** buckets. This document summarizes all changes made to ensure seamless integration.

## ✅ What Was Added

### 1. Documentation

#### New Documentation Files

- **`docs/ODF_RGW_SETUP.md`** - Comprehensive 500+ line guide covering:
  - Step-by-step ODF/RGW setup
  - RGW endpoint discovery (internal and external)
  - Credential management (Noobaa and Ceph RGW methods)
  - TLS certificate configuration
  - Workload profiles for different scenarios
  - Troubleshooting guide
  - Performance tuning recommendations

#### Updated Documentation

- **`README.md`** - Added:
  - ODF/RGW support announcement
  - Quick deployment section for ODF/RGW
  - ODF/RGW CLI example
  - Links to ODF-specific workload profiles

- **`docs/DEPLOYMENT.md`** - Enhanced with:
  - ODF/RGW prerequisites
  - ODF-specific deployment instructions
  - Quick reference commands for RGW operations
  - Path-style addressing requirements

- **`PROJECT_OVERVIEW.md`** - Updated to reflect:
  - ODF/RGW compatibility status
  - New ODF-specific files and profiles

### 2. Kubernetes Manifests

#### New Deployment Files

- **`deploy/kubernetes/configmap-odf-rgw.yaml`**
  - Pre-configured for ODF/RGW endpoints
  - Path-style addressing enabled by default
  - TLS settings configured
  - Optimized workload parameters for RGW

- **`deploy/kubernetes/deployment-odf-rgw.yaml`**
  - Enhanced with path-style and TLS flags
  - Proper environment variable passing
  - ODF/RGW-specific labels

### 3. Workload Profiles

Four new ODF/RGW-optimized profiles in `examples/profiles/`:

- **`odf-rgw-balanced.yaml`**
  - Balanced 40/40/10/5/5 operation mix
  - 64 concurrent workers
  - 1MiB mean object size with log-normal distribution
  - Suitable for general-purpose testing

- **`odf-rgw-write-heavy.yaml`**
  - 70% PUT operations for write performance testing
  - 128 concurrent workers
  - Fixed 4MiB objects
  - Large keyspace (500K objects)

- **`odf-rgw-read-heavy.yaml`**
  - 70% GET, 15% HEAD, 10% LIST operations
  - 256 concurrent workers (very high for read testing)
  - Smaller objects (512KiB mean)
  - Tests read caching behavior

- **`odf-rgw-large-objects.yaml`**
  - Large object testing (100MiB mean)
  - 32 concurrent workers (lower for memory)
  - 5-minute operation timeout
  - Tests backup/archive scenarios

### 4. Automation Script

- **`examples/deploy-odf-rgw.sh`**
  - Automated deployment script with:
  - Automatic credential detection from Noobaa
  - Interactive validation
  - Resource creation and status checking
  - Helpful post-deployment commands
  - Error handling and colored output

## 🔧 Key Configuration Changes

### Path-Style Addressing

RGW requires path-style addressing. This is now:
- Enabled by default in ODF-specific ConfigMaps
- Documented in all RGW-related instructions
- Included as command-line flag in examples

```yaml
path-style: "true"
```

### Endpoint Configuration

Default endpoints configured for ODF:

```yaml
# Internal (recommended for in-cluster workloads)
endpoint: "https://s3.openshift-storage.svc.cluster.local"

# External (via route)
endpoint: "https://s3-rgw-openshift-storage.apps.your-cluster.example.com"
```

### TLS Configuration

Self-signed certificate handling:

```yaml
skip-tls-verify: "false"  # Default: verify certificates
```

Option to skip verification or mount CA certificates documented.

## 📋 Feature Compatibility Matrix

| Feature | AWS S3 | MinIO | Ceph RGW/ODF | Status |
|---------|--------|-------|--------------|--------|
| PUT operations | ✅ | ✅ | ✅ | Full support |
| GET operations | ✅ | ✅ | ✅ | Full support |
| DELETE operations | ✅ | ✅ | ✅ | Full support |
| COPY operations | ✅ | ✅ | ✅ | Full support |
| LIST operations | ✅ | ✅ | ✅ | Full support |
| HEAD operations | ✅ | ✅ | ✅ | Full support |
| Path-style addressing | ✅ | ✅ | ✅ | Required for RGW |
| Virtual-hosted style | ✅ | ✅ | ⚠️ | Use path-style |
| SHA-256 verification | ✅ | ✅ | ✅ | Full support |
| Metadata tags | ✅ | ✅ | ✅ | Full support |
| Bucket versioning | ✅ | ✅ | ✅ | Full support |
| TLS/SSL | ✅ | ✅ | ✅ | Self-signed supported |
| Custom CA certificates | ✅ | ✅ | ✅ | Documented |

## 🚀 Quick Start Commands

### Automated Deployment

```bash
cd examples
./deploy-odf-rgw.sh
```

### Manual Deployment

```bash
oc new-project s3-workload
oc create secret generic s3-creds \
  --from-literal=accessKey=YOUR_ACCESS_KEY \
  --from-literal=secretKey=YOUR_SECRET_KEY

oc apply -f deploy/kubernetes/serviceaccount.yaml
oc apply -f deploy/kubernetes/configmap-odf-rgw.yaml
oc apply -f deploy/kubernetes/deployment-odf-rgw.yaml
oc apply -f deploy/kubernetes/service.yaml
```

### Using Workload Profiles

```bash
s3-workload \
  --endpoint https://s3.openshift-storage.svc.cluster.local \
  --path-style \
  --config examples/profiles/odf-rgw-balanced.yaml
```

## 🔍 RGW-Specific Considerations

### 1. Credential Management

Two methods documented:
- **Noobaa MCG**: Use existing noobaa-admin secret
- **Ceph RGW**: Create dedicated RGW users via radosgw-admin

### 2. Networking

- **Internal Access**: Service endpoint in openshift-storage namespace
- **External Access**: Route creation for outside cluster access
- **Network Policies**: Ensure traffic allowed between namespaces

### 3. Performance

Optimized settings for RGW:
- Appropriate concurrency levels (32-256 depending on workload)
- Timeout values adjusted for RGW response times
- Object sizes optimized for OSD performance

### 4. TLS/SSL

Multiple approaches supported:
- Skip verification (development only)
- Trust self-signed certificates via ConfigMap
- Use existing cluster CA

## 📊 Testing Recommendations

### Initial Testing

Start with low concurrency to verify connectivity:

```bash
--concurrency 8 --duration 1m --mix get=100
```

### Gradual Scale-Up

Increase concurrency progressively:
1. Test with 8 workers
2. Increase to 32 workers
3. Scale to 64 workers
4. Test high concurrency (128-256) for read workloads

### Monitor RGW Resources

```bash
# Watch RGW pods
oc get pods -n openshift-storage -l app=rook-ceph-rgw -w

# Check Ceph health
TOOLS_POD=$(oc get pods -n openshift-storage -l app=rook-ceph-tools -o jsonpath='{.items[0].metadata.name}')
oc exec -n openshift-storage $TOOLS_POD -- ceph status
```

## 🐛 Troubleshooting Quick Reference

### Connection Refused

```bash
# Verify RGW is running
oc get pods -n openshift-storage | grep rgw

# Test connectivity
oc run -it --rm test-curl --image=curlimages/curl --restart=Never -- \
  curl -v https://s3.openshift-storage.svc.cluster.local
```

### Authentication Failed

```bash
# Verify credentials
oc get secret s3-creds -n s3-workload -o jsonpath='{.data.accessKey}' | base64 -d

# Test with AWS CLI
oc run -it --rm aws-cli --image=amazon/aws-cli --restart=Never -- \
  s3 ls --endpoint-url=https://s3.openshift-storage.svc.cluster.local
```

### TLS Errors

```bash
# Check certificate
openssl s_client -connect s3.openshift-storage.svc.cluster.local:443

# Temporarily skip verification
# Update ConfigMap: skip-tls-verify: "true"
```

## 📚 Additional Resources

- **Full Setup Guide**: [docs/ODF_RGW_SETUP.md](docs/ODF_RGW_SETUP.md)
- **CLI Reference**: [docs/CLI.md](docs/CLI.md)
- **Deployment Guide**: [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)

## 🎯 What's Next

The workload generator is now fully compatible with ODF/RGW. Users can:

1. ✅ Deploy directly to OpenShift with ODF
2. ✅ Use pre-configured workload profiles
3. ✅ Test various scenarios (read-heavy, write-heavy, large objects)
4. ✅ Monitor performance via Prometheus metrics
5. ✅ Scale workloads as needed

## 🤝 Contributing

Found an RGW-specific issue? Contributions welcome:
- Report issues specific to ODF/RGW
- Submit workload profiles for specific use cases
- Share performance tuning recommendations

## 📄 License

MIT License - Same as the main project

---

**Last Updated**: November 6, 2025
**Status**: ✅ Production Ready for OpenShift ODF/RGW

