# Build Scripts

Comprehensive Docker image build scripts for the s3-workload project.

## Scripts Overview

### 1. `build-image.sh` - Main Build Script

The primary script for building Docker images with support for multiple registries and architectures.

**Features:**
- Single or multi-architecture builds
- Multiple registry support (GHCR, Docker Hub, Quay.io, custom)
- Automatic version tagging from git
- Build caching control
- Push to registry option
- Works with both Docker and Podman

**Usage:**

```bash
# Simple local build
./scripts/build-image.sh

# Build and push to GitHub Container Registry
./scripts/build-image.sh --registry ghcr.io --version v1.0.0 --push

# Build for multiple architectures
./scripts/build-image.sh --platform linux/amd64,linux/arm64 --push

# Build for Docker Hub
./scripts/build-image.sh --registry docker.io --name myuser/s3-workload --push

# Build without cache
./scripts/build-image.sh --no-cache

# Build without latest tag
./scripts/build-image.sh --version v1.0.0 --no-latest
```

**Environment Variables:**

```bash
export REGISTRY=ghcr.io
export IMAGE_NAME=paragkamble/s3-workload
export VERSION=v1.0.0
export PLATFORM=linux/amd64
export PUSH=true
export LATEST=true
export CACHE=true

./scripts/build-image.sh
```

### 2. `build-multiarch.sh` - Multi-Architecture Build

Simplified script for building images for multiple architectures (amd64, arm64).

**Usage:**

```bash
# Build for amd64 and arm64
./scripts/build-multiarch.sh

# Custom platforms
PLATFORMS=linux/amd64,linux/arm64,linux/arm/v7 ./scripts/build-multiarch.sh

# Specific version
VERSION=v1.0.0 ./scripts/build-multiarch.sh
```

**Prerequisites:**
- Docker buildx installed and configured
- QEMU for cross-platform builds

### 3. `push-image.sh` - Push to Registry

Push pre-built images to a container registry.

**Usage:**

```bash
# Push to default registry (GHCR)
./scripts/push-image.sh --version v1.0.0

# Push to Docker Hub
docker login docker.io
./scripts/push-image.sh --registry docker.io --name myuser/s3-workload

# Push to Quay.io
docker login quay.io
./scripts/push-image.sh --registry quay.io --name myuser/s3-workload
```

## Registry-Specific Instructions

### GitHub Container Registry (GHCR)

```bash
# Authenticate
export GITHUB_TOKEN=ghp_your_token_here
echo $GITHUB_TOKEN | docker login ghcr.io -u your-username --password-stdin

# Build and push
./scripts/build-image.sh \
  --registry ghcr.io \
  --name your-username/s3-workload \
  --version v1.0.0 \
  --push
```

### Docker Hub

```bash
# Authenticate
docker login docker.io

# Build and push
./scripts/build-image.sh \
  --registry docker.io \
  --name your-username/s3-workload \
  --version v1.0.0 \
  --push
```

### Quay.io

```bash
# Authenticate
docker login quay.io

# Build and push
./scripts/build-image.sh \
  --registry quay.io \
  --name your-username/s3-workload \
  --version v1.0.0 \
  --push
```

### OpenShift Internal Registry

```bash
# Login to OpenShift
oc login

# Get registry token
TOKEN=$(oc whoami -t)

# Login to registry
docker login -u $(oc whoami) -p $TOKEN \
  image-registry.openshift-image-registry.svc:5000

# Build and push
./scripts/build-image.sh \
  --registry image-registry.openshift-image-registry.svc:5000 \
  --name myproject/s3-workload \
  --version latest \
  --push
```

## Multi-Architecture Builds

### Setup Docker Buildx

```bash
# Create builder instance
docker buildx create --name mybuilder --driver docker-container --use
docker buildx inspect --bootstrap

# List available platforms
docker buildx ls
```

### Build for Multiple Architectures

```bash
# Build for amd64 and arm64
./scripts/build-multiarch.sh

# Or use build-image.sh directly
./scripts/build-image.sh \
  --platform linux/amd64,linux/arm64 \
  --version v1.0.0 \
  --push
```

