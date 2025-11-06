#!/bin/bash
# Docker Image Build Script for s3-workload
# Supports multiple registries, architectures, and versioning

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Default values
REGISTRY="${REGISTRY:-ghcr.io}"
IMAGE_NAME="${IMAGE_NAME:-paragkamble/s3-workload}"
VERSION="${VERSION:-$(git describe --tags --always 2>/dev/null || echo "dev")}"
GIT_COMMIT="${GIT_COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")}"
BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
PLATFORM="${PLATFORM:-linux/amd64}"
PUSH="${PUSH:-false}"
LATEST="${LATEST:-true}"
CACHE="${CACHE:-true}"

# Script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Usage function
usage() {
    cat << EOF
${CYAN}Docker Image Build Script for s3-workload${NC}

${YELLOW}Usage:${NC}
  $0 [OPTIONS]

${YELLOW}Options:${NC}
  -r, --registry REGISTRY    Container registry (default: ghcr.io)
                             Options: ghcr.io, docker.io, quay.io, or custom
  -n, --name NAME            Image name (default: paragkamble/s3-workload)
  -v, --version VERSION      Image version tag (default: git tag or 'dev')
  -p, --platform PLATFORM    Target platform (default: linux/amd64)
                             For multi-arch: linux/amd64,linux/arm64
  --push                     Push image to registry after build
  --no-latest                Don't tag as 'latest'
  --no-cache                 Build without using cache
  -h, --help                 Display this help message

${YELLOW}Environment Variables:${NC}
  REGISTRY                   Container registry
  IMAGE_NAME                 Image name
  VERSION                    Version tag
  PLATFORM                   Target platform(s)
  PUSH                       Push to registry (true/false)
  LATEST                     Tag as latest (true/false)
  CACHE                      Use build cache (true/false)

${YELLOW}Examples:${NC}
  # Build locally
  $0

  # Build and push to GHCR
  $0 --registry ghcr.io --version v1.0.0 --push

  # Build multi-arch and push
  $0 --platform linux/amd64,linux/arm64 --push

  # Build for Docker Hub
  $0 --registry docker.io --name myuser/s3-workload --push

  # Build for OpenShift internal registry
  $0 --registry image-registry.openshift-image-registry.svc:5000 \\
     --name s3-workload/s3-workload --version latest

${YELLOW}Registry-Specific Examples:${NC}
  # GitHub Container Registry (GHCR)
  export GITHUB_TOKEN=ghp_xxx
  echo \$GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin
  $0 --registry ghcr.io --push

  # Docker Hub
  docker login docker.io
  $0 --registry docker.io --name username/s3-workload --push

  # Quay.io
  docker login quay.io
  $0 --registry quay.io --name username/s3-workload --push

EOF
    exit 0
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -r|--registry)
            REGISTRY="$2"
            shift 2
            ;;
        -n|--name)
            IMAGE_NAME="$2"
            shift 2
            ;;
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -p|--platform)
            PLATFORM="$2"
            shift 2
            ;;
        --push)
            PUSH="true"
            shift
            ;;
        --no-latest)
            LATEST="false"
            shift
            ;;
        --no-cache)
            CACHE="false"
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo -e "${RED}Error: Unknown option $1${NC}"
            usage
            ;;
    esac
done

# Construct full image name
FULL_IMAGE="${REGISTRY}/${IMAGE_NAME}"

