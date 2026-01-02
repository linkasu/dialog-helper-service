#!/usr/bin/env bash
set -euo pipefail

FOLDER_ID="${FOLDER_ID:?FOLDER_ID is required}"
SERVICE_ACCOUNT_ID="${SERVICE_ACCOUNT_ID:?SERVICE_ACCOUNT_ID is required}"
REGISTRY_ID="${REGISTRY_ID:?REGISTRY_ID is required}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
CONTAINER_NAME="${CONTAINER_NAME:-dialog-helper-storage}"
YDB_DSN="${YDB_DSN:?YDB_DSN is required}"
INTERNAL_AUTH_TOKEN="${INTERNAL_AUTH_TOKEN:?INTERNAL_AUTH_TOKEN is required}"

TELEGRAM_BOT_TOKEN="${TELEGRAM_BOT_TOKEN:-}"
TELEGRAM_ADMIN_ID="${TELEGRAM_ADMIN_ID:-}"
ADMIN_OTP_TTL="${ADMIN_OTP_TTL:-}"
ADMIN_SESSION_TTL="${ADMIN_SESSION_TTL:-}"

YC_LOG_GROUP_ID="${YC_LOG_GROUP_ID:-}"
YC_LOG_RESOURCE_TYPES="${YC_LOG_RESOURCE_TYPES:-}"
YC_LOG_RESOURCE_IDS="${YC_LOG_RESOURCE_IDS:-}"
YC_LOG_STREAMS="${YC_LOG_STREAMS:-}"
YC_LOG_LEVELS="${YC_LOG_LEVELS:-}"
YC_LOG_FILTER="${YC_LOG_FILTER:-}"
YC_LOG_ENDPOINT="${YC_LOG_ENDPOINT:-}"
YC_LOG_MAX_BYTES="${YC_LOG_MAX_BYTES:-}"

image="cr.yandex/${REGISTRY_ID}/dialog-helper-storage:${IMAGE_TAG}"

yc serverless container create --name "${CONTAINER_NAME}" --folder-id "${FOLDER_ID}" || true

envs="YDB_DSN=${YDB_DSN},INTERNAL_AUTH_TOKEN=${INTERNAL_AUTH_TOKEN}"

if [[ -n "${TELEGRAM_BOT_TOKEN}" ]]; then
  envs+=",TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}"
fi
if [[ -n "${TELEGRAM_ADMIN_ID}" ]]; then
  envs+=",TELEGRAM_ADMIN_ID=${TELEGRAM_ADMIN_ID}"
fi
if [[ -n "${ADMIN_OTP_TTL}" ]]; then
  envs+=",ADMIN_OTP_TTL=${ADMIN_OTP_TTL}"
fi
if [[ -n "${ADMIN_SESSION_TTL}" ]]; then
  envs+=",ADMIN_SESSION_TTL=${ADMIN_SESSION_TTL}"
fi

if [[ -n "${YC_LOG_GROUP_ID}" ]]; then
  envs+=",YC_LOG_GROUP_ID=${YC_LOG_GROUP_ID}"
fi
if [[ -n "${YC_LOG_RESOURCE_TYPES}" ]]; then
  envs+=",YC_LOG_RESOURCE_TYPES=${YC_LOG_RESOURCE_TYPES}"
fi
if [[ -n "${YC_LOG_RESOURCE_IDS}" ]]; then
  envs+=",YC_LOG_RESOURCE_IDS=${YC_LOG_RESOURCE_IDS}"
fi
if [[ -n "${YC_LOG_STREAMS}" ]]; then
  envs+=",YC_LOG_STREAMS=${YC_LOG_STREAMS}"
fi
if [[ -n "${YC_LOG_LEVELS}" ]]; then
  envs+=",YC_LOG_LEVELS=${YC_LOG_LEVELS}"
fi
if [[ -n "${YC_LOG_FILTER}" ]]; then
  envs+=",YC_LOG_FILTER=${YC_LOG_FILTER}"
fi
if [[ -n "${YC_LOG_ENDPOINT}" ]]; then
  envs+=",YC_LOG_ENDPOINT=${YC_LOG_ENDPOINT}"
fi
if [[ -n "${YC_LOG_MAX_BYTES}" ]]; then
  envs+=",YC_LOG_MAX_BYTES=${YC_LOG_MAX_BYTES}"
fi

yc serverless container revision deploy \
  --container-name "${CONTAINER_NAME}" \
  --folder-id "${FOLDER_ID}" \
  --image "${image}" \
  --cores 1 \
  --memory 256MB \
  --core-fraction 100 \
  --execution-timeout 60s \
  --concurrency 4 \
  --service-account-id "${SERVICE_ACCOUNT_ID}" \
  --environment "${envs}"

echo "Storage deployed. Get URL:"

yc serverless container get --name "${CONTAINER_NAME}" --folder-id "${FOLDER_ID}" --format json
