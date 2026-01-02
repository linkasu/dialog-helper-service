# API

OpenAPI spec: `api/openapi.yaml`.

## POST /v1/infer
Multipart request with audio and JSON payload.

### Headers
- `X-API-KEY: <client key>`

### Multipart fields
- `audio`: audio file (MP3/WAV/OGG)
- `payload`: JSON-encoded `InferPayload`

### Payload example
```
{
  "prompt": "We are at a cafe. Suggest short replies.",
  "language": "ru-RU",
  "persona": "Adult user, polite tone",
  "user_id": "user-123",
  "dialog_id": "dlg-1",
  "step_id": "1",
  "options": {
    "max_candidates": 5,
    "temperature": 0.3
  }
}
```

### Response example
```
{
  "request_id": "...",
  "dialog_hash": "...",
  "user_hash": "...",
  "transcript": {
    "text": "Hello, how are you?",
    "language": "ru-RU"
  },
  "candidates": [
    {"text": "I am fine, thanks.", "rank": 1, "length": 21, "language": "ru-RU"},
    {"text": "Not great today.", "rank": 2, "length": 15, "language": "ru-RU"}
  ],
  "meta": {
    "cache_hit": false,
    "stt_latency_ms": 120,
    "llm_latency_ms": 240,
    "system_prompt_version": "20250101010101"
  }
}
```

## POST /v1/admin/keys
Creates an API key.

Headers:
- `X-ADMIN-KEY: <admin master key>`

Body:
```
{
  "client_id": "client-1",
  "scopes": ["infer"],
  "rate_limit_per_minute": 60,
  "status": "active"
}
```

## GET/POST /v1/admin/system-prompt
Get or update the system prompt. Requires `X-ADMIN-KEY`.
