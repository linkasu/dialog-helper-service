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
- `MAX_MESSAGES` (default: `64`)
- `MAX_MESSAGE_CHARS` (default: `2000`)
- `MAX_BIO_CHARS` (default: `2000`)
- `MAX_PERSONA_CHARS` (default: `256`)
- `MAX_USER_ID_CHARS` (default: `128`)
- `MAX_USER_PROFILE_CHARS` (default: `2000`)
- `MAX_DIALOG_ID_CHARS` (default: `128`)
- `MAX_STEP_ID_CHARS` (default: `64`)
- `MAX_LANGUAGE_CHARS` (default: `32`)
- `DEFAULT_MAX_CANDIDATES` (default: `5`)
- `MAX_CANDIDATES` (default: `10`)
- `DEFAULT_TEMPERATURE` (default: `0.3`)
- `MAX_TEMPERATURE` (default: `1.0`)
- `DEFAULT_MAX_TOKENS` (default: `300`)
- `STT_POLL_INTERVAL` (default: `200ms`)
- `STT_POLL_TIMEOUT` (default: `10s`)
- `PERSIST_RESULTS` (default: `true`)
- `STT_PROVIDER` (`yandex` or `mock`)
- `LLM_PROVIDER` (`yandex` or `mock`)
- `DEFAULT_TIME_WINDOW_SEC` (default: `600`)
- `YC_IAM_TOKEN` (optional for local; otherwise metadata is used)
- `CLIENT_DOC_PATH` (optional; path to `client.md` to serve at `/client.md`)

### Storage service
- `PORT` (default: `8081`)
- `STORAGE_BACKEND` (`ydb` or `memory`)
- `YDB_DSN` (required for `ydb` backend)
- `INTERNAL_AUTH_TOKEN` (required)
- `ADMIN_MASTER_KEY` (optional for local memory backend)
- `TELEGRAM_BOT_TOKEN` (optional; required for OTP login)
- `TELEGRAM_ADMIN_ID` (optional; required for OTP login)
- `ADMIN_OTP_TTL` (default: `5m`)
- `ADMIN_SESSION_TTL` (default: `12h`)
- `YC_LOG_GROUP_ID` (optional; required to fetch runtime logs)
- `YC_LOG_RESOURCE_TYPES` (optional, comma-separated)
- `YC_LOG_RESOURCE_IDS` (optional, comma-separated)
- `YC_LOG_STREAMS` (optional, comma-separated)
- `YC_LOG_LEVELS` (optional, comma-separated: info,warn,error)
- `YC_LOG_FILTER` (optional)
- `YC_LOG_ENDPOINT` (default: `logging.api.cloud.yandex.net:443`)
- `YC_LOG_MAX_BYTES` (optional)
- `YC_IAM_TOKEN` (optional for local; otherwise metadata is used)

### Bootstrap tool
- `YDB_DSN` (required)
- `YC_IAM_TOKEN` (optional for local)
