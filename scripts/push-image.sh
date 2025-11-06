#!/bin/bash
# Push Docker images to registry
# Supports multiple registries with authentication

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Default values
REGISTRY="${REGISTRY:-ghcr.io}"
IMAGE_NAME="${IMAGE_NAME:-paragkamble/s3-workload}"
VERSION="${VERSION:-latest}"

usage() {
    cat << EOF
${CYAN}Push Docker Images to Registry${NC}

${YELLOW}Usage:${NC}
  $0 [OPTIONS]

${YELLOW}Options:${NC}
  -r, --registry REGISTRY    Container registry (default: ghcr.io)
  -n, --name NAME            Image name (default: paragkamble/s3-workload)
  -v, --version VERSION      Version to push (default: latest)
  -h, --help                 Display this help message

${YELLOW}Examples:${NC}
  # Push to GitHub Container Registry
  export GITHUB_TOKEN=ghp_xxx
  echo \$GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin
  $0 --registry ghcr.io --version v1.0.0

  # Push to Docker Hub
  docker login docker.io
  $0 --registry docker.io --name username/s3-workload

  # Push to Quay.io
  docker login quay.io
  $0 --registry quay.io --name username/s3-workload

EOF
    exit 0
}

# Parse arguments
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
        -h|--help)
            usage
            ;;
        *)
            echo -e "${RED}Error: Unknown option $1${NC}"
            usage
            ;;
    esac
done

FULL_IMAGE="${REGISTRY}/${IMAGE_NAME}"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Push Docker Image${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "${CYAN}Image:${NC}    ${FULL_IMAGE}:${VERSION}"
echo -e "${BLUE}========================================${NC}"
echo

# Check if docker is available
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: docker not found${NC}"
    exit 1
fi

# Check if image exists locally
if ! docker image inspect "${FULL_IMAGE}:${VERSION}" &> /dev/null; then
    echo -e "${RED}Error: Image ${FULL_IMAGE}:${VERSION} not found locally${NC}"
    echo -e "${YELLOW}Build it first with: scripts/build-image.sh${NC}"
    exit 1
fi

# Push image
echo -e "${YELLOW}Pushing image...${NC}"
docker push "${FULL_IMAGE}:${VERSION}"

# Push latest tag if it exists
if docker image inspect "${FULL_IMAGE}:latest" &> /dev/null && [[ "$VERSION" != "latest" ]]; then
    echo -e "${YELLOW}Pushing latest tag...${NC}"
    docker push "${FULL_IMAGE}:latest"
fi

echo
echo -e "${GREEN}✓ Images pushed successfully!${NC}"
echo
echo -e "${YELLOW}Verify the push:${NC}"
echo -e "  ${CYAN}docker pull ${FULL_IMAGE}:${VERSION}${NC}"
echo

