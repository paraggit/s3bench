# Docker Build Scripts - Quick Reference

## ✅ What Was Added

Comprehensive Docker image build scripts with support for multiple registries and architectures.

### Files Created

```
scripts/
├── build-image.sh          # Main build script (8.8KB)
├── build-multiarch.sh      # Multi-arch build helper (2.2KB)
├── push-image.sh           # Push to registry (2.8KB)
└── README.md               # Detailed documentation (7.8KB)

.github/workflows/
└── build-image.yml.example # GitHub Actions workflow template
```

## 🚀 Quick Start

### Simple Local Build

```bash
./scripts/build-image.sh
```

### Build and Push to GitHub Container Registry

```bash
# Login to GHCR
export GITHUB_TOKEN=ghp_your_token
echo $GITHUB_TOKEN | docker login ghcr.io -u your-username --password-stdin

# Build and push
./scripts/build-image.sh --version v1.0.0 --push
```

### Build Multi-Architecture Image

```bash
./scripts/build-multiarch.sh
```

### Using Make Targets

```bash
# Build locally
make docker-build

# Build and push
make docker-build-push VERSION=v1.0.0

# Build multi-arch
make docker-multiarch
```

## 🎯 Key Features

### 1. **Multi-Registry Support**

- **GitHub Container Registry (GHCR)** - `ghcr.io`
- **Docker Hub** - `docker.io`
- **Quay.io** - `quay.io`
- **OpenShift Internal Registry** - `image-registry.openshift-image-registry.svc:5000`
- **Custom registries**

### 2. **Multi-Architecture Builds**

Build for multiple platforms simultaneously:
- `linux/amd64` (Intel/AMD 64-bit)
- `linux/arm64` (ARM 64-bit)
- `linux/arm/v7` (ARM 32-bit)

### 3. **Automatic Version Tagging**

For version `v1.2.3`, creates tags:
- `v1.2.3` (specific version)
- `v1.2` (minor version)
- `v1` (major version)
- `latest` (configurable)

### 4. **Smart Build Options**

- Build caching control (`--no-cache`)
- Latest tag control (`--no-latest`)
- Docker or Podman support
- Push to registry after build
- Automatic git commit and build date injection

## 📋 Usage Examples

### GitHub Container Registry

```bash
# Authenticate
export GITHUB_TOKEN=ghp_your_token
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
# Login
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
# Login
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

# Get token and login to registry
TOKEN=$(oc whoami -t)
docker login -u $(oc whoami) -p $TOKEN \
  image-registry.openshift-image-registry.svc:5000

# Build and push
./scripts/build-image.sh \
  --registry image-registry.openshift-image-registry.svc:5000 \
  --name myproject/s3-workload \
  --version latest \
  --push
```

### Multi-Architecture Build

```bash
# Setup buildx (first time only)
docker buildx create --name mybuilder --driver docker-container --use
docker buildx inspect --bootstrap

# Build for amd64 and arm64
./scripts/build-image.sh \
  --platform linux/amd64,linux/arm64 \
  --version v1.0.0 \
  --push

# Or use the helper script
./scripts/build-multiarch.sh
```

## 🔧 Advanced Options

### Environment Variables

```bash
export REGISTRY=ghcr.io
export IMAGE_NAME=myuser/s3-workload
export VERSION=v1.0.0
export PLATFORM=linux/amd64,linux/arm64
export PUSH=true
export LATEST=true
export CACHE=true

./scripts/build-image.sh
```

### Custom Build

```bash
./scripts/build-image.sh \
  --registry my-registry.com \
  --name myteam/s3-workload \
  --version v2.0.0-beta \
  --platform linux/amd64,linux/arm64 \
  --no-cache \
  --push
```

### Build Without Latest Tag

```bash
./scripts/build-image.sh \
  --version v1.0.0-rc1 \
  --no-latest
```

## 🤖 CI/CD Integration

### GitHub Actions

A complete example workflow is provided in `.github/workflows/build-image.yml.example`.

To use it:

```bash
# Create workflows directory
mkdir -p .github/workflows

# Copy example
cp .github/workflows/build-image.yml.example .github/workflows/build-image.yml

# Edit and customize as needed
# Commit and push
```