# Print configuration
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Docker Image Build Configuration${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "${CYAN}Registry:${NC}      ${FULL_IMAGE}"
echo -e "${CYAN}Version:${NC}       ${VERSION}"
echo -e "${CYAN}Git Commit:${NC}    ${GIT_COMMIT}"
echo -e "${CYAN}Build Date:${NC}    ${BUILD_DATE}"
echo -e "${CYAN}Platform:${NC}      ${PLATFORM}"
echo -e "${CYAN}Push:${NC}          ${PUSH}"
echo -e "${CYAN}Tag Latest:${NC}    ${LATEST}"
echo -e "${CYAN}Use Cache:${NC}     ${CACHE}"
echo -e "${BLUE}========================================${NC}"
echo

# Check if Docker/Podman is available
if command -v docker &> /dev/null; then
    CONTAINER_TOOL="docker"
    echo -e "${GREEN}✓ Using Docker${NC}"
elif command -v podman &> /dev/null; then
    CONTAINER_TOOL="podman"
    echo -e "${GREEN}✓ Using Podman${NC}"
else
    echo -e "${RED}Error: Neither docker nor podman found. Please install one of them.${NC}"
    exit 1
fi

# Check if buildx is available for multi-arch builds
MULTI_ARCH=false
if [[ "$PLATFORM" == *","* ]]; then
    MULTI_ARCH=true
    if [[ "$CONTAINER_TOOL" == "docker" ]]; then
        if ! docker buildx version &> /dev/null; then
            echo -e "${RED}Error: docker buildx is required for multi-arch builds${NC}"
            echo -e "${YELLOW}Install buildx: https://docs.docker.com/buildx/working-with-buildx/${NC}"
            exit 1
        fi
        echo -e "${GREEN}✓ Multi-arch build enabled with buildx${NC}"
    else
        echo -e "${YELLOW}⚠ Multi-arch build with podman - ensure qemu-user-static is installed${NC}"
    fi
fi

# Change to project root
cd "$PROJECT_ROOT"

# Build arguments
BUILD_ARGS=(
    --build-arg "VERSION=${VERSION}"
    --build-arg "GIT_COMMIT=${GIT_COMMIT}"
    --build-arg "BUILD_DATE=${BUILD_DATE}"
)

# Cache arguments
if [[ "$CACHE" == "false" ]]; then
    BUILD_ARGS+=(--no-cache)
fi

# Tags
TAGS=(
    -t "${FULL_IMAGE}:${VERSION}"
)

if [[ "$LATEST" == "true" ]]; then
    TAGS+=(-t "${FULL_IMAGE}:latest")
fi

# Additional tags based on version
if [[ "$VERSION" =~ ^v?([0-9]+)\.([0-9]+)\.([0-9]+) ]]; then
    MAJOR="${BASH_REMATCH[1]}"
    MINOR="${BASH_REMATCH[2]}"
    TAGS+=(-t "${FULL_IMAGE}:v${MAJOR}")
    TAGS+=(-t "${FULL_IMAGE}:v${MAJOR}.${MINOR}")
fi

echo -e "${YELLOW}Building image...${NC}"
echo

# Build command
if [[ "$MULTI_ARCH" == "true" ]] && [[ "$CONTAINER_TOOL" == "docker" ]]; then
    # Multi-arch build with buildx
    BUILDX_ARGS=(
        buildx build
        --platform "$PLATFORM"
        "${BUILD_ARGS[@]}"
        "${TAGS[@]}"
    )
    
    if [[ "$PUSH" == "true" ]]; then
        BUILDX_ARGS+=(--push)
    else
        BUILDX_ARGS+=(--load)
    fi
    
    BUILDX_ARGS+=(.)
    
    echo -e "${CYAN}Command:${NC} docker ${BUILDX_ARGS[*]}"
    echo
    
    docker "${BUILDX_ARGS[@]}"
else
    # Single platform build
    BUILD_CMD=(
        "$CONTAINER_TOOL" build
        --platform "$PLATFORM"
        "${BUILD_ARGS[@]}"
        "${TAGS[@]}"
        .
    )
    
    echo -e "${CYAN}Command:${NC} ${BUILD_CMD[*]}"
    echo
    
    "${BUILD_CMD[@]}"
fi

echo
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}✓ Build completed successfully!${NC}"
echo -e "${GREEN}========================================${NC}"
echo

# Show built images
echo -e "${YELLOW}Built images:${NC}"
for tag in "${TAGS[@]}"; do
    image_name="${tag#-t }"
    echo -e "  ${CYAN}${image_name}${NC}"
done
echo

# Push if requested
if [[ "$PUSH" == "true" ]] && [[ "$MULTI_ARCH" == "false" ]]; then
    echo -e "${YELLOW}Pushing images to registry...${NC}"
    echo
    
    for tag in "${TAGS[@]}"; do
        image_name="${tag#-t }"
        echo -e "${CYAN}Pushing: ${image_name}${NC}"
        $CONTAINER_TOOL push "$image_name"
    done
    
    echo
    echo -e "${GREEN}✓ Images pushed successfully!${NC}"
    echo
elif [[ "$PUSH" == "true" ]] && [[ "$MULTI_ARCH" == "true" ]]; then
    echo -e "${GREEN}✓ Multi-arch images pushed via buildx!${NC}"
    echo
fi

# Print usage instructions
echo -e "${YELLOW}Usage instructions:${NC}"
echo
echo -e "Test the image locally:"
echo -e "  ${CYAN}$CONTAINER_TOOL run --rm ${FULL_IMAGE}:${VERSION} --help${NC}"
echo
echo -e "Run with environment variables:"
echo -e "  ${CYAN}$CONTAINER_TOOL run --rm \\${NC}"
echo -e "    ${CYAN}-e AWS_ACCESS_KEY_ID=xxx \\${NC}"
echo -e "    ${CYAN}-e AWS_SECRET_ACCESS_KEY=xxx \\${NC}"
echo -e "    ${CYAN}${FULL_IMAGE}:${VERSION} \\${NC}"
echo -e "    ${CYAN}--endpoint https://s3.amazonaws.com \\${NC}"
echo -e "    ${CYAN}--bucket test-bucket --duration 1m${NC}"
echo
echo -e "Deploy to Kubernetes:"
echo -e "  ${CYAN}kubectl set image deployment/s3-workload \\${NC}"
echo -e "    ${CYAN}s3-workload=${FULL_IMAGE}:${VERSION}${NC}"
echo
echo -e "Deploy to OpenShift:"
echo -e "  ${CYAN}oc set image deployment/s3-workload \\${NC}"
echo -e "    ${CYAN}s3-workload=${FULL_IMAGE}:${VERSION}${NC}"
echo

if [[ "$PUSH" == "false" ]]; then
    echo -e "${YELLOW}Note: Image was not pushed to registry.${NC}"
    echo -e "To push, run with ${CYAN}--push${NC} flag:"
    echo -e "  ${CYAN}$0 --version ${VERSION} --push${NC}"
    echo
fi

echo -e "${GREEN}Done!${NC}"

