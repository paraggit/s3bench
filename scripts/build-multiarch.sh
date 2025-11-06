#!/bin/bash
# Multi-architecture Docker build script
# Builds for amd64, arm64, and optionally other platforms

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Default platforms
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64}"
REGISTRY="${REGISTRY:-ghcr.io}"
IMAGE_NAME="${IMAGE_NAME:-paragkamble/s3-workload}"
VERSION="${VERSION:-$(git describe --tags --always 2>/dev/null || echo "dev")}"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Multi-Architecture Docker Build${NC}"
echo -e "${BLUE}========================================${NC}"
echo -e "${CYAN}Platforms:${NC} ${PLATFORMS}"
echo -e "${CYAN}Registry:${NC}  ${REGISTRY}/${IMAGE_NAME}"
echo -e "${CYAN}Version:${NC}   ${VERSION}"
echo -e "${BLUE}========================================${NC}"
echo

# Check for docker buildx
if ! docker buildx version &> /dev/null; then
    echo -e "${RED}Error: docker buildx is required for multi-arch builds${NC}"
    echo -e "${YELLOW}Install buildx: https://docs.docker.com/buildx/working-with-buildx/${NC}"
    exit 1
fi

# Create builder if it doesn't exist
BUILDER_NAME="s3-workload-builder"
if ! docker buildx inspect "$BUILDER_NAME" &> /dev/null; then
    echo -e "${YELLOW}Creating buildx builder instance...${NC}"
    docker buildx create --name "$BUILDER_NAME" --driver docker-container --use
    docker buildx inspect --bootstrap
    echo -e "${GREEN}✓ Builder created${NC}"
    echo
else
    echo -e "${GREEN}✓ Using existing builder: $BUILDER_NAME${NC}"
    docker buildx use "$BUILDER_NAME"
    echo
fi

# Build and push
echo -e "${YELLOW}Building multi-architecture images...${NC}"
echo

"${SCRIPT_DIR}/build-image.sh" \
    --registry "$REGISTRY" \
    --name "$IMAGE_NAME" \
    --version "$VERSION" \
    --platform "$PLATFORMS" \
    --push

echo
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Multi-arch build complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo
echo -e "${YELLOW}Inspect the manifest:${NC}"
echo -e "  ${CYAN}docker buildx imagetools inspect ${REGISTRY}/${IMAGE_NAME}:${VERSION}${NC}"
echo

