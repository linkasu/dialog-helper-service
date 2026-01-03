# Deployment

This document assumes Yandex Cloud resources already exist (see `docs/03-yc-setup.md`).

## 1) Build and push images
```
export REGISTRY_ID=<registry-id>
export IMAGE_TAG=latest
./scripts/build-images.sh
```

## 2) Apply YDB schema
```
export YDB_DSN="grpcs://ydb.serverless.yandexcloud.net:2135/?database=/ru-central1/.../dialog-helper-db"

go run ./cmd/migrate
```

## 3) Deploy monolith container (storage + inference)
```
export FOLDER_ID=b1go18ta52hg00o51kkg
export SERVICE_ACCOUNT_ID=<service-account-id>
export REGISTRY_ID=<registry-id>
export IMAGE_TAG=latest
export YDB_DSN="grpcs://ydb.serverless.yandexcloud.net:2135/?database=/ru-central1/.../dialog-helper-db"
export INTERNAL_AUTH_TOKEN=<random-token>
export YC_FOLDER_ID=b1go18ta52hg00o51kkg

./scripts/deploy-storage.sh
```

Get the storage URL from the output:
```
yc serverless container get --name dialog-helper-storage --folder-id b1go18ta52hg00o51kkg --format json
```

## 4) Bootstrap admin key + system prompt
```
export YDB_DSN="grpcs://ydb.serverless.yandexcloud.net:2135/?database=/ru-central1/.../dialog-helper-db"

go run ./cmd/bootstrap --system-prompt-file docs/system_prompt.txt
```
Save the printed `ADMIN_MASTER_KEY`.

## 5) API Gateway
Allow the gateway service account to invoke containers:
```
yc serverless container add-access-binding \
  --name dialog-helper-storage \
  --folder-id b1go18ta52hg00o51kkg \
  --role serverless.containers.invoker \
  --subject serviceAccount:<service-account-id>
```

Create the gateway from `api/api-gateway.yaml` (uses `{{STORAGE_CONTAINER_ID}}`, `{{SERVICE_ACCOUNT_ID}}` variables):
```
export STORAGE_CONTAINER_ID=<storage-container-id>

python3 - <<'PY'
from pathlib import Path

content = Path("api/api-gateway.yaml").read_text()
content = content.replace("{{SERVICE_ACCOUNT_ID}}", "<service-account-id>")
content = content.replace("{{STORAGE_CONTAINER_ID}}", "<storage-container-id>")
Path("/tmp/api-gateway.yaml").write_text(content)
PY

yc serverless api-gateway create dialog-helper-gateway \
  --folder-id b1go18ta52hg00o51kkg \
  --spec /tmp/api-gateway.yaml
```

Get the gateway URL:
```
yc serverless api-gateway get dialog-helper-gateway --folder-id b1go18ta52hg00o51kkg --format json
```

Allow public access to the gateway (optional, if you want unauthenticated clients):
```
yc serverless api-gateway add-access-binding dialog-helper-gateway \
  --folder-id b1go18ta52hg00o51kkg \
  --role api-gateway.viewer \
  --subject system:allUsers
```

Optional hardening: remove public invocation from containers after the gateway is ready.
```
yc serverless container remove-access-binding \
  --name dialog-helper-storage \
  --folder-id b1go18ta52hg00o51kkg \
  --role serverless.containers.invoker \
  --subject system:allUsers
```
