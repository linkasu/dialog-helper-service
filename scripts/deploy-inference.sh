#!/usr/bin/env bash
set -euo pipefail

FOLDER_ID="${FOLDER_ID:?FOLDER_ID is required}"
SERVICE_ACCOUNT_ID="${SERVICE_ACCOUNT_ID:?SERVICE_ACCOUNT_ID is required}"
REGISTRY_ID="${REGISTRY_ID:?REGISTRY_ID is required}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
CONTAINER_NAME="${CONTAINER_NAME:-dialog-helper-inference}"
STORAGE_URL="${STORAGE_URL:?STORAGE_URL is required}"
INTERNAL_AUTH_TOKEN="${INTERNAL_AUTH_TOKEN:?INTERNAL_AUTH_TOKEN is required}"
YC_FOLDER_ID="${YC_FOLDER_ID:?YC_FOLDER_ID is required}"

YC_GPT_MODEL_URI="${YC_GPT_MODEL_URI:-}"

image="cr.yandex/${REGISTRY_ID}/dialog-helper-inference:${IMAGE_TAG}"

yc serverless container create --name "${CONTAINER_NAME}" --folder-id "${FOLDER_ID}" || true

yc serverless container revision deploy \
  --container-name "${CONTAINER_NAME}" \
  --folder-id "${FOLDER_ID}" \
  --image "${image}" \
  --cores 1 \
  --memory 512MB \
  --core-fraction 100 \
  --execution-timeout 60s \
  --concurrency 4 \
  --service-account-id "${SERVICE_ACCOUNT_ID}" \
  --environment STORAGE_URL="${STORAGE_URL}",INTERNAL_AUTH_TOKEN="${INTERNAL_AUTH_TOKEN}",YC_FOLDER_ID="${YC_FOLDER_ID}",YC_GPT_MODEL_URI="${YC_GPT_MODEL_URI}"

echo "Inference deployed. Get URL:"

yc serverless container get --name "${CONTAINER_NAME}" --folder-id "${FOLDER_ID}" --format json
