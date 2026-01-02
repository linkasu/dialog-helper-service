# Yandex Cloud setup

All resources are created in folder `b1go18ta52hg00o51kkg`, region `ru-central1`.

## 1) Service account
```
yc iam service-account create \
  --name dialog-helper-sa-2025 \
  --folder-id b1go18ta52hg00o51kkg
```

Capture the service account ID:
```
yc iam service-account get dialog-helper-sa-2025 --folder-id b1go18ta52hg00o51kkg --format json
```

### Grant roles
```
yc resource-manager folder add-access-binding \
  --id b1go18ta52hg00o51kkg \
  --role serverless.containers.editor \
  --service-account-name dialog-helper-sa-2025

yc resource-manager folder add-access-binding \
  --id b1go18ta52hg00o51kkg \
  --role api-gateway.admin \
  --service-account-name dialog-helper-sa-2025

yc resource-manager folder add-access-binding \
  --id b1go18ta52hg00o51kkg \
  --role ydb.editor \
  --service-account-name dialog-helper-sa-2025

yc resource-manager folder add-access-binding \
  --id b1go18ta52hg00o51kkg \
  --role lockbox.viewer \
  --service-account-name dialog-helper-sa-2025

yc resource-manager folder add-access-binding \
  --id b1go18ta52hg00o51kkg \
  --role ai.speechkit-stt.user \
  --service-account-name dialog-helper-sa-2025

yc resource-manager folder add-access-binding \
  --id b1go18ta52hg00o51kkg \
  --role ai.languageModels.user \
  --service-account-name dialog-helper-sa-2025
```

## 2) Container registry
```
yc container registry create \
  --name dialog-helper-registry \
  --folder-id b1go18ta52hg00o51kkg
```

Store registry ID:
```
yc container registry get dialog-helper-registry --format json
```

Configure docker auth:
```
yc container registry configure-docker
```

## 3) YDB serverless database
```
yc ydb database create dialog-helper-db \
  --folder-id b1go18ta52hg00o51kkg \
  --serverless
```

Get endpoint + database path:
```
yc ydb database get dialog-helper-db --format json
```

Export DSN:
```
export YDB_DSN="grpcs://ydb.serverless.yandexcloud.net:2135/?database=/ru-central1/.../dialog-helper-db"
```

## 4) Apply schema
Follow `docs/04-db-schema.md`.

## 5) API Gateway (optional)
After containers are deployed, update `api/api-gateway.yaml` with service URLs and create the gateway:
```
yc api-gateway create \
  --name dialog-helper-api \
  --spec api/api-gateway.yaml \
  --folder-id b1go18ta52hg00o51kkg
```

