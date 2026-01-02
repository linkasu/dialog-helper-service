# Overview

This repository implements the API platform described in the C4 draft: a stateless inference API that accepts dialog messages (text or audio) and returns a transcript (if audio) plus reply options. The platform stores only minimal, hashed data in YDB.

## Scope
- Accept multipart requests with dialog messages and optional audio.
- Perform STT using Yandex SpeechKit.
- Build a system prompt + biography + dialog messages and call YandexGPT.
- Return transcript (if audio) and reply options (user selects).
- Store minimal cache results and hashed metadata in YDB (optional).

## Components
- API Gateway (Yandex API Gateway)
- Inference service (serverless container)
- Storage service (serverless container)
- YDB (serverless)
- Yandex SpeechKit STT
- Yandex LLM (foundation models)

## Data flow
1. Client sends multipart request to `/v1/infer` (messages + optional audio).
2. Inference verifies `X-API-KEY` via Storage.
3. Inference computes hashes and checks cache in Storage.
4. Inference calls STT for transcript when audio is provided.
5. Inference calls LLM for suggestions.
6. Inference returns response and asynchronously persists minimal result.

## Stateless behavior
- No durable state inside containers.
- Short-lived caches only (in-memory, TTL).
- Persisted data is stored in YDB only.
