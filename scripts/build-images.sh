#!/usr/bin/env bash
set -euo pipefail

REGISTRY_ID="${REGISTRY_ID:?REGISTRY_ID is required}"
IMAGE_TAG="${IMAGE_TAG:-latest}"

inference_image="cr.yandex/${REGISTRY_ID}/dialog-helper-inference:${IMAGE_TAG}"
storage_image="cr.yandex/${REGISTRY_ID}/dialog-helper-storage:${IMAGE_TAG}"

docker buildx build --platform linux/amd64 -f Dockerfile.inference -t "${inference_image}" . --push

docker buildx build --platform linux/amd64 -f Dockerfile.storage -t "${storage_image}" . --push

echo "Pushed:"
echo "  ${inference_image}"
echo "  ${storage_image}"
