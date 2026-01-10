# API

OpenAPI spec: `api/openapi.yaml`.

## GET /client.md
Returns the public client integration guide (Markdown).

## POST /v1/infer
Multipart request with dialog messages and optional audio.

### Headers
- `X-API-KEY: <client key>`

### Multipart fields
- `audio`: audio file (MP3/WAV/OGG/WEBM) for the message marked with `"audio": true`
- `payload`: JSON-encoded `InferPayload`

### Payload example
```
{
  "disabled_person_biography": "Adult user, polite tone",
  "messages": [
    {"role": "speaker", "content": "привет"},
    {"role": "disabled_person", "content": "привет, как я могу помочь?"},
    {"role": "speaker", "audio": true}
  ],
  "language": "ru-RU",
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
  "transcript": "привет",
  "response": [
    "Я хорошо, спасибо!",
    "Не очень, но держусь."
  ]
}
```

`transcript` is included only when audio was provided.

## POST /v1/admin/keys
Creates an API key.

Headers:
- `X-ADMIN-KEY: <admin master key>`
- `X-ADMIN-SESSION: <otp session token>` (OTP login)

Body:
```
{
  "client_id": "client-1",
  "scopes": ["infer"],
  "rate_limit_per_minute": 60,
  "status": "active"
}
```

## GET /v1/admin/keys
List API keys.

Headers:
- `X-ADMIN-KEY` or `X-ADMIN-SESSION`

Query params:
- `limit` (optional)
- `include_revoked` (optional)

## DELETE /v1/admin/keys/{api_key_hash}
Revokes an API key.

Headers:
- `X-ADMIN-KEY` or `X-ADMIN-SESSION`

## GET/POST /v1/admin/system-prompt
Get or update the system prompt. Requires `X-ADMIN-KEY` or `X-ADMIN-SESSION`.

## POST /v1/admin/otp/send
Send OTP to Telegram (admin login).

## POST /v1/admin/otp/verify
Verify OTP and get a session token.

Body:
```
{"code": "123456"}
```

Response:
```
{"session_token": "...", "expires_at": "2026-01-01T00:00:00Z"}
```

## GET /v1/admin/stats
Usage stats (requests + active users).

Query params:
- `window` (duration, e.g. `24h`)
- `since` (RFC3339)

## GET /v1/admin/logs
Runtime logs from Yandex Cloud Logging (if configured).

Query params:
- `limit` (default 100)
- `window` / `since`

Response (example):
```
{
  "items": [
    {
      "timestamp": "2026-01-01T12:00:00Z",
      "level": "INFO",
      "message": "inference service listening on :8080",
      "resource_type": "serverless.container",
      "resource_id": "b25...",
      "stream_name": "stdout"
    }
  ]
}
```
