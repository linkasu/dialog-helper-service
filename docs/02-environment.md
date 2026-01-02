# Environment setup

## Required tools
- Go 1.24+ (aligned with `go.mod`).
- Docker with buildx.
- `yc` CLI (`yc init`).
- Git.

## Global setup
```
yc config set folder-id b1go18ta52hg00o51kkg
```

## Environment variables

### Inference service
- `PORT` (default: `8080`)
- `STORAGE_URL` (required)
- `INTERNAL_AUTH_TOKEN` (required)
- `YC_FOLDER_ID` (required)
- `YC_GPT_MODEL_URI` (optional)
- `YC_STT_MODEL` (default: `general`)
- `DEFAULT_LANGUAGE` (default: `ru-RU`)
- `SYSTEM_PROMPT_FALLBACK` (optional)
- `CONFIG_CACHE_TTL` (default: `1m`)
- `MAX_AUDIO_BYTES` (default: `8388608`)
- `DEFAULT_MAX_CANDIDATES` (default: `5`)
- `DEFAULT_TEMPERATURE` (default: `0.3`)
- `DEFAULT_MAX_TOKENS` (default: `300`)
- `STT_POLL_INTERVAL` (default: `200ms`)
- `STT_POLL_TIMEOUT` (default: `10s`)
- `PERSIST_RESULTS` (default: `true`)
- `STT_PROVIDER` (`yandex` or `mock`)
- `LLM_PROVIDER` (`yandex` or `mock`)
- `DEFAULT_TIME_WINDOW_SEC` (default: `600`)
- `YC_IAM_TOKEN` (optional for local; otherwise metadata is used)

### Storage service
- `PORT` (default: `8081`)
- `STORAGE_BACKEND` (`ydb` or `memory`)
- `YDB_DSN` (required for `ydb` backend)
- `INTERNAL_AUTH_TOKEN` (required)
- `ADMIN_MASTER_KEY` (optional for local memory backend)
- `YC_IAM_TOKEN` (optional for local; otherwise metadata is used)

### Bootstrap tool
- `YDB_DSN` (required)
- `YC_IAM_TOKEN` (optional for local)