The workflow:
- ✅ Builds on push to main/master
- ✅ Builds on tags (v*)
- ✅ Tests PRs without pushing
- ✅ Multi-architecture builds (amd64, arm64)
- ✅ Vulnerability scanning with Trivy
- ✅ SBOM generation
- ✅ Image testing

### GitLab CI

```yaml
build-image:
  image: docker:latest
  services:
    - docker:dind
  script:
    - docker login -u $CI_REGISTRY_USER -p $CI_REGISTRY_PASSWORD $CI_REGISTRY
    - ./scripts/build-image.sh --registry $CI_REGISTRY --version $CI_COMMIT_TAG --push
  only:
    - tags
```

### Jenkins

```groovy
pipeline {
    agent any
    stages {
        stage('Build') {
            steps {
                sh './scripts/build-image.sh --version ${GIT_TAG} --push'
            }
        }
    }
}
```

## 🧪 Testing Built Images

### Local Test

```bash
# Run help
docker run --rm ghcr.io/paragkamble/s3-workload:v1.0.0 --help

# Run version
docker run --rm ghcr.io/paragkamble/s3-workload:v1.0.0 version

# Dry run test
docker run --rm \
  -e AWS_ACCESS_KEY_ID=test \
  -e AWS_SECRET_ACCESS_KEY=test \
  ghcr.io/paragkamble/s3-workload:v1.0.0 \
  --endpoint https://s3.amazonaws.com \
  --bucket test-bucket \
  --duration 1m \
  --dry-run
```

### Inspect Multi-Arch Manifest

```bash
docker buildx imagetools inspect ghcr.io/paragkamble/s3-workload:v1.0.0
```

### Verify Platforms

```bash
docker manifest inspect ghcr.io/paragkamble/s3-workload:v1.0.0 | grep -A 2 platform
```

## 📊 Script Comparison

| Script | Purpose | Use Case |
|--------|---------|----------|
| `build-image.sh` | Main build script | All build scenarios |
| `build-multiarch.sh` | Multi-arch builds | Production releases |
| `push-image.sh` | Push existing images | Separate build/push |
| `make docker` | Simple local build | Quick testing |
| `make docker-build` | Build with scripts | Advanced builds |
| `make docker-multiarch` | Multi-arch via make | Makefile workflows |

## 🛠️ Troubleshooting

### Docker Not Found

```bash
# Install Docker or use Podman
# Scripts support both Docker and Podman
```

### Buildx Not Available

```bash
# Install buildx plugin
docker buildx version

# Or update Docker Desktop
```

### Multi-Arch Build Fails

```bash
# Install QEMU for cross-platform builds
docker run --privileged --rm tonistiigi/binfmt --install all

# Verify platforms
docker buildx ls
```

### Authentication Failed

```bash
# Check login status
docker info | grep Username

# Re-login
docker login ghcr.io
```

### No Space Left on Device

```bash
# Clean up Docker
docker system prune -a --volumes

# Or build without cache
./scripts/build-image.sh --no-cache
```

## 📚 Documentation

- **Full build documentation**: [scripts/README.md](scripts/README.md)
- **GitHub Actions example**: [.github/workflows/build-image.yml.example](.github/workflows/build-image.yml.example)
- **Dockerfile**: [Dockerfile](Dockerfile)
- **Makefile targets**: Run `make help`

## 🎉 Summary

The build scripts provide:

✅ **Multi-registry support** - GHCR, Docker Hub, Quay.io, custom  
✅ **Multi-architecture builds** - amd64, arm64, arm/v7  
✅ **Automatic versioning** - From git tags  
✅ **Smart tagging** - Major, minor, patch, latest  
✅ **CI/CD ready** - GitHub Actions template included  
✅ **Flexible configuration** - CLI args and env vars  
✅ **Docker & Podman** - Works with both  
✅ **Comprehensive docs** - Scripts and workflows documented  

## 🔗 Quick Links

- Main build script: [scripts/build-image.sh](scripts/build-image.sh)
- Build docs: [scripts/README.md](scripts/README.md)
- GitHub Actions: [.github/workflows/build-image.yml.example](.github/workflows/build-image.yml.example)
- Dockerfile: [Dockerfile](Dockerfile)
- Project README: [README.md](README.md)

---

**Ready to build?** Start with:

```bash
./scripts/build-image.sh --help
```