### Inspect Multi-Arch Manifest

```bash
docker buildx imagetools inspect ghcr.io/paragkamble/s3-workload:v1.0.0
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Build and Push Docker Image

on:
  push:
    tags:
      - 'v*'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v2
      
      - name: Login to GHCR
        uses: docker/login-action@v2
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      
      - name: Build and push
        run: |
          ./scripts/build-image.sh \
            --registry ghcr.io \
            --name ${{ github.repository }} \
            --version ${GITHUB_REF#refs/tags/} \
            --platform linux/amd64,linux/arm64 \
            --push
```

### GitLab CI

```yaml
build-image:
  image: docker:latest
  services:
    - docker:dind
  script:
    - docker login -u $CI_REGISTRY_USER -p $CI_REGISTRY_PASSWORD $CI_REGISTRY
    - ./scripts/build-image.sh --registry $CI_REGISTRY --push
  only:
    - tags
```

## Version Tagging

The scripts automatically create multiple tags based on the version:

**For version `v1.2.3`:**
- `v1.2.3` (specific version)
- `v1.2` (minor version)
- `v1` (major version)
- `latest` (unless `--no-latest` is specified)

**For git commit:**
- `dev` (if no version tag)
- `abc1234` (git commit short hash)

## Testing Built Images

### Test Locally

```bash
# Run help
docker run --rm ghcr.io/paragkamble/s3-workload:v1.0.0 --help

# Run with environment variables
docker run --rm \
  -e AWS_ACCESS_KEY_ID=xxx \
  -e AWS_SECRET_ACCESS_KEY=xxx \
  ghcr.io/paragkamble/s3-workload:v1.0.0 \
  --endpoint https://s3.amazonaws.com \
  --bucket test-bucket \
  --duration 1m \
  --dry-run
```

### Test in Kubernetes

```bash
# Update deployment image
kubectl set image deployment/s3-workload \
  s3-workload=ghcr.io/paragkamble/s3-workload:v1.0.0

# Or edit deployment directly
kubectl edit deployment/s3-workload
```

### Test in OpenShift

```bash
# Update deployment image
oc set image deployment/s3-workload \
  s3-workload=ghcr.io/paragkamble/s3-workload:v1.0.0
```

## Troubleshooting

### Build Fails with "no space left on device"

```bash
# Clean up Docker
docker system prune -a --volumes

# Or build without cache
./scripts/build-image.sh --no-cache
```

### Authentication Failed

```bash
# Check if logged in
docker info | grep Username

# Re-login
docker login ghcr.io
```

### Multi-Arch Build Not Working

```bash
# Install QEMU
docker run --privileged --rm tonistiigi/binfmt --install all

# Verify platforms
docker buildx ls
```

### Buildx Builder Issues

```bash
# Remove and recreate builder
docker buildx rm s3-workload-builder
docker buildx create --name s3-workload-builder --driver docker-container --use
docker buildx inspect --bootstrap
```

## Best Practices

1. **Always tag with version numbers** - Avoid using only `latest`
2. **Use semantic versioning** - Follow `vX.Y.Z` format
3. **Build multi-arch for production** - Support both amd64 and arm64
4. **Test images before pushing** - Run smoke tests locally
5. **Use CI/CD for releases** - Automate builds on git tags
6. **Keep images small** - The Dockerfile uses multi-stage builds and distroless base
7. **Sign images** - Consider using cosign for image signing

## Script Maintenance

To modify default values, edit the scripts or set environment variables:

```bash
# Custom defaults
export REGISTRY=my-registry.com
export IMAGE_NAME=myteam/s3-workload
export VERSION=$(git describe --tags --always)

# Use in scripts
./scripts/build-image.sh --push
```

## Additional Resources

- [Docker Buildx Documentation](https://docs.docker.com/buildx/working-with-buildx/)
- [GitHub Container Registry](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [Docker Hub](https://docs.docker.com/docker-hub/)
- [Quay.io](https://docs.quay.io/)
- [Multi-Platform Images](https://docs.docker.com/build/building/multi-platform/)

