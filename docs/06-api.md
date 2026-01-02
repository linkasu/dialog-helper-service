# API

OpenAPI spec: `api/openapi.yaml`.

## GET /client.md
Returns the public client integration guide (Markdown).

## POST /v1/infer
Multipart request with dialog messages and optional audio.

### Headers
- `X-API-KEY: <client key>`

### Multipart fields
- `audio`: audio file (MP3/WAV/OGG) for the message marked with `"audio": true`
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
