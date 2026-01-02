# Deployment

This document assumes Yandex Cloud resources already exist (see `docs/03-yc-setup.md`).

## 1) Build and push images
```
export REGISTRY_ID=<registry-id>
export IMAGE_TAG=latest
./scripts/build-images.sh
```

## 2) Deploy storage container
```
export FOLDER_ID=b1go18ta52hg00o51kkg
export SERVICE_ACCOUNT_ID=<service-account-id>
export REGISTRY_ID=<registry-id>
export IMAGE_TAG=latest
export YDB_DSN="grpcs://ydb.serverless.yandexcloud.net:2135/?database=/ru-central1/b1go18ta52hg00o51kkg/dialog-helper-db"
export INTERNAL_AUTH_TOKEN=<random-token>

./scripts/deploy-storage.sh
```

Get the storage URL from the output:
```
yc serverless container get --name dialog-helper-storage --folder-id b1go18ta52hg00o51kkg --format json
```

## 3) Bootstrap admin key + system prompt
```
export YDB_DSN="grpcs://ydb.serverless.yandexcloud.net:2135/?database=/ru-central1/b1go18ta52hg00o51kkg/dialog-helper-db"

go run ./cmd/bootstrap --system-prompt-file docs/system_prompt.txt
```
Save the printed `ADMIN_MASTER_KEY`.

## 4) Deploy inference container
```
export STORAGE_URL=<storage-container-url>
export YC_FOLDER_ID=b1go18ta52hg00o51kkg
export INTERNAL_AUTH_TOKEN=<same-token-as-storage>

./scripts/deploy-inference.sh
```

Get the inference URL:
```
yc serverless container get --name dialog-helper-inference --folder-id b1go18ta52hg00o51kkg --format json
```

## 5) API Gateway
Update `api/api-gateway.yaml` with `INFERENCE_URL`, `STORAGE_URL`, and `SERVICE_ACCOUNT_ID`, then create the gateway.
