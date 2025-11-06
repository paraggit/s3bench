#!/bin/bash
# Cleanup script for s3-workload
# Removes only objects created by s3-workload tool

set -e

ENDPOINT="${S3_ENDPOINT:-https://s3.amazonaws.com}"
REGION="${AWS_REGION:-us-east-1}"
BUCKET="${S3_BUCKET:-bench-bucket}"
PREFIX="${PREFIX:-bench/}"

echo "Cleaning up s3-workload objects..."
echo "  Endpoint: $ENDPOINT"
echo "  Bucket: $BUCKET"
echo "  Prefix: $PREFIX"
echo ""
read -p "Are you sure? (yes/no): " confirm

if [ "$confirm" != "yes" ]; then
    echo "Aborted."
    exit 0
fi

s3-workload \
  --endpoint "$ENDPOINT" \
  --region "$REGION" \
  --bucket "$BUCKET" \
  --prefix "$PREFIX" \
  --cleanup

echo "Cleanup completed."

