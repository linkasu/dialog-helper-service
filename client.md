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
- `audio` (optional): audio file with the interlocutor's speech.

If `audio` is omitted, provide `transcript` in the payload.

### Payload schema (JSON)
```json
{
  "prompt": "string (required)",
  "language": "ru-RU",
  "persona": "string",
  "user_id": "string",
  "user_profile": "string",
  "dialog_id": "string",
  "step_id": "string",
  "time_window_sec": 600,
  "transcript": "string",
  "options": {
    "max_candidates": 5,
    "temperature": 0.3
  }
}
```

Notes:
- `prompt` is the client-provided context. It is always merged with the system prompt.
- `language` defaults to `ru-RU` if omitted.
- `time_window_sec` affects dialog hash stability for caching/idempotency.
- `options` allow per-request tuning of candidates count and temperature.

### Audio formats
Supported input formats:
- MP3 (`audio/mpeg`)
- WAV (`audio/wav`)
- OGG/OPUS (`audio/ogg`)

Default max size: 8 MB (configurable on the server).

### Example (audio + prompt)
```bash
curl -sS \
  -H "X-API-KEY: <client-api-key>" \
  -F 'payload={"prompt":"Подтверди встречу на следующей неделе.","user_id":"user-1","dialog_id":"dialog-1","step_id":"1","language":"ru-RU"}' \
  -F "audio=@/path/to/audio.mp3;type=audio/mpeg" \
  https://<gateway-id>.apigw.yandexcloud.net/v1/infer
```

### Example (text-only)
```bash
curl -sS \
  -H "X-API-KEY: <client-api-key>" \
  -F 'payload={"prompt":"Сформулируй ответы.","transcript":"Здравствуйте, как дела?","user_id":"user-1","dialog_id":"dialog-1","step_id":"2","language":"ru-RU"}' \
  https://<gateway-id>.apigw.yandexcloud.net/v1/infer
```

### Response
```json
{
  "request_id": "string",
  "dialog_hash": "string",
  "user_hash": "string",
  "transcript": {
    "text": "string",
    "language": "ru-RU"
  },
  "candidates": [
    {
      "text": "string",
      "rank": 1,
      "length": 20,
      "language": "ru-RU"
    }
  ],
  "errors": [
    { "code": "llm_failed", "message": "llm request failed" }
  ],
  "meta": {
    "cache_hit": false,
    "stt_latency_ms": 1200,
    "llm_latency_ms": 650,
    "system_prompt_version": "20260102160445"
  }
}
```

### Error responses
Common error codes:
- `missing_api_key` / `invalid_api_key`
- `invalid_request` / `missing_prompt`
- `stt_failed` (STT provider error)
- `llm_failed` (LLM provider error)

## Idempotency and caching
The server computes a `dialog_hash` from API key, user data, prompt, audio/transcript, and time window.
If a request repeats with the same inputs in the same window, the cached response is returned with `meta.cache_hit=true`.

## Client-side recommendations
- Store `dialog_hash` and `user_hash` for correlation in your client app.
- Always send `dialog_id` and `step_id` for stable cache behavior.
- If audio upload is slow, fall back to text-only using `transcript`.
