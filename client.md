# Client API Guide

This document describes the requests sent by client applications (AAC clients, mobile/web apps) to the Dialog Helper API.

## Base URL
Use the API Gateway URL (example):
```
https://<gateway-id>.apigw.yandexcloud.net
```

## Authentication
All client requests must include:
```
X-API-KEY: <client-api-key>
```
API keys are issued by administrators via the admin service (out of scope for client apps).

## Health check
```
GET /healthz
```
Response:
```
{"status":"ok"}
```

## Inference
```
POST /v1/infer
Content-Type: multipart/form-data
```

### Multipart fields
- `payload` (required): JSON string with request parameters.
- `audio` (optional): audio file for the message marked with `"audio": true`.

If `audio` is omitted, include the interlocutor's text in the `messages` array.

### Payload schema (JSON)
```json
{
  "disabled_person_biography": "string",
  "messages": [
    {"role": "speaker", "content": "string"},
    {"role": "disabled_person", "content": "string"},
    {"role": "speaker", "audio": true}
  ],
  "language": "ru-RU",
  "persona": "string",
  "user_id": "string",
  "user_profile": "string",
  "dialog_id": "string",
  "step_id": "string",
  "time_window_sec": 600,
  "options": {
    "max_candidates": 5,
    "temperature": 0.3
  }
}
```

Notes:
- `disabled_person_biography` provides personal context to shape replies.
- `messages` is the dialog history; `role` can be `speaker` or `disabled_person`.
- Use `"audio": true` on a single `speaker` message when providing an audio file.
- `language` defaults to `ru-RU` if omitted.
- `time_window_sec` affects dialog hash stability for caching/idempotency.
- `options` allow per-request tuning of candidates count and temperature.

### Audio formats
Supported input formats:
- MP3 (`audio/mpeg`)
- WAV (`audio/wav`)
- OGG/OPUS (`audio/ogg`)

Default max size: 8 MB (configurable on the server).

### Example (audio + dialog)
```bash
curl -sS \
  -H "X-API-KEY: <client-api-key>" \
  -F 'payload={"disabled_person_biography":"Вежливый взрослый пользователь.","messages":[{"role":"speaker","content":"Здравствуйте!"},{"role":"disabled_person","content":"Здравствуйте, чем помочь?"},{"role":"speaker","audio":true}],"user_id":"user-1","dialog_id":"dialog-1","step_id":"1","language":"ru-RU"}' \
  -F "audio=@/path/to/audio.mp3;type=audio/mpeg" \
  https://<gateway-id>.apigw.yandexcloud.net/v1/infer
```

### Example (text-only)
```bash
curl -sS \
  -H "X-API-KEY: <client-api-key>" \
  -F 'payload={"disabled_person_biography":"Вежливый взрослый пользователь.","messages":[{"role":"speaker","content":"Здравствуйте, как дела?"}],"user_id":"user-1","dialog_id":"dialog-1","step_id":"2","language":"ru-RU"}' \
  https://<gateway-id>.apigw.yandexcloud.net/v1/infer
```

### Response
```json
{
  "transcript": "string",
  "response": [
    "string"
  ]
}
```

`transcript` is included only when audio was provided.

### Error responses
Common error codes:
- `missing_api_key` / `invalid_api_key`
- `invalid_request` (missing messages, invalid roles, or audio mismatch)
- `stt_failed` (STT provider error)
- `llm_failed` (LLM provider error)

## Idempotency and caching
The server computes an internal dialog hash from API key, user data, biography, messages, audio, and time window.
If a request repeats with the same inputs in the same window, a cached response may be returned.

## Client-side recommendations
- Always send `dialog_id` and `step_id` for stable cache behavior.
- If audio upload is slow, fall back to text-only by sending `messages` with `content`.
